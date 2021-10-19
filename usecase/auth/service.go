package auth

import (
	"github.com/knadh/listmonk/models"
	"log"
)

type Service interface {
	Authenticate(*log.Logger, string, string) (*models.DefaultResponse, error)
}

// TokenGenerator represents token generator (jwt) interface
type TokenGenerator interface {
	GenerateToken(*models.Users) (string, string, error)
}

// Securer represents security interface
type Securer interface {
	Token(string) string
}
