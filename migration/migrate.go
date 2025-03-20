package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrationManager handles database migrations
type MigrationManager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB, logger *zap.Logger) *MigrationManager {
	return &MigrationManager{
		db:     db,
		logger: logger,
	}
}

// GetMigrations returns all migrations in order
func (m *MigrationManager) GetMigrations() []*gormigrate.Migration {
	// マイグレーションをバージョンごとに順番に登録
	// 新しいマイグレーションを追加する場合は、このリストに追加するだけ
	return []*gormigrate.Migration{
		V1Migration(), // 初期スキーマ定義
	}
}

// MigrateDB runs all pending migrations
func (m *MigrationManager) MigrateDB() error {
	m.logger.Info("Running database migrations")

	migrator := gormigrate.New(m.db, gormigrate.DefaultOptions, m.GetMigrations())

	// スキーマ初期化のオプション設定
	migrator.InitSchema(func(tx *gorm.DB) error {
		m.logger.Info("Initializing database schema")
		return nil
	})

	if err := migrator.Migrate(); err != nil {
		m.logger.Error("Migration failed", zap.Error(err))
		return err
	}

	m.logger.Info("Migration completed successfully")
	return nil
}

// RollbackLastMigration rolls back the last applied migration
func (m *MigrationManager) RollbackLastMigration() error {
	m.logger.Info("Rolling back last migration")

	migrator := gormigrate.New(m.db, gormigrate.DefaultOptions, m.GetMigrations())

	if err := migrator.RollbackLast(); err != nil {
		m.logger.Error("Rollback failed", zap.Error(err))
		return err
	}

	m.logger.Info("Rollback completed successfully")
	return nil
}

// RollbackToVersion rolls back migrations to a specific version
func (m *MigrationManager) RollbackToVersion(version string) error {
	m.logger.Info("Rolling back to version", zap.String("version", version))

	migrator := gormigrate.New(m.db, gormigrate.DefaultOptions, m.GetMigrations())

	if err := migrator.RollbackTo(version); err != nil {
		m.logger.Error("Rollback to version failed", zap.Error(err))
		return err
	}

	m.logger.Info("Rollback to version completed successfully")
	return nil
}
