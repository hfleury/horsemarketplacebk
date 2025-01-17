package services

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser_success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	userRequest := models.UserResquest{
		Username:     rtnStringPointer("unitTest"),
		Email:        rtnStringPointer("unit@test.com"),
		PasswordHash: rtnStringPointer("p4ssw[]rd"),
	}

	userId := uuid.New()
	passHash, err := bcrypt.GenerateFromPassword([]byte(*userRequest.PasswordHash), bcrypt.DefaultCost)
	assert.NoError(t, err, "no error at Generate Password")
	passString := string(passHash)
	truePoint := true
	timeNow := time.Now()

	expectedUser := &models.User{
		Id:           &userId,
		Username:     userRequest.Username,
		Email:        userRequest.Email,
		PasswordHash: &passString,
		IsActive:     &truePoint,
		IsVerified:   &truePoint,
		LastLogin:    &timeNow,
		CreatedAt:    &timeNow,
		UpdatedAt:    &timeNow,
	}

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockUserRepo.EXPECT().IsUsernameTaken(ctx, *userRequest.Username).Return(false, nil).Times(1)
	mockUserRepo.EXPECT().IsEmailTaken(ctx, *userRequest.Email).Return(false, nil).Times(1)
	mockUserRepo.EXPECT().Insert(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *models.User) (*models.User, error) {
		assert.Equal(t, userRequest.Username, user.Username)
		assert.Equal(t, userRequest.Email, user.Email)
		err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*userRequest.PasswordHash))
		assert.NoError(t, err, "Password hash should match")
		return expectedUser, nil
	}).Times(1)

	mockZipperService := config.NewMockLogging(ctrl)
	mockZipperService.EXPECT().Log(ctx, config.InfoLevel, "Username in use", gomock.Any()).AnyTimes()
	mockZipperService.EXPECT().Log(ctx, config.InfoLevel, "Email in use", gomock.Any()).AnyTimes()

	userService := NewUserService(mockUserRepo, mockZipperService)
	rtnCreateUser, err := userService.CreateUser(ctx, userRequest)

	assert.NoError(t, err, "No error expected")
	assert.NotNil(t, rtnCreateUser, "Expected not to be nil")
	assert.Equal(t, expectedUser, rtnCreateUser, "Expected user to match")
}

func TestCreateUser_fail_username_nil(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	userRequest := models.UserResquest{
		Username:     nil,
		Email:        rtnStringPointer("unit@test.com"),
		PasswordHash: rtnStringPointer("p4ssw[]rd"),
	}

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockZipperService := config.NewMockLogging(ctrl)

	userService := NewUserService(mockUserRepo, mockZipperService)
	rtnCreateUser, err := userService.CreateUser(ctx, userRequest)

	assert.Error(t, err, "Expected an error for nil username")
	assert.Nil(t, rtnCreateUser, "Expected result to be nil")
	assert.Equal(t, "username and email cannot be empty", err.Error(), "Error message should match")
}

func rtnStringPointer(str string) *string {
	return &str
}
