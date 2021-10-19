package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

type UDB interface {
	View(*sqlx.DB, *models.UserReq) (models.Users, error)
}
