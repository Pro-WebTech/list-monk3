package payment

type ItemEmailPlanReq struct {
	Products string `json:"products"`
	PlanQty  int64  `json:"plan_qty"`
}
type StripUrlResp struct {
	Url string `json:"url"`
}

type (
	ProductPlan struct {
		Products ListProduct `json:"products"`
	}

	ListProduct struct {
		Sms               []Plan `json:"sms"`
		Emails            []Plan `json:"emails"`
		Validations       []Plan `json:"validations"`
		PushNotifications []Plan `json:"pushNotifications"`
	}

	Plan struct {
		PlanName  string `json:"plan_name"`
		PlanQty   string `json:"plan_qty"`
		PlanPrice string `json:"plan_price"`
	}
)
