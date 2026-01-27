package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dcm-project/policy-manager/internal/apiserver"
	"github.com/dcm-project/policy-manager/internal/config"
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

	// Create handler
	policyHandler := v1alpha1.NewPolicyHandler(policyService)

	// Create TCP listener
	listener, err := net.Listen("tcp", cfg.Service.BindAddress)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Create server
	srv := apiserver.New(cfg, listener, policyHandler)

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

	log.Println("Server stopped")
}
