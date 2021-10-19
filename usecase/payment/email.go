package payment

import (
	"encoding/json"
	"errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/utl/structs"
)

func (p *Payment) CheckOutEmailPlan(lo *log.Logger, req *ItemEmailPlanReq, baseUrl, email string) (res *models.DefaultResponse, err error) {
	stripUrl := &StripUrlResp{}
	successUrl := baseUrl + "/success"
	cancelUrl := baseUrl + "/settings"
	if req.PlanQty == 0 {
		err = errors.New("Invalid request")
		return &models.DefaultResponse{Data: stripUrl, Code: http.StatusBadRequest, Message: err.Error()}, err
	}

	settings, err := p.sdb.FindByKey(p.db, "emailsent.plan")
	if err != nil {
		lo.Println("err queryDB[CheckOutEmailPlan]: ", err)
		return &models.DefaultResponse{Data: stripUrl, Code: http.StatusBadRequest, Message: err.Error()}, err
	}

	productPlan := &ProductPlan{}
	err = json.Unmarshal([]byte(settings.Value), productPlan)
	if err != nil {
		lo.Println("err Unmarshal[CheckOutEmailPlan]: ", err)
		return &models.DefaultResponse{Data: stripUrl, Code: http.StatusBadRequest, Message: err.Error()}, err
	}

	plan := []Plan{}
	switch req.Products {
	case "sms":
		plan = productPlan.Products.Sms
	case "emails":
		plan = productPlan.Products.Emails
	case "validations":
		plan = productPlan.Products.Validations
	case "push":
		plan = productPlan.Products.PushNotifications
	}

	var validReq bool
	for _, each := range plan {
		planQty, _ := strconv.ParseInt(each.PlanQty, 10, 64)
		if planQty == req.PlanQty {
			amount, _ := strconv.ParseFloat(each.PlanPrice, 64)
			params := &stripe.CheckoutSessionParams{
				PaymentMethodTypes: stripe.StringSlice([]string{
					"card",
				}),
				Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
				LineItems: []*stripe.CheckoutSessionLineItemParams{
					&stripe.CheckoutSessionLineItemParams{
						PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
							Currency: stripe.String("usd"),
							ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
								Name: stripe.String(each.PlanName),
							},
							UnitAmountDecimal: stripe.Float64(amount * 100),
						},
						Quantity: stripe.Int64(1),
					},
				},
				SuccessURL: stripe.String(successUrl),
				CancelURL:  stripe.String(cancelUrl),
			}
			params.AddMetadata("platform", baseUrl)
			params.AddMetadata("email", email)
			params.AddMetadata("product", req.Products)
			params.AddMetadata("planeName", each.PlanName)
			params.AddMetadata("planQty", each.PlanQty)
			session, err := session.New(params)
			if err != nil {
				lo.Println("err Unmarshal[CheckOutEmailPlan]: ", err)
				return &models.DefaultResponse{Data: stripUrl, Code: http.StatusBadRequest, Message: err.Error()}, err
			}
			stripUrl.Url = session.URL

			validReq = true
			break
		}
	}

	if !validReq {
		err = errors.New("Invalid request, not match!")
		return &models.DefaultResponse{Data: stripUrl, Code: http.StatusBadRequest, Message: err.Error()}, err
	}

	return &models.DefaultResponse{Data: stripUrl, Code: http.StatusOK, Message: "success"}, err
}

func (p *Payment) WebhookStripe(lo *log.Logger, event stripe.Event) (url *models.DefaultResponse, err error) {
	switch event.Type {
	case "checkout.session.completed":
		var checkoutRes stripe.CheckoutSession
		err = json.Unmarshal(event.Data.Raw, &checkoutRes)
		raw, _ := json.Marshal(checkoutRes)
		if err != nil {
			lo.Println("err weebhook [WebhookStripe]: ", event.Type, " (Error parsing webhook JSON)", err.Error())
			return
		}
		qty, errs := strconv.ParseInt(checkoutRes.Metadata["planQty"], 10, 64)
		if errs != nil {
			qty = 0
		}

		entity := &models.StripePaymentHistory{
			Product:   checkoutRes.Metadata["product"],
			PlanName:  checkoutRes.Metadata["planeName"],
			PlanQty:   qty,
			EventType: event.Type,
			Status:    string(checkoutRes.PaymentStatus),
			Invoice:   checkoutRes.PaymentIntent.ID,
			Platform:  checkoutRes.Metadata["platform"],
			Email:     checkoutRes.Metadata["email"],
			Amount:    checkoutRes.AmountTotal / 100,
			Currency:  string(checkoutRes.Currency),
			Mode:      string(checkoutRes.Mode),
			Raw:       string(raw),
		}

		var col strings.Builder
		var bindVal strings.Builder
		bind := []interface{}{}
		structs.MergeSqlInsert(entity, &col, &bindVal, &bind)
		err = p.sphdb.Create(p.db, event.Type, entity.Invoice, &col, &bindVal, &bind)
		if err != nil {
			lo.Println("err weebhook [WebhookStripe]: ", event.Type, " (Error insert DB)", err.Error())
			return
		}

		//Update
		if checkoutRes.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
			key := ""
			switch entity.Product {
			case "sms":
				key = "smssent.allowed"
			case "emails":
				key = "emailsent.allowed"
			case "validations":
				key = "validations.allowed"
			case "push":
				key = "pushsent.allowed"
			}
			settings, errs := p.sdb.FindByKey(p.db, key)
			if errs != nil {
				lo.Println("err weebhook [WebhookStripe]: ", event.Type, " (Error Find emailsent.allowed)", errs.Error())
				return
			}
			emailSentAllowed, _ := strconv.ParseInt(settings.Value, 10, 64)
			emailSentAllowed += qty
			p.sdb.UpdateValue(p.db, key, strconv.FormatInt(emailSentAllowed, 10))
		}
	default:
		raw, errs := json.Marshal(event.Data.Raw)
		if errs != nil {
			lo.Println("err weebhook [WebhookStripe]: ", event.Type, " (Error Marshal webhook JSON)", err.Error())
			return
		}
		entity := &models.StripePaymentHistory{
			EventType: event.Type,
			Raw:       string(raw),
			Invoice:   event.Data.Object["id"].(string),
		}

		var col strings.Builder
		var bindVal strings.Builder
		bind := []interface{}{}
		structs.MergeSqlInsert(entity, &col, &bindVal, &bind)
		err = p.sphdb.Create(p.db, event.Type, entity.Invoice, &col, &bindVal, &bind)
		if err != nil {
			lo.Println("err weebhook [WebhookStripe]: ", event.Type, " (Error insert DB)", err.Error())
			return
		}
	}

	return &models.DefaultResponse{Data: nil, Code: http.StatusOK, Message: "success"}, err
}
