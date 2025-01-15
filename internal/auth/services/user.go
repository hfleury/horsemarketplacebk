package services

import (
	"context"
	"fmt"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

type UserService struct{}

func NewUserService() *UserService { return &UserService{} }

func (us *UserService) CreateUser(ctx context.Context, userRequest models.UserResquest) (*models.User, error) {
	fmt.Print(userRequest)
	fmt.Print(ctx)
	return nil, nil
}
