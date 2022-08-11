package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL   *url.URL
	Alive bool
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

type key int

const Retries key = iota

func GetRetriesFromContext(r *http.Request) int {
	if retries, ok := r.Context().Value(Retries).(int); ok {
		return retries
	}
	return 0
}

func loadBalanceHandler(w http.ResponseWriter, r *http.Request) {
	backend := backendPool.GetNextBackend()
	if backend == nil {
		http.Error(w, "No backend available", http.StatusServiceUnavailable)
		return
	}

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = backend.URL.Host
			r.URL.Path = "/"
			r.URL.Scheme = backend.URL.Scheme
			r.Host = backend.URL.Host
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

		backend.SetAlive(false)
		http.Error(w, "Try again", http.StatusServiceUnavailable)
	}

	rp.ServeHTTP(w, r)
}

func loadBackendFromConfig(be []string) []*Backend {
	var backends []*Backend

	for _, s := range be {
		backendUrl, err := url.Parse(s)
		if err != nil {
			log.Fatal(err)
		}
		backends = append(backends, &Backend{URL: backendUrl, Alive: true})
	}

	return backends
}

func main() {
	backends := loadBackendFromConfig(os.Args[1:])
	backendPool = NewBackendPool(backends)

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      http.HandlerFunc(loadBalanceHandler),
	}

	go func() {
		log.Println("Starting server")
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	wait := time.Second * 15
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Shutting down")
	os.Exit(0)
}
