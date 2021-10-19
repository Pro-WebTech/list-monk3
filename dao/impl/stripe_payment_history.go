package impl

import (
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

func NewStripePaymentHistoryDaoImpl(lo *log.Logger) *StripePaymentHistoryDaoImpl {
	return &StripePaymentHistoryDaoImpl{lo: lo}
}

type StripePaymentHistoryDaoImpl struct {
	lo *log.Logger
}

func (s *StripePaymentHistoryDaoImpl) Create(db *sqlx.DB, eventType, invoice string, col *strings.Builder, bindVal *strings.Builder, bind *[]interface{}) (err error) {
	var total int
	err = db.QueryRow("SELECT count(0) FROM stripe_payment_history where invoice = ? AND event_type = ?", invoice, eventType).Scan(&total)

	if err != nil || total == 0 {
		var buf strings.Builder
		buf.WriteString("INSERT INTO ")
		buf.WriteString(models.TblStripePaymentHistory)
		buf.WriteString(" (")
		buf.WriteString(col.String())
		buf.WriteString(") VALUES (")
		buf.WriteString(bindVal.String())
		buf.WriteString(")")
		stmt, errs := db.Prepare(buf.String())
		if errs != nil {
			return errs
		}
		defer stmt.Close()
		_, err = stmt.Exec(*bind...)
		return
	} else {
		return
	}
}
