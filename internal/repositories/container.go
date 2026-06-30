package repositories

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"gorm.io/gorm"
)

type Container struct {
	DB               *gorm.DB
	TenantRepository *Repository[models.Tenant]
}

func NewContainer(db *gorm.DB) *Container {
	return &Container{
		DB:               db,
		TenantRepository: New[models.Tenant](db, models.TableNameTenant),
	}
}
