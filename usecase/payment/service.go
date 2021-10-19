package payment

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/stripe/stripe-go/v72"

	"github.com/knadh/listmonk/dao"
	"github.com/knadh/listmonk/models"
)

type Service interface {
	CheckOutEmailPlan(lo *log.Logger, req *ItemEmailPlanReq, baseUrl, email string) (url *models.DefaultResponse, err error)
	WebhookStripe(lo *log.Logger, req stripe.Event) (url *models.DefaultResponse, err error)
}

type Payment struct {
	db    *sqlx.DB
	sdb   dao.SDB
	sphdb dao.SPHDB
}

func New(db *sqlx.DB, sdb dao.SDB, sphdb dao.SPHDB) *Payment {
	return &Payment{db: db, sdb: sdb, sphdb: sphdb}
}
