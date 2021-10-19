package dao

import (
	"strings"

	"github.com/jmoiron/sqlx"
)

type SPHDB interface {
	Create(db *sqlx.DB, eventType, invoice string, col *strings.Builder, bindVal *strings.Builder, bind *[]interface{}) (err error)
}
