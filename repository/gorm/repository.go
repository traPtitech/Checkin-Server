package gorm

import (
	"fmt"
	"os"
	"time"

	driverMysql "github.com/go-sql-driver/mysql"
	"github.com/traPtitech/Checkin-Server/migration"
	"github.com/traPtitech/Checkin-Server/repository"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Implements repository.Repository interface.
type Repository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Create a new repository instance
func NewRepository(logger *zap.Logger) (repository.Repository, error) {
	// MariaDB connection string
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "checkin")

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSNConfig: &driverMysql.Config{
			User:                 dbUser,
			Passwd:               dbPassword,
			Net:                  "tcp",
			Addr:                 fmt.Sprintf("%s:%s", dbHost, dbPort),
			DBName:               dbName,
			Collation:            "utf8mb4_general_ci",
			ParseTime:            true,
			AllowNativePasswords: true,
		},
	}), &gorm.Config{
		// MariaDBにはnanosecondを保存できないため、microsecondまでprecisionを予め落とす
		NowFunc: func() time.Time {
			return time.Now().Truncate(time.Microsecond)
		},
	})
	if err != nil {
		return nil, err
	}

	// Run database migrations
	migrator := migration.NewMigrationManager(db, logger)
	if err := migrator.MigrateDB(); err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &Repository{
		db:     db,
		logger: logger,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Admin operations

// GetAdmins returns all admins
func (r *Repository) GetAdmins() ([]api.Admin, error) {
	r.logger.Debug("GetAdmins called")
	// TODO: Implement database query
	return []api.Admin{}, nil
}

// CreateAdmin creates a new admin
func (r *Repository) CreateAdmin(admin api.Admin) error {
	r.logger.Debug("CreateAdmin called")
	// TODO: Implement database query
	return nil
}

// DeleteAdmin deletes an admin by ID
func (r *Repository) DeleteAdmin(id string) error {
	r.logger.Debug("DeleteAdmin called", zap.String("id", id))
	// TODO: Implement database query
	return nil
}

// Customer operations

// GetCustomer gets a customer by ID
func (r *Repository) GetCustomer(id string) (*api.Customer, error) {
	r.logger.Debug("GetCustomer called", zap.String("id", id))
	// TODO: Implement database query
	return &api.Customer{}, nil
}

// CreateCustomer creates a new customer
func (r *Repository) CreateCustomer(customer api.Customer) (*api.Customer, error) {
	r.logger.Debug("CreateCustomer called")
	// TODO: Implement database query
	return &customer, nil
}

// UpdateCustomer updates an existing customer
func (r *Repository) UpdateCustomer(customer api.Customer) error {
	r.logger.Debug("UpdateCustomer called")
	// TODO: Implement database query
	return nil
}

// Invoice operations

// CreateInvoice creates a new invoice
func (r *Repository) CreateInvoice(invoice api.Invoice) (*api.Invoice, error) {
	r.logger.Debug("CreateInvoice called")
	// TODO: Implement database query
	return &invoice, nil
}

// GetInvoices returns a list of invoices with pagination
func (r *Repository) GetInvoices(limit int, offset int) ([]api.Invoice, error) {
	r.logger.Debug("GetInvoices called", zap.Int("limit", limit), zap.Int("offset", offset))
	// TODO: Implement database query
	return []api.Invoice{}, nil
}

// GetCheckoutSessions returns a list of checkout sessions with pagination
func (r *Repository) GetCheckoutSessions(limit int, offset int) ([]api.GetCheckoutSessionsResponse, error) {
	r.logger.Debug("GetCheckoutSessions called", zap.Int("limit", limit), zap.Int("offset", offset))
	// TODO: Implement database query
	return []api.GetCheckoutSessionsResponse{}, nil
}

// Webhook handling

// HandleInvoicePaidWebhook handles an invoice.paid webhook event
func (r *Repository) HandleInvoicePaidWebhook(invoice api.Invoice) error {
	r.logger.Debug("HandleInvoicePaidWebhook called")
	// TODO: Implement webhook handling logic
	return nil
}
