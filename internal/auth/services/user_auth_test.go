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

// We'll use a gomock-generated mock for SessionRepository (NewMockSessionRepository)

func TestLoginCreatesSession(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Prepare mocks
	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)

	// Create a user with hashed password
	userID := uuid.New()
	username := "unitTest"
	email := "unit@test.com"
	password := "P4ssw0rd!"
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	passString := string(passHash)

	user := &models.User{
		Id:           &userID,
		Username:     &username,
		Email:        &email,
		PasswordHash: &passString,
	}

	// Expect repository call
	mockUserRepo.EXPECT().SelectUserByUsername(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *models.User) (*models.User, error) {
		return user, nil
	})

	// Create a real TokenService with a test config and the mock logger
	cfg := &config.AllConfiguration{PasetoKey: "01234567890123456789012345678901"} // 32 bytes
	tokenService := NewTokenService(cfg, mockLogger)

	mockSession := repositories.NewMockSessionRepository(ctrl)
	var capturedToken string
	mockSession.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, uid, token, expires string) error {
		capturedToken = token
		return nil
	})

	us := NewUserService(mockUserRepo, mockLogger, tokenService, mockSession)

	loginReq := models.UserLogin{Username: &username, PasswordHash: &password}
	resp, err := us.Login(ctx, loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEmpty(t, capturedToken)
	assert.Equal(t, capturedToken, resp.RefreshToken)

	// Refresh expiry should be roughly ~7 days from now
	exp, err := time.Parse(time.RFC3339, resp.RefreshExpiresAt)
	assert.NoError(t, err)
	assert.True(t, exp.After(time.Now().Add(6*24*time.Hour)))
}

func TestLogoutRevokesSession(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	tokenService := (*TokenService)(nil)

	mockSession := repositories.NewMockSessionRepository(ctrl)
	testToken := "test-refresh-token"
	mockSession.EXPECT().Revoke(gomock.Any(), testToken).Return(nil)

	us := NewUserService(mockUserRepo, mockLogger, tokenService, mockSession)

	err := us.Logout(ctx, testToken)
	assert.NoError(t, err)
}
