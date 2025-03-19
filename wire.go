//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/traPtitech/Checkin-Server/repository/gorm"
	"github.com/traPtitech/Checkin-Server/router"
	"go.uber.org/zap"
)

// InitializeApp sets up the application with all its dependencies
func InitializeApp() (*App, error) {
	wire.Build(
		NewApp,
		gorm.NewRepository,
		router.NewHandlers,
		provideLogger,
	)
	return &App{}, nil
}

// provideLogger creates a zap logger
func provideLogger() (*zap.Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return logger, nil
}
