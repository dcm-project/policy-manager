package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dcm-project/policy-manager/internal/apiserver"
	"github.com/dcm-project/policy-manager/internal/config"
	"github.com/dcm-project/policy-manager/internal/engineserver"
	"github.com/dcm-project/policy-manager/internal/handlers/engine"
	"github.com/dcm-project/policy-manager/internal/handlers/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/service"
	"github.com/dcm-project/policy-manager/internal/store"
)

type Server interface {
	Run(ctx context.Context) error
}

func main() {
	os.Exit(run())
}

func run() int {
	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return 1
	}

	// Initialize database
	db, err := store.InitDB(cfg)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return 1
	}

	// Create store
	dataStore := store.NewStore(db)
	defer func() {
		if err := dataStore.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Create service
	policyService := service.NewPolicyService(dataStore)

	// Create public API handler
	policyHandler := v1alpha1.NewPolicyHandler(policyService)

	// Create public API TCP listener
	publicListener, err := net.Listen("tcp", cfg.Service.BindAddress)
	if err != nil {
		log.Printf("Failed to create public API listener: %v", err)
		return 1
	}
	defer publicListener.Close()

	// Create public API server
	publicSrv := apiserver.New(cfg, publicListener, policyHandler)

	// Create private engine API handler
	engineHandler := engine.NewHandler()

	// Create private engine API TCP listener
	engineListener, err := net.Listen("tcp", cfg.Service.EngineBindAddress)
	if err != nil {
		log.Printf("Failed to create engine API listener: %v", err)
		return 1
	}
	defer engineListener.Close()

	// Create private engine API server
	engineSrv := engineserver.New(cfg, engineListener, engineHandler)

	if err := runServers([]Server{publicSrv, engineSrv}); err != nil {
		log.Printf("Failed to run servers: %v", err)
		return 1
	}

	return 0
}

func runServers(servers []Server) error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	errChan := make(chan error, len(servers))
	for _, server := range servers {
		wg.Add(1)
		go func(server Server) {
			defer wg.Done()
			if err := server.Run(ctx); err != nil {
				errChan <- err
			}
		}(server)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var firstErr error
	for err := range errChan {
		if err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}

	return firstErr
}
