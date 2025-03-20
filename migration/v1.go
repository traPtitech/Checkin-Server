package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// V1Migration は初期スキーマ定義のマイグレーション
func V1Migration() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202503210001_initial_schema",
		Migrate: func(tx *gorm.DB) error {
			// Admin テーブルの作成
			if err := tx.AutoMigrate(&Admin{}); err != nil {
				return err
			}

			// Customer テーブルの作成
			if err := tx.AutoMigrate(&Customer{}); err != nil {
				return err
			}

			// Invoice テーブルの作成
			if err := tx.AutoMigrate(&Invoice{}); err != nil {
				return err
			}

			// CheckoutSession テーブルの作成
			if err := tx.AutoMigrate(&CheckoutSession{}); err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropTable(&CheckoutSession{}); err != nil {
				return err
			}
			if err := tx.Migrator().DropTable(&Invoice{}); err != nil {
				return err
			}
			if err := tx.Migrator().DropTable(&Customer{}); err != nil {
				return err
			}
			if err := tx.Migrator().DropTable(&Admin{}); err != nil {
				return err
			}
			return nil
		},
	}
}
