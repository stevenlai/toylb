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
	"time"
)

var backends []*Backend

type Backend struct {
	URL string
}

var mu sync.Mutex
var currentBackend int

func loadBalanceHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()

	destination := backends[currentBackend%len(backends)]
	rpURL, err := url.Parse(destination.URL)
	if err != nil {
		log.Fatal(err)
	}
	currentBackend++

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
	backends = loadBackendFromConfig(os.Args[1:])

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
