package main

import (
	"context"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/integration"
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
	cfg.NumLabels = 1 << 22
	cfg.LabelSize = 8
	cfg.NumFiles = 4
	assert.NoError(cfg.Validate())

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
	assert.Equal(uint64(info.Config.NumLabels), cfg.NumLabels)
	assert.Equal(uint(info.Config.LabelSize), cfg.LabelSize)
	assert.Equal(uint(info.Config.K1), cfg.K1)
	assert.Equal(uint(info.Config.K2), cfg.K2)
	assert.Equal(uint(info.Config.NumFiles), cfg.NumFiles)
}

func testInitialize(h *integration.Harness, assert *require.Assertions, ctx context.Context, cfg *config.Config) {
	_, err := h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.NoError(err)

	resProof, err := h.Execute(ctx, &api.ExecuteRequest{Id: id, Challenge: shared.ZeroChallenge})
	assert.NoError(err)

	proof := shared.Proof{}
	err = proof.Decode(resProof.Proof.Data)
	assert.NoError(err)

	v, err := validation.NewValidator(cfg)
	assert.NoError(err)
	err = v.Validate(id, &proof)
	assert.NoError(err)

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
	cfg.NumLabels = 1 << 25
	cfg.LabelSize = 8
	cfg.NumFiles = 4
	cfg.MaxWriteFilesParallelism = 2
	cfg.MaxWriteInFileParallelism = 2

	h := newHarness(assert, &cfg)

	// Verify the initialization state.
	resState, err := h.GetState(ctx, &api.GetStateRequest{Id: id})
	assert.NoError(err)
	assert.Equal("NotStarted", resState.State.String())
	assert.Equal(cfg.Space(), resState.RequiredSpace)

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
	assert.True(resState.RequiredSpace < cfg.Space())
	assert.True(resState.RequiredSpace > 0)

	// Complete the initialization procedure.
	_, err = h.Initialize(ctx, &api.InitializeRequest{Id: id})
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
