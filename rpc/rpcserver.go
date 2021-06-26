package rpc

// NOTE: PoST RPC server is currently disabled.

/*

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/shared"
	"golang.org/x/net/context"
	"sync"
)

var (
	ErrAlreadyInitializing = errors.New("already initializing")
)

type (
	Config = config.Config
	Logger = shared.Logger
)

// rpcServer is a gRPC, RPC front end to the POST server.
type rpcServer struct {
	cfg    *Config
	logger Logger
	signal *shared.Signal

	initializing map[string]bool

	sync.Mutex
}

// A compile time check to ensure that rpcServer fully implements PostServer.
var _ api.PostServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(s *shared.Signal, cfg *Config, logger Logger) (*rpcServer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &rpcServer{
		cfg:          cfg,
		logger:       logger,
		signal:       s,
		initializing: make(map[string]bool),
	}, nil
}

func (r *rpcServer) Initialize(ctx context.Context, in *api.InitializeRequest) (*api.InitializeResponse, error) {
	if err := r.addInitializing(in.Id); err != nil {
		return nil, err
	}
	defer r.removeInitializing(in.Id)

	init, err := initialization.NewInitializer(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}

	init.SetLogger(r.logger)
	if err := init.Initialize(initialization.CPUProviderID()); err != nil {
		return nil, err
	}

	return &api.InitializeResponse{}, nil
}

func (r *rpcServer) InitializeAsync(ctx context.Context, in *api.InitializeAsyncRequest) (*api.InitializeAsyncResponse, error) {
	init, err := initialization.NewInitializer(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}
	if err := init.VerifyNotCompleted(); err != nil {
		return nil, err
	}

	if err := r.addInitializing(in.Id); err != nil {
		return nil, err
	}

	go func() {
		defer r.removeInitializing(in.Id)

		init, _ := initialization.NewInitializer(r.cfg, in.Id)
		init.SetLogger(r.logger)
		if err := init.Initialize(initialization.CPUProviderID()); err != nil {
			r.logger.Error("initialization failure: %v", err)
			return
		}
	}()

	return &api.InitializeAsyncResponse{}, nil
}

func (r *rpcServer) Execute(ctx context.Context, in *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	prover, err := proving.NewProver(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}
	prover.SetLogger(r.logger)
	proof, proofMetadata, err := prover.GenerateProof(in.Challenge)
	if err != nil {
		return nil, err
	}

	err = shared.PersistProof(r.cfg.DataDir, proof, proofMetadata)
	if err != nil {
		return nil, err
	}

	return &api.ExecuteResponse{
		Proof: &api.Proof{
			Nonce:   proof.Nonce,
			Indices: proof.Indices,
		},
		ProofMetadata: &api.ProofMetadata{
			Id:        proofMetadata.ID,
			Challenge: proofMetadata.Challenge,
			NumLabels: proofMetadata.NumLabels,
			LabelSize: uint32(proofMetadata.LabelSize),
			K1:        uint32(proofMetadata.K1),
			K2:        uint32(proofMetadata.K2),
		},
	}, nil
}

func (r *rpcServer) ExecuteAsync(ctx context.Context, in *api.ExecuteAsyncRequest) (*api.ExecuteAsyncResponse, error) {
	init, err := initialization.NewInitializer(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}
	if err := init.VerifyCompleted(); err != nil {
		return nil, err
	}

	go func() {
		prover, _ := proving.NewProver(r.cfg, in.Id)
		prover.SetLogger(r.logger)
		proof, proofMetadata, err := prover.GenerateProof(in.Challenge)
		if err != nil {
			r.logger.Error("execution failure: %v", err)
			return
		}

		err = shared.PersistProof(r.cfg.DataDir, proof, proofMetadata)
		if err != nil {
			r.logger.Error("proof persisting failure: %v", err)
			return
		}
	}()

	return &api.ExecuteAsyncResponse{}, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *api.GetProofRequest) (*api.GetProofResponse, error) {
	proof, proofMetadata, err := shared.FetchProof(r.cfg.DataDir, in.Challenge)
	if err != nil {
		return nil, err
	}

	return &api.GetProofResponse{
		Proof: &api.Proof{
			Nonce:   proof.Nonce,
			Indices: proof.Indices,
		},
		ProofMetadata: &api.ProofMetadata{
			Challenge: proofMetadata.Challenge,
			NumLabels: proofMetadata.NumLabels,
			LabelSize: uint32(proofMetadata.LabelSize),
			K1:        uint32(proofMetadata.K1),
			K2:        uint32(proofMetadata.K2),
		},
	}, nil
}

func (r *rpcServer) Reset(ctx context.Context, in *api.ResetRequest) (*api.ResetResponse, error) {
	init, err := initialization.NewInitializer(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}
	init.SetLogger(r.logger)
	err = init.Reset()
	if err != nil {
		return nil, err
	}

	return &api.ResetResponse{}, nil
}

func (r *rpcServer) GetState(ctx context.Context, in *api.GetStateRequest) (*api.GetStateResponse, error) {
	r.Lock()
	idHex := hex.EncodeToString(in.Id)
	exists := r.initializing[idHex]
	r.Unlock()

	if exists {
		return &api.GetStateResponse{State: api.GetStateResponse_Initializing}, nil
	}

	init, err := initialization.NewInitializer(r.cfg, in.Id)
	if err != nil {
		return nil, err
	}
	init.SetLogger(r.logger)

	numLabelsWritten, err := init.DiskNumLabelsWritten()
	if err != nil {
		return nil, err
	}
	var state api.GetStateResponse_State
	if numLabelsWritten == 0 {
		state = api.GetStateResponse_NotStarted // zero value.
	} else if numLabelsWritten < r.cfg.NumLabels {
		state = api.GetStateResponse_Stopped
	} else {
		state = api.GetStateResponse_Completed
	}
	return &api.GetStateResponse{State: state, NumLabelsWritten: numLabelsWritten}, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	out := &api.GetInfoResponse{
		Version: shared.Version(),
		Config: &api.Config{
			Datadir:   r.cfg.DataDir,
			NumLabels: uint64(r.cfg.NumLabels),
			LabelSize: uint32(r.cfg.LabelSize),
			K1:        uint32(r.cfg.K1),
			K2:        uint32(r.cfg.K2),
			NumFiles:  uint32(r.cfg.NumFiles),
		},
	}

	return out, nil
}

func (r *rpcServer) Shutdown(context.Context, *api.ShutdownRequest) (*api.ShutdownResponse, error) {
	r.signal.RequestShutdown()
	return &api.ShutdownResponse{}, nil
}

func (r *rpcServer) addInitializing(id []byte) error {
	r.Lock()
	defer r.Unlock()

	idHex := hex.EncodeToString(id)
	if r.initializing[idHex] {
		return ErrAlreadyInitializing
	}
	r.initializing[idHex] = true
	return nil
}

func (r *rpcServer) removeInitializing(id []byte) {
	r.Lock()
	defer r.Unlock()

	idHex := hex.EncodeToString(id)
	delete(r.initializing, idHex)
}
*/
