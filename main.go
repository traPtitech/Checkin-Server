package main

import (
	"fmt"

	"github.com/traPtitech/Checkin-Server/repository/gorm"
	"github.com/traPtitech/Checkin-Server/router"
	"go.uber.org/zap"
)

var (
	port = 3000
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize repository
	repo, err := gorm.NewRepository(logger)
	if err != nil {
		logger.Fatal("failed to initialize repository", zap.Error(err))
	}

	// Initialize handlers
	r := router.Handlers{
		Logger: logger,
		Repo:   repo,
	}

	// Setup and start Echo server
	e := r.Setup()
	logger.Info("starting server", zap.Int("port", port))
	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
		logger.Info("shutting down the server")
	}
}
