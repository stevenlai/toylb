package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
)

func loadBackendFromConfig(be []string) []*Backend {
	var backends []*Backend

	for _, s := range be {
		backendUrl, err := url.Parse(s)
		if err != nil {
			log.Fatal(err)
		}
		backends = append(backends, NewBackend(backendUrl))
	}

	if len(backends) < 1 {
		os.Exit(1)
	}

	return backends
}

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:80", u.Host), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func runPassiveHealthCheck() {
	ticker := time.NewTicker(time.Second * 30)
	for {
		<-ticker.C
		backendPool.HealthCheck()
	}
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
			log.Fatal(err)
		}
	}()

	go runPassiveHealthCheck()

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
