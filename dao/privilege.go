package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

type PDB interface {
	FindByRoleID(db *sqlx.DB, roleID int64) (entities []models.Privilege, err error)
}
