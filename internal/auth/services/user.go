package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo     repositories.UserRepository
	logger       config.Logging
	tokenService *TokenService
}

func NewUserService(userRepo repositories.UserRepository, logger config.Logging, tokenService *TokenService) *UserService {
	return &UserService{
		userRepo:     userRepo,
		logger:       logger,
		tokenService: tokenService,
	}
}

func (us *UserService) CreateUser(ctx context.Context, userRequest models.UserCreateResquest) (*models.User, error) {
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
			return nil, errors.New("username or email already in use")
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

	passHashed, err := us.hashPassword(ctx, *userRequest.PasswordHash)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = &passHashed

	userCreated, err := us.userRepo.Insert(ctx, &user)
	if err != nil {
		return nil, err
	}

	return userCreated, nil
}

func (us *UserService) SelectUserByUsername(ctx context.Context, user *models.UserGetRequest) (*models.User, error) {
	if user.Email == nil && user.Username == nil {
		return nil, errors.New("either username or email must be provided")
	}

	var modelUser *models.User
	var err error

	if user.Username != nil {
		modelUser = &models.User{Username: user.Username}
		modelUser, err = us.userRepo.SelectUserByUsername(ctx, modelUser)
		if err != nil {
			return nil, fmt.Errorf("error retrieving user by username: %w", err)
		}
	} else if user.Email != nil {
		modelUser = &models.User{Email: user.Email}
		modelUser, err = us.userRepo.SelectUserByEmail(ctx, modelUser)
		if err != nil {
			return nil, fmt.Errorf("error retrieving user by email: %w", err)
		}
	}

	return modelUser, nil
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

func (us *UserService) hashPassword(ctx context.Context, password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "failed to hash password", map[string]any{
			"Error": err.Error(),
		})
		return "", err
	}

	return string(passwordHash), nil
}

func (us *UserService) Login(ctx context.Context, userLogin models.UserLogin) (*models.LoginResponse, error) {
	if userLogin.Username == nil || userLogin.PasswordHash == nil {
		return nil, errors.New("username and password must be provided")
	}

	user := &models.User{Username: userLogin.Username}
	user, err := us.userRepo.SelectUserByUsername(ctx, user)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*userLogin.PasswordHash))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Create token with user ID, username, and email
	token, err := us.tokenService.CreateToken(user.Id.String(), *user.Username, *user.Email, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	// Return safe response without sensitive data
	expiresAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	loginResponse := &models.LoginResponse{
		Token: token,
		User: models.UserResponse{
			Username: *user.Username,
			Email:    *user.Email,
		},
		ExpiresAt: expiresAt,
	}

	return loginResponse, nil
}
