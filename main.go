package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/middleware"
	"github.com/traPtitech/Checkin-Server/repository"
	"github.com/traPtitech/Checkin-Server/router"
	"github.com/traPtitech/Checkin-Server/service/stripe"
	traqservice "github.com/traPtitech/Checkin-Server/service/traq"
	"go.uber.org/zap"
)

var (
	port = 3000
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Connect to DB
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		logger.Fatal("DATABASE_DSN is not set")
	}
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
	traqService := traqservice.NewTraQService(logger)

	jwtConfig := middleware.NewJWTConfig()

	handlers := router.Handlers{
		Logger:               logger,
		Repo:                 repo,
		SC:                   stripeService,
		TC:                   traqService,
		Mailer:               router.MockMailer{Logger: logger},
		JWTConfig:            jwtConfig,
		AdminTraQIDs:         parseAdminTraQIDs(os.Getenv("ADMIN_TRAQ_IDS")),
		PublicAPIBaseURL:     strings.TrimSpace(os.Getenv("PUBLIC_API_BASE_URL")),
		VerificationTokenTTL: parseVerificationTokenTTL(os.Getenv("VERIFY_EMAIL_TOKEN_TTL_MINUTES")),
	}

	e := echo.New()
	handlers.Setup(e)

	if err := e.Start(fmt.Sprintf(":%d", port)); err != nil {
		e.Logger.Info("shutting down the server")
	}
}

func parseAdminTraQIDs(raw string) map[string]struct{} {
	admins := map[string]struct{}{}
	for _, id := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		admins[trimmed] = struct{}{}
	}
	return admins
}

func parseVerificationTokenTTL(raw string) time.Duration {
	if strings.TrimSpace(raw) == "" {
		return 15 * time.Minute
	}
	minutes, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || minutes <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}
