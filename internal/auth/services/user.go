package services

import (
	"context"
	"errors"
	"regexp"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
)

type UserService struct {
	userRepo *repositories.UserRepoPsql
	logger   config.Logging
}

func NewUserService(userRepo *repositories.UserRepoPsql, logger config.Logging) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (us *UserService) CreateUser(ctx context.Context, userRequest models.UserResquest) (*models.User, error) {
	user := models.User{}
	if userRequest.Username == nil {
		return nil, errors.New("username and email cannot be empty")
	} else {
		exist, err := us.userRepo.IsUsernameTaken(ctx, *userRequest.Username)
		if err != nil {
			return nil, err
		}

		if exist {
			us.logger.Log(ctx, config.InfoLevel, "Username in use", map[string]any{
				"Message": "Username in use",
				"Data":    *userRequest.Username,
			})
			return nil, errors.New("Username or email already in use")
		}

		user.Username = userRequest.Username
	}

	if userRequest.Email == nil {
		return nil, errors.New("username and email cannot be empty")
	} else {
		exist, err := us.userRepo.IsEmailTaken(ctx, *userRequest.Email)
		if err != nil {
			return nil, err
		}

		if exist {
			us.logger.Log(ctx, config.InfoLevel, "Email in use", map[string]any{
				"Message": "Email in use",
				"Data":    *userRequest.Username,
			})
			return nil, errors.New("username or email already in use")
		}

		user.Email = userRequest.Email
	}

	err := us.validatePassword(*userRequest.PasswordHash)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (us *UserService) validatePassword(password string) error {
	letterPattern := `[a-zA-Z]`
	numberPattern := `[0-9]`
	specialCharPattern := `[!@#~$%^&*()_+\-=[\]{}|\\:;"'<>,.?/]`

	if matched, _ := regexp.MatchString(letterPattern, password); !matched {
		return errors.New("password must contain at least one letter")
	}

	if matched, _ := regexp.MatchString(numberPattern, password); !matched {
		return errors.New("password must contain at least one number")
	}

	if matched, _ := regexp.MatchString(specialCharPattern, password); !matched {
		return errors.New("password must contain at least one special character")
	}

	if len(password) < 8 {
		return errors.New("password must contain at least 8 characters")
	}

	return nil
}

func (us *UserService) hashPassword(password string) (string, error) {

}
