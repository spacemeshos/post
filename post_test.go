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
	test func(h *integration.Harness, assert *require.Assertions, ctx context.Context)
}

var testCases = []*harnessTestCase{
	{name: "info", test: testInfo},
	{name: "initialize", test: testInitialize},
}

var cfg = config.DefaultConfig()

func TestHarness(t *testing.T) {
	assert := require.New(t)

	h, err := integration.NewHarness(cfg)
	assert.NoError(err)

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

	defer func() {
		err := h.TearDown()
		assert.NoError(err, "failed to tear down harness")
		t.Logf("harness teared down")
	}()

	assert.NoError(err)
	assert.NotNil(h)
	t.Logf("harness launched")

	for _, testCase := range testCases {
		success := t.Run(testCase.name, func(t1 *testing.T) {
			ctx, _ := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
			testCase.test(h, assert, ctx)
		})

		if !success {
			break
		}
	}
}

func testInitialize(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
	assert.Nil(info.State)

	id := []byte("deadbeef")
	initProof, err := h.Initialize(ctx, &api.InitializeRequest{Id: id})
	assert.NoError(err)

	nativeProof := &proving.Proof{
		Identity:     initProof.Proof.Id,
		Challenge:    shared.Challenge(initProof.Proof.Challenge),
		MerkleRoot:   initProof.Proof.MerkleRoot,
		ProvenLeaves: initProof.Proof.ProvenLeaves,
		ProofNodes:   initProof.Proof.ProofNodes,
	}

	err = validation.NewValidator(cfg).Validate(nativeProof)
	assert.NoError(err)

	proof, err := h.GetProof(ctx, &api.GetProofRequest{Id: id, Challenge: shared.ZeroChallenge})
	assert.NoError(err)
	assert.Equal(proof.Proof, initProof.Proof)

	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
	//assert.Equal(info.State.Id, id)
	//assert.Equal(len(info.State.ProvenChallenges), 1)
	//assert.Equal(info.State.ProvenChallenges[0], hex.EncodeToString(shared.ZeroChallenge))

	_, err = h.Reset(ctx, &api.ResetRequest{Id: id})
	assert.NoError(err)

	info, err = h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
	assert.Nil(info.State)
}

func testInfo(h *integration.Harness, assert *require.Assertions, ctx context.Context) {
	info, err := h.GetInfo(ctx, &api.GetInfoRequest{})
	assert.NoError(err)
	assert.Equal(info.Version, shared.Version())
	assert.Equal(uint64(info.Config.SpacePerUnit), cfg.SpacePerUnit)
	assert.Equal(uint(info.Config.Difficulty), cfg.Difficulty)
	assert.Equal(uint(info.Config.Labels), cfg.NumProvenLabels)
	assert.Equal(uint(info.Config.CacheLayer), cfg.LowestLayerToCacheDuringProofGeneration)
}
