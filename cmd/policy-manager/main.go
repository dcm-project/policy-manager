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

func main() {
	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := store.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
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
		log.Fatalf("Failed to create public API listener: %v", err)
	}
	defer publicListener.Close()

	// Create public API server
	publicSrv := apiserver.New(cfg, publicListener, policyHandler)

	// Create engine API handler
	engineHandler := engine.NewHandler()

	// Create engine API TCP listener
	engineListener, err := net.Listen("tcp", cfg.Service.EngineBindAddress)
	if err != nil {
		log.Fatalf("Failed to create engine API listener: %v", err)
	}
	defer engineListener.Close()

	// Create engine API server
	engineSrv := engineserver.New(cfg, engineListener, engineHandler)

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Run both servers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Go(func() {
		if err := publicSrv.Run(ctx); err != nil {
			errChan <- err
		}
	})

	wg.Go(func() {
		if err := engineSrv.Run(ctx); err != nil {
			errChan <- err
		}
	})

	// Wait for first error or completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}

	log.Println("All servers stopped")
}
