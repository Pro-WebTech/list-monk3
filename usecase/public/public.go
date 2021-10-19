package public

import (
	"github.com/jmoiron/sqlx"

	"github.com/knadh/listmonk/dao"
)

type Public struct {
	db  *sqlx.DB
	sdb dao.SDB
}

func New(db *sqlx.DB, sdb dao.SDB) *Public {
	return &Public{db: db, sdb: sdb}
}
