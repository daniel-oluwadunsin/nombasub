package db

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
}
