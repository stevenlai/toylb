package main

import "sync/atomic"

type BackendPool struct {
	backends       []*Backend
	currentBackend uint64
}

var backendPool *BackendPool

func NewBackendPool(bes []*Backend) *BackendPool {
	bp := BackendPool{
		backends: bes,
	}
	return &bp
}

func (bp *BackendPool) NextIndex() uint64 {
	return atomic.AddUint64(&bp.currentBackend, uint64(1)) % uint64(len(bp.backends))
}

func (bp *BackendPool) GetNextBackend() *Backend {
	next := bp.NextIndex()
	backendCount := uint64(len(bp.backends))
	l := backendCount + next
	for i := next; i < l; i++ {
		idx := i % backendCount
		if bp.backends[idx].Alive {
			if i != next {
				atomic.StoreUint64(&bp.currentBackend, idx)
			}
			return bp.backends[idx]
		}
	}

	return nil
}

func (bp *BackendPool) HealthCheck() {
	for _, b := range bp.backends {
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
	}
}
