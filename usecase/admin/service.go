package admin

import (
	"log"

	"github.com/knadh/listmonk/models"
)

type Service interface {
	GetPlatformSettings(lo *log.Logger) (*models.DefaultResponse, error)
	UpdatePlatformSettings(lo *log.Logger, req *models.SettingReq) (*models.DefaultResponse, error)

	GetPlatformStats(lo *log.Logger) (*models.DefaultResponse, error)
}
