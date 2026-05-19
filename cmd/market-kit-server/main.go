package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/solobat/market-kit/server"
)

func main() {
	config := server.LoadConfig()
	app, err := server.New(config)
	if err != nil {
		log.Fatalf("create market-kit server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("market-kit listening on %s", config.HTTPAddr)
	if err := app.ListenAndServe(ctx); err != nil && err != context.Canceled {
		log.Fatalf("run market-kit server: %v", err)
	}
}
