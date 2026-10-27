package database

import (
	"fmt"
	"log"
	"time"

	"order-management/config"
	orderModel "order-management/internal/order"
	productModel "order-management/internal/product"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewGormDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	gormLogLevel := logger.Warn
	if cfg.LogLevel == "debug" {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(gormLogLevel),
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve sql.DB: %w", err)
	}

	// Performance Requirement 11: Connection Pooling
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping verification failed: %w", err)
	}

	// AutoMigrate both domain models
	if err := db.AutoMigrate(&orderModel.Order{}, &productModel.Product{}); err != nil {
		log.Printf("[WARN] AutoMigrate warning: %v", err)
	}

	log.Println("[INFO] Successfully connected to PostgreSQL via GORM with connection pooling.")
	return db, nil
}
