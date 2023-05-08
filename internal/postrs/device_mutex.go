package postrs

import "sync"

// deviceMutex is a mutual exclusion lock for calls to different devices. It can be
// used to prevent concurrent calls to the same device from multiple goroutines.
//
// It wraps a map of device IDs to mutexes and provides a method to get a
// mutex for a given device ID in a thread-safe manner.
type deviceMutex struct {
	mtx    sync.Mutex
	device map[uint]*sync.Mutex
}

func (g *deviceMutex) Device(deviceId uint) *sync.Mutex {
	g.mtx.Lock()
	defer g.mtx.Unlock()

	if g.device == nil {
		g.device = make(map[uint]*sync.Mutex)
	}

	if _, ok := g.device[deviceId]; !ok {
		g.device[deviceId] = new(sync.Mutex)
	}

	return g.device[deviceId]
}
