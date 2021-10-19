package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
)

type SDB interface {
	FindAll(db *sqlx.DB) ([]models.Settings, error)
	UpdateValue(db *sqlx.DB, key string, value string) error
	FindAStats(db *sqlx.DB) (types.JSONText, error)
	FindByKey(db *sqlx.DB, key string) (models.Settings, error)
}
