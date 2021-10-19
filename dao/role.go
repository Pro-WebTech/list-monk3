package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

type RDB interface {
	FindAll(*sqlx.DB) ([]models.RoleEntity, error)
}
