package postrs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/spacemeshos/post/config"
)

func Test_deviceMutex(t *testing.T) {
	var mtx deviceMutex

	lock1A := mtx.Device(1)
	lock1B := mtx.Device(1)
	lock2 := mtx.Device(2)

	locked1 := make(chan struct{})
	lock1A.Lock() // lock on device 1

	var eg errgroup.Group
	eg.Go(func() error {
		lock1B.Lock()
		defer lock1B.Unlock()

		select {
		case <-locked1:
		default:
			require.Fail(t, "lock1B acquired lock1A's lock prematurely")
		}

		return nil
	})

	lock2.Lock() // lock on device 2
	defer lock2.Unlock()

	time.Sleep(100 * time.Millisecond)
	close(locked1)
	lock1A.Unlock()

	require.NoError(t, eg.Wait())
}

func TestTranslateScryptParams(t *testing.T) {
	params := config.ScryptParams{
		N: 1 << (15 + 1),
		R: 1 << 5,
		P: 1 << 1,
	}

	cParams := translateScryptParams(params)

	require.EqualValues(t, 15, cParams.nfactor)
	require.EqualValues(t, 5, cParams.rfactor)
	require.EqualValues(t, 1, cParams.pfactor)
}
