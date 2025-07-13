package middleware

import (
	"github.com/bbedward/enwind-api/config"
	"github.com/bbedward/enwind-api/internal/repositories/repositories"
	"github.com/danielgtaylor/huma/v2"
)

type Middleware struct {
	repository repositories.RepositoriesInterface
	api        huma.API
	cfg        *config.Config
}

func NewMiddleware(cfg *config.Config, repository repositories.RepositoriesInterface, api huma.API) *Middleware {
	return &Middleware{
		repository: repository,
		api:        api,
		cfg:        cfg,
	}
}
