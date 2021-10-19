package main

import (
	"github.com/knadh/listmonk/usecase/auth"
	"github.com/labstack/echo"
	"log"
	"net/http"
)

type authHandler struct {
	svc auth.Service
	lo  *log.Logger
}

func setupAuthHandler(svc auth.Service, lo *log.Logger) *authHandler {
	return &authHandler{
		svc: svc,
		lo:  lo,
	}
}

type credentials struct {
	Username string `json:"username" validate:"required"`
	Code     string `json:"code" validate:"required"`
}

func (h *authHandler) login(c echo.Context) error {
	cred := new(credentials)
	if err := c.Bind(cred); err != nil {
		return err
	}
	r, err := h.svc.Authenticate(lo, cred.Username, cred.Code)
	if err != nil {
		h.lo.Println("err Authenticate: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}
