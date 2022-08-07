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

var destinations = os.Args[1:]
var mu sync.Mutex
var currentDestination int = 0

func loadBalanceHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	// select a backend

	destination := destinations[currentDestination%len(destinations)]
	rpURL, err := url.Parse(destination)
	if err != nil {
		log.Fatal(err)
	}
	currentDestination++

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

func main() {
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
