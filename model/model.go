package model

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	dsn := "host=localhost user=test password=$China2020 dbname=ss_log port=5432 sslmode=disable"
	var err error
	cst, _ := time.LoadLocation("Asia/Shanghai")
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().In(cst)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	log.Println("Database connected")

	if err := DB.AutoMigrate(&Location{}, &Log{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	log.Println("Database migrated")
	return nil
}
