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
	URL string
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

var mu sync.Mutex

func loadBalanceHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()

	backend := backendPool.backends[backendPool.currentBackend]
	rpURL, err := url.Parse(backend.URL)
	if err != nil {
		log.Fatal(err)
	}

	backendPool.currentBackend = backendPool.NextIndex()

	mu.Unlock()

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = rpURL.Host
			r.URL.Path = "/"
			r.URL.Scheme = rpURL.Scheme
			r.Host = rpURL.Host
		},
	}

	rp.ServeHTTP(w, r)

}

func loadBackendFromConfig(be []string) []*Backend {
	var backends []*Backend
	for _, s := range be {
		backends = append(backends, &Backend{URL: s})
	}

	return backends
}

func main() {
	backends := loadBackendFromConfig(os.Args[1:])
	backendPool = NewBackendPool(backends)

	http.HandleFunc("/", loadBalanceHandler)

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
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
