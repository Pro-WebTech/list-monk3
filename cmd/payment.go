package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/knadh/listmonk/usecase/payment"

	"github.com/labstack/echo"
	"github.com/stripe/stripe-go/v72"
)

type paymentHandler struct {
	svc payment.Service
	lo  *log.Logger
}

func setupPaymentHandler(svc payment.Service, lo *log.Logger) *paymentHandler {
	return &paymentHandler{
		svc: svc,
		lo:  lo,
	}
}

func (h *paymentHandler) checkoutEmailPlan(c echo.Context) error {
	app := c.Get("app").(*App)
	req := &payment.ItemEmailPlanReq{}

	if err := c.Bind(req); err != nil {
		h.lo.Println("err Bind checkoutEmailPlan: ", err)
		return err
	}

	email := c.Get("email").(string)
	if len(email) == 0 {
		if err := c.Bind(req); err != nil {
			h.lo.Println("err [checkoutEmailPlan]: email jwt is nil ")
			return errors.New("exp JWT")
		}
	}

	r, err := h.svc.CheckOutEmailPlan(h.lo, req, app.constants.RootURL, email)
	if err != nil {
		h.lo.Println("err checkoutEmailPlan: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}

func (h *paymentHandler) handlerStripe(c echo.Context) (err error) {
	r := c.Request()
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(c.Response().Writer, r.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.lo.Println("err weebhook [handlerStripe]: reading request body - ", err.Error())
		return
	}

	event := stripe.Event{}

	if err = json.Unmarshal(payload, &event); err != nil {
		h.lo.Println("err weebhook [handlerStripe]: Failed to parse webhook body json- ", err.Error())
		return
	}

	res, err := h.svc.WebhookStripe(h.lo, event)
	if err != nil {
		h.lo.Println("err checkoutEmailPlan: ", err)
		return err
	}

	return c.JSON(http.StatusOK, res)
}
