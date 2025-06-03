package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/franouer/proto-sync/internal/app"
	"github.com/franouer/proto-sync/internal/infrastructure"
	interfaces "github.com/franouer/proto-sync/internal/interface"
)

func main() {
	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Initialize dependencies
	logger := infrastructure.NewColorLogger()
	fileRepo := infrastructure.NewFileRepository(logger)
	goModRepo := infrastructure.NewGoModRepository(logger)
	bufRepo := infrastructure.NewBufRepository(logger, fileRepo)

	// Initialize application service
	protoSyncService := app.NewProtoSyncService(logger, fileRepo, goModRepo, bufRepo)

	// Initialize CLI handler
	cliHandler := interfaces.NewCLIHandler(protoSyncService, logger)

	// Create root command and execute
	rootCmd := cliHandler.CreateRootCommand()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Error("Application failed: %v", err)
		os.Exit(1)
	}
}
