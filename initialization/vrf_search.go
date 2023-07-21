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

	"go.uber.org/zap"

	"github.com/spacemeshos/post/internal/postrs"
	"github.com/spacemeshos/post/oracle"
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

	woReference, err := oracle.New(
		oracle.WithProviderID(CPUProviderID()),
		oracle.WithCommitment(metadata.CommitmentAtxId),
		oracle.WithVRFDifficulty(difficulty),
		oracle.WithScryptParams(initOpts.Scrypt),
		oracle.WithLogger(logger),
	)
	if err != nil {
		return 0, nil, err
	}
	defer woReference.Close()

	for _, file := range allFiles {
		info, err := file.Info()
		if err != nil {
			logger.Error("failed to get file info", zap.Error(err))
			continue
		}

		if !shared.IsInitFile(info) {
			continue
		}

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

		idx, l, err := searchForNonce(ctx, bufio.NewReader(file), difficulty, woReference)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to search for nonce: %w", err)
		}
		if l != nil {
			nonceValue = l
			nonce = firstLabelIndex + idx
			logger.Info("found nonce", zap.Uint64("index", nonce), zap.String("label", hex.EncodeToString(nonceValue)))
			difficulty = nonceValue // override difficulty to the new lowest label

		}
	}
	if nonceValue != nil {
		logger.Info("updating postdata_metadata.json with found nonce", zap.Uint64("nonce", nonce), zap.String("NonceValue", hex.EncodeToString(nonceValue)))
		metadata.Nonce = &nonce
		metadata.NonceValue = shared.NonceValue(nonceValue)
		SaveMetadata(initOpts.DataDir, metadata)
		return nonce, nonceValue, nil
	}
	return 0, nil, ErrNonceNotFound
}

// searchForNonce searches for a nonce in the given reader.
func searchForNonce(ctx context.Context, r io.Reader, difficulty []byte, oracle *oracle.WorkOracle) (nonce uint64, nonceValue []byte, err error) {
	labelBuf := make([]byte, postrs.LabelLength)
	labelIndex := uint64(0)
	for {
		select {
		case <-ctx.Done():
			return nonce, labelBuf, ctx.Err()
		default:
			// continue looking for a nonce
		}

		_, err := io.ReadFull(r, labelBuf)
		switch {
		case err == io.EOF:
			return nonce, nonceValue, nil
		case err == io.ErrUnexpectedEOF:
			return 0, nil, errors.New("unexpected end of file - file is truncated. please reinit it")
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
		labelIndex++
	}
}

// checkLabels checks if label is lower than difficulty.
// It will regenerate the whole 32B of the label if its most significant 16B are equal to difficulty
// in order to check the lower bytes.
// * index:      the index of the label (used to regenerate it)
// * label:		 16B most significant bytes of the label
// * difficulty: all 32b of the difficulty
// * wo:         the oracle used to regenerate the label.
func checkLabel(index uint64, label, difficulty []byte, wo *oracle.WorkOracle) (bool, error) {
	comp := bytes.Compare(label, difficulty)
	if comp < 0 {
		return true, nil
	} else if comp == 0 {
		// need to regenerate label to verify lower bytes
		res, err := wo.Position(index)
		if err != nil {
			return false, fmt.Errorf("failed to regenerate label: %w", err)
		}
		return bytes.Compare(res.Output, difficulty) < 0, nil
	}
	return false, nil
}
