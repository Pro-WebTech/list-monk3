package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo"

	"github.com/knadh/listmonk/usecase/public"
)

type publicHandler struct {
	svc public.Service
	lo  *log.Logger
}

func setupPublicHandler(svc public.Service, lo *log.Logger) *publicHandler {
	return &publicHandler{
		svc: svc,
		lo:  lo,
	}
}

func (h *publicHandler) getEmailPlan(c echo.Context) error {
	r, err := h.svc.GetEmailPlan(h.lo)
	if err != nil {
		h.lo.Println("err getEmailPlan: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}

func (h *publicHandler) getLogoUrl(c echo.Context) error {
	r, err := h.svc.GetLogoUrl(h.lo)
	if err != nil {
		h.lo.Println("err getLogoUrl: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}
