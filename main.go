package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/middleware"
	"github.com/traPtitech/Checkin-Server/repository"
	"github.com/traPtitech/Checkin-Server/router"
	"github.com/traPtitech/Checkin-Server/service/stripe"
	"go.uber.org/zap"
)

var (
	port = 3000
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Connect to DB
	dsn := "user:password@tcp(localhost:3306)/checkin?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Fatal("failed to open db", zap.Error(err))
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		logger.Fatal("failed to ping db", zap.Error(err))
	}

	repo := repository.New(db)

	stripeService, err := stripe.NewStripeService(logger)
	if err != nil {
		logger.Fatal("failed to init stripe service", zap.Error(err))
	}

	jwtConfig := middleware.NewJWTConfig()

	handlers := router.Handlers{
		Logger:    logger,
		Repo:      repo,
		SC:        stripeService,
		JWTConfig: jwtConfig,
	}

	e := echo.New()
	handlers.Setup(e)

	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
		e.Logger.Info("shutting down the server")
	}
}
