package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rqManager.Start(ctx)

	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
