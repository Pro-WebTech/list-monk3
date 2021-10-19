package public

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/knadh/listmonk/models"
)

func (p Public) GetEmailPlan(lo *log.Logger) (*models.DefaultResponse, error) {
	settings, err := p.sdb.FindByKey(p.db, "emailsent.plan")
	if err != nil {
		return &models.DefaultResponse{Data: nil, Code: http.StatusInternalServerError, Message: err.Error()}, nil
	}

	res := []EmailPlan{}
	err = json.Unmarshal([]byte(settings.Value), &res)
	if err != nil {
		lo.Println("err Unmarshal[GetEmailPlan]: ", err)
	}
	return &models.DefaultResponse{Data: res, Code: http.StatusOK, Message: "success"}, nil
}
