package impl

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/utl/structs"
	"github.com/labstack/gommon/log"
	"strings"
)

func NewUserDaoImpl() *UserDaoImpl {
	return &UserDaoImpl{}
}

type UserDaoImpl struct{}

func (u *UserDaoImpl) View(db *sqlx.DB, req *models.UserReq) (models.Users, error) {
	var users models.Users
	var buf strings.Builder

	buf.WriteString("SELECT * FROM ")
	buf.WriteString(models.TblUsers)
	buf.WriteString(" WHERE 1 = $1 ")
	bind := []interface{}{1}

	if len(req.Email) > 0 {
		buf.WriteString("AND email = $2 ")
		bind = append(bind, req.Email)
	} else if req.Id > 0 {
		buf.WriteString("AND id = $2 ")
		bind = append(bind, req.Id)
	}
	rows, err := db.Query(buf.String(), bind...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	if err != nil {
		log.Errorf("err View: %v", err)
		return users, err
	}

	for rows.Next() {
		structs.MergeRow(rows, &users)
	}

	return users, nil
}
