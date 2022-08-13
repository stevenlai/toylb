package main

import (
	"net/http"
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

	backend.Proxy.ServeHTTP(w, r)
}
