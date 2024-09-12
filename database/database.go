package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql" // Import the GORM driver for your database
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	Init()
}

func Init() *gorm.DB {
	godotenv.Load()

	username := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, dbname)
	//fmt.Println(dsn)
	if dsn == "" {
		log.Fatal("DSN is empty")
	}

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{}) // Initialize the DB variable
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := DB.DB() // Now this should work without causing a panic
	if err != nil {
		log.Fatalf("Failed to access underlying DB connection: %v", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetConnMaxLifetime(15 * time.Minute)

	return DB
}

func Close() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to access underlying DB connection: %v", err)
	}
	sqlDB.Close()
}
