package main

import (
	"context"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/integration"
	"github.com/spacemeshos/post/proving"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/post/validation"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// harnessTestCase represents a test-case which utilizes an instance
// of the Harness to exercise functionality.
type harnessTestCase struct {
	name string
	test func(h *integration.Harness, assert *require.Assertions, ctx context.Context, cfg *config.Config)
}

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "initialize", test: testInitialize},
	{name: "initialize parallel", test: testInitializeParallel},
}

var (
	cfg = config.DefaultConfig()
	id  = []byte("deadbeef")
)

func TestHarness(t *testing.T) {
	assert := require.New(t)

	cfg := *cfg
	cfg.SpacePerUnit = 1 << 25
	cfg.NumFiles = 4

	h := newHarness(assert, &cfg)
	defer func() {
		err := h.TearDown(true)
		assert.NoError(err, "failed to tear down harness")
	}()

	for _, testCase := range testCases {
		success := t.Run(testCase.name, func(t1 *testing.T) {
			ctx, _ := context.WithTimeout(context.Background(), time.Duration(30*time.Second))
			testCase.test(h, assert, ctx, &cfg)
		})

		if !success {
			break
		}
	}
}

func testInfo(h *integration.Harness, assert *require.Assertions, ctx context.Context, cfg *config.Config) {
	info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
	assert.Equal(info.Version, shared.Version())
	assert.Equal(uint64(info.Config.SpacePerUnit), cfg.SpacePerUnit)
	assert.Equal(uint(info.Config.Difficulty), cfg.Difficulty)
	assert.Equal(uint(info.Config.Labels), cfg.NumProvenLabels)
	assert.Equal(uint(info.Config.CacheLayer), cfg.LowestLayerToCacheDuringProofGeneration)
}

func testInitialize(h *integration.Harness, assert *require.Assertions, ctx context.Context, cfg *config.Config) {
	resInit, err := h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.NoError(err)

	nativeProof := wireToNativeProof(resInit.Proof)
	v, err := validation.NewValidator(cfg)
	assert.NoError(err)
	err = v.Validate(id, nativeProof)
	assert.NoError(err)

	resProof, err := h.GetProof(ctx, &api.GetProofRequest{Id: id, Challenge: shared.ZeroChallenge})
	assert.NoError(err)
	assert.Equal(resProof.Proof, resInit.Proof)

	_, err = h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = already completed")

	_, err = h.Reset(ctx, &api.ResetRequest{Id: id})
	assert.NoError(err)

	_, err = h.Reset(ctx, &api.ResetRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = not started")
}

func testInitializeParallel(h *integration.Harness, assert *require.Assertions, ctx context.Context, cfg *config.Config) {
	_, err := h.InitializeAsync(ctx, &api.InitializeAsyncRequest{Id: id})
	assert.NoError(err)

	_, err = h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = already initializing")

	_, err = h.InitializeAsync(ctx, &api.InitializeAsyncRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = already initializing")
}

func TestHarness_CrashRecovery(t *testing.T) {
	assert := require.New(t)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(30*time.Second))

	cfg := *cfg
	cfg.SpacePerUnit = 1 << 26
	cfg.NumFiles = 4
	cfg.MaxWriteFilesParallelism = 2
	cfg.MaxWriteInFileParallelism = 2

	h := newHarness(assert, &cfg)

	// Verify the initialization state.
	resState, err := h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.NoError(err)
	assert.Equal("NotStarted", resState.State.String())
	assert.Equal(cfg.SpacePerUnit, resState.RequiredSpace)

	// Start initializing, and wait a short time, so completion won't be reached.
	_, err = h.InitializeAsync(ctx, &api.InitializeAsyncRequest{Id: id})
	assert.NoError(err)
	time.Sleep(1 * time.Second)

	// Verify the initialization state.
	resState, err = h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.NoError(err)
	assert.Equal("Initializing", resState.State.String())
	assert.Equal(uint64(0), resState.RequiredSpace)

	// Kill the post server, stopping the initialization process ungracefully.
	err = h.TearDown(false)
	assert.NoError(err, "failed to crash post server")

	// Launch another server, with different init-critical config.
	diffCfg := cfg
	diffCfg.NumFiles = cfg.NumFiles << 1
	h = newHarness(assert, &diffCfg)

	// Verify that initialization recovery is not allowed.
	_, err = h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = config mismatch")
	_, err = h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = config mismatch")
	err = h.TearDown(false)
	assert.NoError(err, "failed to tear down harness")

	// Launch another server, with the same config.
	h = newHarness(assert, &cfg)
	defer func() {
		err = h.TearDown(true)
		assert.NoError(err, "failed to tear down harness")
	}()

	// Verify the initialization state.
	resState, err = h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.NoError(err)
	assert.Equal("Crashed", resState.State.String())
	assert.True(resState.RequiredSpace < cfg.SpacePerUnit)
	assert.True(resState.RequiredSpace > 0)

	// Complete the initialization procedure.
	resInit, err := h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.NoError(err)

	nativeProof := wireToNativeProof(resInit.Proof)
	v, err := validation.NewValidator(&cfg)
	assert.NoError(err)
	err = v.Validate(id, nativeProof)
	assert.NoError(err)

	// Verify the initialization state.
	_, err = h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.EqualError(err, "rpc error: code = Unknown desc = already completed")

	resState, err = h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.NoError(err)
	assert.Equal("Completed", resState.State.String())
	assert.Equal(uint64(0), resState.RequiredSpace)
}

func newHarness(assert *require.Assertions, cfg *config.Config) *integration.Harness {
	h, err := integration.NewHarness(cfg)
	assert.NoError(err)
	assert.NotNil(h)

	go func() {
		for {
			select {
			case err, more := <-h.ProcessErrors():
				if !more {
					return
				}
				assert.Fail("post server finished with error", err)
			}
		}
	}()

	return h
}

func wireToNativeProof(proof *api.Proof) *proving.Proof {
	return &proving.Proof{
		//Identity:     proof.Id,
		Challenge:    shared.Challenge(proof.Challenge),
		MerkleRoot:   proof.MerkleRoot,
		ProvenLeaves: proof.ProvenLeaves,
		ProofNodes:   proof.ProofNodes,
	}
}
