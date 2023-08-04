package initialization

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"

	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/oracle"
	"github.com/spacemeshos/post/persistence"
	"github.com/spacemeshos/post/shared"
)

var ErrNonceNotFound = errors.New("nonce not found")

type searchForNonceOpts struct {
	logger            *zap.Logger
	powDifficultyFunc func(uint64) []byte
}

type searchForNonceOpt func(*searchForNonceOpts)

func SearchWithLogger(logger *zap.Logger) searchForNonceOpt {
	return func(opts *searchForNonceOpts) {
		opts.logger = logger
	}
}

func searchWithPowDifficultyFunc(powDifficultyFunc func(uint64) []byte) searchForNonceOpt {
	return func(opts *searchForNonceOpts) {
		opts.powDifficultyFunc = powDifficultyFunc
	}
}

// SearchForNonce is searches for a nonce in the already initialized data.
// Will return ErrNonceNotFound if no nonce was found.
// Otherwise, it will return the nonce the 16B of label it points to.
func SearchForNonce(ctx context.Context, cfg Config, initOpts InitOpts, opts ...searchForNonceOpt) (nonce uint64, nonceValue []byte, err error) {
	options := searchForNonceOpts{
		logger:            zap.NewNop(),
		powDifficultyFunc: shared.PowDifficulty,
	}
	for _, opt := range opts {
		opt(&options)
	}
	logger := options.logger

	metadata, err := LoadMetadata(initOpts.DataDir)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	difficulty := options.powDifficultyFunc(metadata.LabelsPerUnit * uint64(metadata.NumUnits))
	logger.Info("searching for lowes nonce",
		zap.String("difficulty", hex.EncodeToString(difficulty)),
		zap.String("datadir", initOpts.DataDir),
	)

	allFiles, err := os.ReadDir(initOpts.DataDir)
	if err != nil {
		return 0, nil, fmt.Errorf("couldn't open the data directory: %w", err)
	}

	cpuProviderID := CPUProviderID()
	woReference, err := oracle.New(
		oracle.WithProviderID(&cpuProviderID),
		oracle.WithCommitment(metadata.CommitmentAtxId),
		oracle.WithVRFDifficulty(difficulty),
		oracle.WithScryptParams(initOpts.Scrypt),
		oracle.WithLogger(logger),
	)
	if err != nil {
		return 0, nil, err
	}
	defer woReference.Close()

	// Filter and sort init files.
	var initFiles []os.FileInfo
	for _, file := range allFiles {
		info, err := file.Info()
		if err != nil {
			logger.Error("failed to get file info", zap.Error(err))
			continue
		}
		if shared.IsInitFile(info) {
			initFiles = append(initFiles, info)
		}
	}
	sort.Sort(persistence.NumericalSorter(initFiles))

	for _, file := range initFiles {
		fileIndex, err := shared.ParseFileIndex(file.Name())
		if err != nil {
			logger.Panic("failed to parse file index", zap.Error(err))
		}

		if fileIndex < initOpts.FromFileIdx || (initOpts.ToFileIdx != nil && fileIndex > *initOpts.ToFileIdx) {
			logger.Debug("skipping file", zap.String("file", file.Name()))
			continue
		}

		firstLabelIndex := uint64(fileIndex) * metadata.MaxFileSize / postrs.LabelLength
		logger.Info("looking for VRF nonce in file", zap.String("file", file.Name()))

		file, err := os.Open(filepath.Join(initOpts.DataDir, file.Name()))
		if err != nil {
			logger.Error("failed to open file", zap.Error(err))
			continue
		}
		defer file.Close()

		idx, label, err := searchForNonce(ctx, bufio.NewReader(file), difficulty, woReference)
		if label != nil {
			nonceValue = label
			nonce = firstLabelIndex + idx
			if err := persistNonce(nonce, nonceValue, metadata, initOpts.DataDir, logger); err != nil {
				return nonce, nonceValue, err
			}
			difficulty = nonceValue // override difficulty to the new lowest label
		}

		switch {
		case errors.Is(err, context.Canceled):
			logger.Info("search for nonce interrupted", zap.Uint64("nonce", nonce), zap.String("nonceValue", hex.EncodeToString(nonceValue)))
			return nonce, nonceValue, err
		case err != nil:
			return 0, nil, fmt.Errorf("failed to search for nonce: %w", err)
		}
	}
	if nonceValue != nil {
		return nonce, nonceValue, nil
	}
	return 0, nil, ErrNonceNotFound
}

func persistNonce(nonce uint64, label []byte, metadata *shared.PostMetadata, datadir string, logger *zap.Logger) error {
	logger.Info("found nonce: updating postdata_metadata.json", zap.Uint64("nonce", nonce), zap.String("NonceValue", hex.EncodeToString(label)))
	metadata.Nonce = &nonce
	metadata.NonceValue = shared.NonceValue(label)
	if err := SaveMetadata(datadir, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	return nil
}

// searchForNonce searches for a nonce in the given reader.
func searchForNonce(ctx context.Context, r io.Reader, difficulty []byte, oracle *oracle.WorkOracle) (nonce uint64, nonceValue []byte, err error) {
	labelBuf := make([]byte, postrs.LabelLength)

	for labelIndex := uint64(0); ; labelIndex++ {
		select {
		case <-ctx.Done():
			return nonce, nonceValue, ctx.Err()
		default:
			// continue looking for a nonce
		}

		_, err := io.ReadFull(r, labelBuf)
		switch {
		case err == io.EOF:
			return nonce, nonceValue, nil
		case err == io.ErrUnexpectedEOF:
			return 0, nil, fmt.Errorf("file appears truncated. please re-init it: %w", err)
		}

		ok, err := checkLabel(labelIndex, labelBuf, difficulty, oracle)
		if err != nil {
			return 0, nil, fmt.Errorf("checking label: %w", err)
		}
		if ok {
			nonce = labelIndex
			nonceValue = append(nonceValue[:0], labelBuf...)
			difficulty = nonceValue // override difficulty to the new lowest label
		}
	}
}

// checkLabels checks if label is lower than difficulty.
// It will regenerate the whole 32B of the label if its most significant 16B are equal to difficulty
// in order to check the lower bytes.
// * index:      the index of the label (used to regenerate it)
// * label:      16B most significant bytes of the label
// * difficulty: 32b of the difficulty
// * wo:         the oracle used to regenerate the label.
func checkLabel(index uint64, label, difficulty []byte, wo *oracle.WorkOracle) (bool, error) {
	comp := bytes.Compare(label, difficulty)
	switch {
	case comp < 0:
		return true, nil
	case comp == 0:
		// need to regenerate label to verify lower bytes
		res, err := wo.Position(index)
		if err != nil {
			return false, fmt.Errorf("failed to regenerate label: %w", err)
		}
		return bytes.Compare(res.Output, difficulty) < 0, nil
	default: // comp > 0
		return false, nil
	}
}
