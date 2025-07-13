package repositories

import (
	"context"

	"github.com/bbedward/enwind-api/ent"
	repository "github.com/bbedward/enwind-api/internal/repositories"
	user_repo "github.com/bbedward/enwind-api/internal/repositories/user"
)

// Repositories provides access to all repositories
//
//go:generate go run -mod=mod github.com/vburenin/ifacemaker -f "*.go" -i RepositoriesInterface -p repositories -s Repositories -o repositories_iface.go
type Repositories struct {
	db   *ent.Client
	base *repository.BaseRepository
	user user_repo.UserRepositoryInterface
}

// NewRepositories creates a new Repositories facade
func NewRepositories(db *ent.Client) *Repositories {
	base := repository.NewBaseRepository(db)
	userRepo := user_repo.NewUserRepository(db)
	return &Repositories{
		db:   db,
		base: base,
		user: userRepo,
	}
}

// Ent() returns the ent client
func (r *Repositories) Ent() *ent.Client {
	return r.db
}

// User returns the User repository
func (r *Repositories) User() user_repo.UserRepositoryInterface {
	return r.user
}

func (r *Repositories) WithTx(ctx context.Context, fn func(tx repository.TxInterface) error) error {
	return r.base.WithTx(ctx, fn)
}
