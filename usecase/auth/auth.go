package auth

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/dao"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/utl/constants"
	"github.com/knadh/listmonk/utl/secure"
	"log"
	"net/http"
)

type Auth struct {
	db  *sqlx.DB
	udb dao.UDB
	tg  TokenGenerator
	sec Securer
}

func New(db *sqlx.DB, j TokenGenerator, sec Securer, udb dao.UDB) *Auth {
	return &Auth{db: db, tg: j, sec: sec, udb: udb}
}

func (a Auth) Authenticate(lo *log.Logger, username string, password string) (*models.DefaultResponse, error) {
	u, err := a.udb.View(a.db, &models.UserReq{Email: username})
	if err != nil {
		lo.Println("err db: ", err)
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}

	raws, _ := secure.Decode([]byte(u.Pass))
	ok, _ := raws.Verify([]byte(password))
	if !ok {
		lo.Println("err validation pas")
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}

	if u.Active != constants.Active {
		lo.Println("err user not active")
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}

	token, expire, err := a.tg.GenerateToken(&u)
	if err != nil {
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}

	authRep := &models.AuthToken{Token: token, Expires: expire, RefreshToken: a.sec.Token(token)}

	return &models.DefaultResponse{Data: authRep, Code: http.StatusOK, Message: "success"}, nil
}
