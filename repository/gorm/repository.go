package gorm

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Implements repository.Repository interface.
type Repository struct {
	db     *gorm.DB
	logger *zap.Logger
}
