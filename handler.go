package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"
)

type key int

const (
	Retries key = iota
	Attempts
)

func GetRetriesFromContext(r *http.Request) int {
	if retries, ok := r.Context().Value(Retries).(int); ok {
		return retries
	}
	return 0
}

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

func loadBalanceHandler(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

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

		attempts := GetAttemptsFromContext(r)
		ctx := context.WithValue(r.Context(), Attempts, attempts+1)
		loadBalanceHandler(w, r.WithContext(ctx))
	}

	rp.ServeHTTP(w, r)
}
