package rpc

import (
	"encoding/hex"
	"github.com/spacemeshos/post/initialization"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/signal"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotInitialized = status.Error(codes.FailedPrecondition, "not initialized")
	ErrNoProofExists  = status.Error(codes.FailedPrecondition, "no computed proof exists")
)

type state struct {
	id     []byte
	dir    string
	proofs map[string]*proving.Proof
}

// rpcServer is a gRPC, RPC front end to the POST server.
type rpcServer struct {
	s       *signal.Signal
	params  *shared.Params
	datadir string
	lograte uint64
	state   *state
}

// A compile time check to ensure that rpcServer fully implements PostServer.
var _ api.PostServer = (*rpcServer)(nil)

// newRPCServer creates and returns a new instance of the rpcServer.
func NewRPCServer(s *signal.Signal, params *shared.Params, datadir string, lograte uint64) *rpcServer {
	return &rpcServer{
		s:       s,
		params:  params,
		datadir: datadir,
		lograte: lograte,
	}
}

func (r *rpcServer) Initialize(ctx context.Context, in *api.InitializeRequest) (*api.InitializeResponse, error) {
	dir := shared.GetDir(r.datadir, in.Id)
	proof, err := initialization.Initialize(in.Id, r.params.SpacePerUnit, r.params.NumOfProvenLabels, r.params.Difficulty, dir, r.lograte)
	if err != nil {
		return nil, err
	}

	r.state = new(state)
	r.state.id = in.Id
	r.state.dir = dir
	r.state.proofs = make(map[string]*proving.Proof)
	r.state.proofs[hex.EncodeToString(shared.ZeroChallenge)] = proof // The map key is an empty string ("").

	out := &api.InitializeResponse{Proof: &api.Proof{
		Id:           proof.Identity,
		Challenge:    proof.Challenge,
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}}

	return out, nil
}

func (r *rpcServer) Execute(ctx context.Context, in *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if r.state == nil {
		return nil, ErrNotInitialized
	}

	proof, err := proving.GenerateProof(r.state.id, in.Challenge, r.params.NumOfProvenLabels, r.params.Difficulty, r.state.dir)
	if err != nil {
		return nil, err
	}

	r.state.proofs[hex.EncodeToString(in.Challenge)] = proof

	out := &api.ExecuteResponse{Proof: &api.Proof{
		Id:           proof.Identity,
		Challenge:    proof.Challenge,
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}}

	return out, nil
}

func (r *rpcServer) GetProof(ctx context.Context, in *api.GetProofRequest) (*api.GetProofResponse, error) {
	if r.state == nil {
		return nil, ErrNotInitialized
	}

	proof, ok := r.state.proofs[hex.EncodeToString(in.Challenge)]
	if !ok {
		return nil, ErrNoProofExists
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
	if r.state == nil {
		return nil, status.Error(codes.FailedPrecondition, initialization.ErrIdNotInitialized.Error())
	}

	res, err := initialization.Reset(r.state.dir)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	r.state = nil
	out := &api.ResetResponse{
		DeletedDir:        res.DeletedDir,
		NumOfDeletedFiles: int32(res.NumOfDeletedFiles),
	}

	return out, nil
}

func (r *rpcServer) GetInfo(ctx context.Context, in *api.GetInfoRequest) (*api.GetInfoResponse, error) {
	out := &api.GetInfoResponse{
		Version: shared.Version(),
		Params: &api.Params{
			SpacePerUnit: int64(r.params.SpacePerUnit),
			Difficulty:   int32(r.params.Difficulty),
			T:            int32(r.params.NumOfProvenLabels),
			CacheLayer:   int32(r.params.LowestLayerToCacheDuringProofGeneration),
		},
		State: wireState(r.state),
	}

	return out, nil
}

func (r *rpcServer) Shutdown(context.Context, *api.ShutdownRequest) (*api.ShutdownResponse, error) {
	r.s.RequestShutdown()
	return &api.ShutdownResponse{}, nil
}

func wireState(state *state) *api.State {
	if state == nil {
		return nil
	}

	challenges := make([]string, 0)
	for challenge := range state.proofs {
		challenges = append(challenges, challenge)
	}

	return &api.State{
		Id:               state.id,
		Dir:              state.dir,
		ProvenChallenges: challenges,
	}
}
