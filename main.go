package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/router"
	"go.uber.org/zap"
)

// App holds all the dependencies for our application
type App struct {
	Logger *zap.Logger
	Server *echo.Echo
}

// NewApp creates a new application instance
func NewApp(logger *zap.Logger, handlers *router.Handlers) *App {
	e := handlers.Setup()
	return &App{
		Logger: logger,
		Server: e,
	}
}

// Start begins the HTTP server
func (a *App) Start() error {
	portStr := os.Getenv("CHECKIN_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		a.Logger.Fatal("failed to parse CHECKIN_PORT", zap.Error(err))
	}
	a.Logger.Info("starting server", zap.Int("port", port))
	if err := a.Server.Start(fmt.Sprintf(":%d", port)); err != nil {
		a.Logger.Info("shutting down the server")
		return err
	}
	return nil
}

func main() {
	// Initialize application with Wire
	app, err := InitializeApp()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize application: %v", err))
	}
	defer app.Logger.Sync()

	// Start the server
	if err := app.Start(); err != nil {
		app.Logger.Fatal("server error", zap.Error(err))
	}
}
