package impl

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/utl/structs"
	"log"
	"strings"
)

func NewRoleDaoImpl(lo *log.Logger) *RoleDaoImpl {
	return &RoleDaoImpl{lo: lo}
}

type RoleDaoImpl struct {
	lo *log.Logger
}

func (u *RoleDaoImpl) FindAll(db *sqlx.DB) (entity []models.RoleEntity, err error) {
	entity = []models.RoleEntity{}
	var buf strings.Builder
	buf.WriteString("SELECT * FROM ")
	buf.WriteString(models.TblRole)
	buf.WriteString(" WHERE 1 = $1 ")
	bind := []interface{}{1}

	rows, err := db.Query(buf.String(), bind...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		u.lo.Printf("err View: %v", err)
		return entity, err
	}

	for rows.Next() {
		var eachRow models.RoleEntity
		structs.MergeRow(rows, &eachRow)
		entity = append(entity, eachRow)
	}

	return entity, nil
}
