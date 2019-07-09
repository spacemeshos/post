package rpc

import (
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/shared"
	"golang.org/x/net/context"
)

var (
	VerifyInitialized    = shared.VerifyInitialized
	VerifyNotInitialized = shared.VerifyNotInitialized
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
	i      *initialization.Initializer
	p      *proving.Prover
}

// A compile time check to ensure that rpcServer fully implements PostServer.
var _ api.PostServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(s *shared.Signal, cfg *Config, logger Logger) (*rpcServer, error) {
	return &rpcServer{
		cfg:    cfg,
		logger: logger,
		signal: s,
		i:      initialization.NewInitializer(cfg, logger),
		p:      proving.NewProver(cfg, logger),
	}, nil
}

func (r *rpcServer) Initialize(ctx context.Context, in *api.InitializeRequest) (*api.InitializeResponse, error) {
	proof, err := r.i.Initialize(in.Id)
	if err != nil {
		return nil, err
	}

	err = shared.PersistProof(r.cfg.DataDir, proof)
	if err != nil {
		return nil, err
	}

	out := &api.InitializeResponse{Proof: &api.Proof{
		Id:           proof.Identity,
		Challenge:    proof.Challenge,
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}}

	return out, nil
}

func (r *rpcServer) InitializeAsync(ctx context.Context, in *api.InitializeAsyncRequest) (*api.InitializeAsyncResponse, error) {
	if err := VerifyNotInitialized(r.cfg, in.Id); err != nil {
		return nil, err
	}

	go func() {
		proof, err := r.i.Initialize(in.Id)
		if err != nil {
			r.logger.Error("initialization failure: %v", err)
			return
		}

		err = shared.PersistProof(r.cfg.DataDir, proof)
		if err != nil {
			r.logger.Error("proof persisting failure: %v", err)
			return
		}
	}()

	return &api.InitializeAsyncResponse{}, nil
}

func (r *rpcServer) Execute(ctx context.Context, in *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	proof, err := r.p.GenerateProof(in.Id, in.Challenge)
	if err != nil {
		return nil, err
	}

	err = shared.PersistProof(r.cfg.DataDir, proof)
	if err != nil {
		return nil, err
	}

	out := &api.ExecuteResponse{Proof: &api.Proof{
		Id:           proof.Identity,
		Challenge:    proof.Challenge,
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}}

	return out, nil
}

func (r *rpcServer) ExecuteAsync(ctx context.Context, in *api.ExecuteAsyncRequest) (*api.ExecuteAsyncResponse, error) {
	if err := VerifyInitialized(r.cfg, in.Id); err != nil {
		return nil, err
	}

	go func() {
		proof, err := r.p.GenerateProof(in.Id, in.Challenge)
		if err != nil {
			r.logger.Error("execution failure: %v", err)
			return
		}

		err = shared.PersistProof(r.cfg.DataDir, proof)
		if err != nil {
			r.logger.Error("proof persisting failure: %v", err)
			return
		}
	}()

	return &api.ExecuteAsyncResponse{}, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *api.GetProofRequest) (*api.GetProofResponse, error) {
	if err := VerifyInitialized(r.cfg, in.Id); err != nil {
		return nil, err
	}

	proof, err := shared.FetchProof(r.cfg.DataDir, in.Id, in.Challenge)
	if err != nil {
		return nil, err
	}

	out := &api.GetProofResponse{Proof: &api.Proof{
		Id:           proof.Identity,
		Challenge:    proof.Challenge,
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}}

	return out, nil
}

func (r *rpcServer) Reset(ctx context.Context, in *api.ResetRequest) (*api.ResetResponse, error) {
	err := r.i.Reset(in.Id)
	if err != nil {
		return nil, err
	}

	return &api.ResetResponse{}, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	out := &api.GetInfoResponse{
		Version: shared.Version(),
		Config: &api.Config{
			Datadir:      r.cfg.DataDir,
			SpacePerUnit: int64(r.cfg.SpacePerUnit),
			Difficulty:   int32(r.cfg.Difficulty),
			Labels:       int32(r.cfg.NumProvenLabels),
			CacheLayer:   int32(r.cfg.LowestLayerToCacheDuringProofGeneration),
		},
	}

	return out, nil
}

func (r *rpcServer) Shutdown(context.Context, *api.ShutdownRequest) (*api.ShutdownResponse, error) {
	r.signal.RequestShutdown()
	return &api.ShutdownResponse{}, nil
}
