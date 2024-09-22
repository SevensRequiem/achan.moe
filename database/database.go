package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"achan.moe/logs"
	"github.com/go-gorm/caches/v4"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type memoryCacher struct {
	store *sync.Map
}

func (c *memoryCacher) init() {
	if c.store == nil {
		c.store = &sync.Map{}
	}
}

func (c *memoryCacher) Get(ctx context.Context, key string, q *caches.Query[any]) (*caches.Query[any], error) {
	c.init()
	val, ok := c.store.Load(key)
	if !ok {
		logs.Error("Key not found in cache")
		return nil, nil
	}

	if err := q.Unmarshal(val.([]byte)); err != nil {
		logs.Error("Failed to unmarshal cache value: %v", err)
		return nil, err
	}
	logs.Debug("Cache hit")
	return q, nil
}

func (c *memoryCacher) Store(ctx context.Context, key string, val *caches.Query[any]) error {
	c.init()
	res, err := val.Marshal()
	if err != nil {
		logs.Error("Failed to marshal cache value: %v", err)
		return err
	}

	c.store.Store(key, res)
	logs.Debug("Cache stored")
	return nil
}

func (c *memoryCacher) Invalidate(ctx context.Context) error {
	c.store = &sync.Map{}
	logs.Debug("Cache invalidated")
	return nil
}

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
	if dsn == "" {
		logs.Fatal("Failed to get database connection string")
	}

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logs.Fatal("Failed to connect to database: %v", err)
	}

	cachesPlugin := &caches.Caches{Conf: &caches.Config{
		Cacher: &memoryCacher{},
	}}

	if err := DB.Use(cachesPlugin); err != nil {
		logs.Fatal("Failed to use cache plugin: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		logs.Fatal("Failed to access underlying DB connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetMaxOpenConns(1000)
	sqlDB.SetConnMaxLifetime(15 * time.Minute)

	return DB
}

func Close() {
	sqlDB, err := DB.DB()
	if err != nil {
		logs.Error("Failed to access underlying DB connection: %v", err)
	}
	sqlDB.Close()
}
