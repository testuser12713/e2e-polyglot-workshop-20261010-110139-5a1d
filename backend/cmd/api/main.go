// Command api is the single entry point of the Go API: it loads the
// configuration, opens the PostgreSQL pool, runs the schema migration, wires the
// invoice queue publisher, mounts the router and starts the HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"workshop/backend/internal/api"
	"workshop/backend/internal/auth"
	"workshop/backend/internal/config"
	"workshop/backend/internal/order"
	"workshop/backend/internal/queue"
	"workshop/backend/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("api: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := store.Migrate(ctx, pool); err != nil {
		return err
	}

	if err := auth.SeedFirstEmployee(ctx, pool, cfg.SeedEmployeeEmail, cfg.SeedEmployeePassword); err != nil {
		return err
	}

	valkeyOpts, err := redis.ParseURL(cfg.ValkeyURL)
	if err != nil {
		return fmt.Errorf("parse VALKEY_URL: %w", err)
	}
	rdb := redis.NewClient(valkeyOpts)
	defer func() { _ = rdb.Close() }()
	order.SetNotifier(queue.NewPublisher(rdb, cfg.QueueName))

	handler := api.NewRouter(api.Deps{Config: cfg, Pool: pool})
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("api: listening on :%s", cfg.HTTPPort)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
