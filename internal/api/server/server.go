package server

import (
	"github.com/bbedward/enwind-api/config"
	"github.com/bbedward/enwind-api/internal/repositories/repositories"
)

// EmptyInput can be used when no input is needed.
type EmptyInput struct{}

// BaseAuthInput can be used when no input is needed and the user must be authenticated.
type BaseAuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token" required:"true"`
}

// DeletedResponse is used to return a deleted response.
type DeletedResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// Server implements generated.ServerInterface
type Server struct {
	Cfg        *config.Config
	Repository repositories.RepositoriesInterface
}
