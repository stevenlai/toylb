package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL   *url.URL
	Alive bool
	Proxy *httputil.ReverseProxy

	sync.RWMutex
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

func NewBackend(url *url.URL) *Backend {
	be := &Backend{
		URL:   url,
		Alive: true,
	}

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = be.URL.Host
			r.URL.Path = "/"
			r.URL.Scheme = be.URL.Scheme
			r.Host = be.URL.Host
		},
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		retries := GetRetriesFromContext(r)
		if retries < 3 {
			<-time.After(10 * time.Millisecond)
			ctx := context.WithValue(r.Context(), Retries, retries+1)
			rp.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		be.SetAlive(false)

		attempts := GetAttemptsFromContext(r)
		ctx := context.WithValue(r.Context(), Attempts, attempts+1)
		loadBalanceHandler(w, r.WithContext(ctx))
	}

	be.Proxy = rp

	return be
}
