package repository

import api "github.com/traPtitech/Checkin-openapi/server"

// Repository defines the data access layer interface
type Repository interface {
	// Admin operations
	GetAdmins() ([]api.Admin, error)
	CreateAdmin(admin api.Admin) error
	DeleteAdmin(id string) error

	// Customer operations
	GetCustomer(id string) (*api.Customer, error)
	CreateCustomer(customer api.Customer) (*api.Customer, error)
	UpdateCustomer(customer api.Customer) error

	// Invoice operations
	CreateInvoice(invoice api.Invoice) (*api.Invoice, error)
	GetInvoices(limit int, offset int) ([]api.Invoice, error)
	GetCheckoutSessions(limit int, offset int) ([]api.GetCheckoutSessionsResponse, error)

	// Webhook handling
	HandleInvoicePaidWebhook(invoice api.Invoice) error
}
