package public

import (
	"log"

	"github.com/knadh/listmonk/models"
)

type Service interface {
	GetEmailPlan(lo *log.Logger) (*models.DefaultResponse, error)
	GetLogoUrl(lo *log.Logger) (*models.DefaultResponse, error)
}
