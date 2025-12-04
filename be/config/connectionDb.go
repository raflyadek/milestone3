package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectionDb() *gorm.DB {
	if err := godotenv.Load(); err != nil {
		log.Printf("error load env %s", err)
	}

	dsn := os.Getenv("POSTGRE_URL")

	// Configure PostgreSQL driver to disable prepared statements
	// This is critical for Supabase connection pooling
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Disable prepared statements at driver level
	}

	// Open database with GORM config
	db, err := gorm.Open(postgres.New(pgConfig), &gorm.Config{
		PrepareStmt: false, // Also disable at GORM level for safety
	})
	if err != nil {
		log.Fatalf("error connect to database %s", err)
	}

	// Configure connection pool for better resource management
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("error getting database instance: %s", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	fmt.Println("success connect to db")
	return db
}
