package models

const (
	TblStripePaymentHistory string = "stripe_payment_history"
)

type StripePaymentHistory struct {
	Product   string `json:"product"`
	PlanName  string `json:"plan_name"`
	PlanQty   int64  `json:"plan_qty"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Invoice   string `json:"invoice"`
	Platform  string `json:"platform"`
	Email     string `json:"email"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Mode      string `json:"mode"`
	Raw       string `json:"raw"`
}
