package campaign

import (
	"log"

	"github.com/jmoiron/sqlx"

	"github.com/knadh/listmonk/dao"
)

type Service interface {
	GetListMessengers(lo *log.Logger) (resp []MessengersResponse, err error)
}

type Campaign struct {
	db  *sqlx.DB
	sdb dao.SDB
}

func New(db *sqlx.DB, sdb dao.SDB) *Campaign {
	return &Campaign{db: db, sdb: sdb}
}
