package admin

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/dao"
	"github.com/knadh/listmonk/models"
)

type Admin struct {
	db  *sqlx.DB
	sdb dao.SDB
}

func New(db *sqlx.DB, sdb dao.SDB) *Admin {
	return &Admin{db: db, sdb: sdb}
}

func (a Admin) GetPlatformStats(lo *log.Logger) (*models.DefaultResponse, error) {
	stats, err := a.sdb.FindAStats(a.db)
	if err != nil {
		lo.Println("err db: ", err)
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}
	return &models.DefaultResponse{Data: stats, Code: http.StatusOK, Message: "success"}, nil
}

func (a Admin) GetPlatformSettings(lo *log.Logger) (*models.DefaultResponse, error) {
	settings, err := a.sdb.FindAll(a.db)
	if err != nil {
		lo.Println("err db: ", err)
		return &models.DefaultResponse{Data: nil, Code: http.StatusUnauthorized, Message: "username or password is wrong"}, nil
	}
	return &models.DefaultResponse{Data: settings, Code: http.StatusOK, Message: "success"}, nil
}

func (a Admin) UpdatePlatformSettings(lo *log.Logger, req *models.SettingReq) (*models.DefaultResponse, error) {
	for _, each := range req.Settings {
		a.sdb.UpdateValue(a.db, each.Key, each.Value)
	}

	return a.GetPlatformSettings(lo)
}
