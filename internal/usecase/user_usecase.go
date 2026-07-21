package usecase

import (
	"context"

	"github.com/nicolas2601/go-graphql-orders-api/internal/domain"
)

// UserUseCase expone la lectura de usuarios (para la query me y para resolver el dueno de una orden).
type UserUseCase struct {
	users domain.UserRepository
}

// NewUserUseCase construye el caso de uso con el repositorio inyectado.
func NewUserUseCase(users domain.UserRepository) *UserUseCase {
	if users == nil {
		panic("usecase: user repository must not be nil")
	}
	return &UserUseCase{users: users}
}

// Get devuelve un usuario por id, o ErrUserNotFound si no existe.
func (uc *UserUseCase) Get(ctx context.Context, id string) (domain.User, error) {
	return uc.users.GetByID(ctx, id)
}
