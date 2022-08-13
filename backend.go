package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Backend struct {
	URL   *url.URL
	Alive bool
	sync.RWMutex
	Proxy *httputil.ReverseProxy
}

func (b *Backend) SetAlive(alive bool) {
	b.Lock()
	b.Alive = alive
	b.Unlock()
}

func (b *Backend) IsAlive() bool {
	var alive bool
	b.RLock()
	alive = b.Alive
	b.RUnlock()
	return alive
}
