package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"time"
)

func main() {
	destination := os.Args[1]
	rpURL, err := url.Parse(destination)
	if err != nil {
		log.Fatal(err)
	}

	rp := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL.Host = rpURL.Host
			r.URL.Path = "/"
			r.URL.Scheme = rpURL.Scheme
			r.Host = rpURL.Host
		},
	}

	http.Handle("/", rp)

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
