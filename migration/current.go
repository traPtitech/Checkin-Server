package migration

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base model for all models
type Model struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate will set a UUID rather than numeric ID
func (base *Model) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}

// Admin model
type Admin struct {
	Model
	Name string `gorm:"type:varchar(255);not null" json:"name"`
}

// Customer model
type Customer struct {
	Model
	Name     string  `gorm:"type:varchar(255);not null" json:"name"`
	Email    *string `gorm:"type:varchar(255);uniqueIndex" json:"email"`
	Phone    *string `gorm:"type:varchar(20)" json:"phone"`
	Address  *string `gorm:"type:text" json:"address"`
	MetaData string  `gorm:"type:json" json:"meta_data"`
}

// Invoice model
type Invoice struct {
	Model
	CustomerID    uuid.UUID  `gorm:"type:char(36);not null" json:"customer_id"`
	Customer      Customer   `gorm:"foreignKey:CustomerID" json:"customer"`
	Amount        int64      `gorm:"not null" json:"amount"`
	Currency      string     `gorm:"type:varchar(3);not null" json:"currency"`
	Status        string     `gorm:"type:varchar(20);not null" json:"status"`
	PaymentMethod *string    `gorm:"type:varchar(50)" json:"payment_method"`
	InvoiceNumber string     `gorm:"type:varchar(50);uniqueIndex" json:"invoice_number"`
	DueDate       *time.Time `json:"due_date"`
	PaidDate      *time.Time `json:"paid_date"`
	CheckoutURL   *string    `gorm:"type:text" json:"checkout_url"`
	ExternalID    *string    `gorm:"type:varchar(255);index" json:"external_id"`
	ExternalData  string     `gorm:"type:json" json:"external_data"`
}

// CheckoutSession model
type CheckoutSession struct {
	Model
	InvoiceID   uuid.UUID `gorm:"type:char(36);not null" json:"invoice_id"`
	Invoice     Invoice   `gorm:"foreignKey:InvoiceID" json:"invoice"`
	SessionID   string    `gorm:"type:varchar(255);uniqueIndex" json:"session_id"`
	Status      string    `gorm:"type:varchar(20);not null" json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	RedirectURL string    `gorm:"type:text" json:"redirect_url"`
}
