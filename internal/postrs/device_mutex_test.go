package postrs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func Test_DeviceMutex(t *testing.T) {
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
