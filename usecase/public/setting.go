package public

import (
	"log"
	"net/http"
	"strings"

	"github.com/knadh/listmonk/models"
)

func (p Public) GetLogoUrl(lo *log.Logger) (*models.DefaultResponse, error) {
	settings, err := p.sdb.FindByKey(p.db, "app.logo_url")
	if err != nil {
		return &models.DefaultResponse{Data: nil, Code: http.StatusInternalServerError, Message: err.Error()}, nil
	}
	settings.Value = strings.Trim(settings.Value, "\"")
	return &models.DefaultResponse{Data: settings, Code: http.StatusOK, Message: "success"}, nil
}
