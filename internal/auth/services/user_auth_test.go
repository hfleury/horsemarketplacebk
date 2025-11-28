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

// fakeSessionRepo is a lightweight in-test implementation of SessionRepository
type fakeSessionRepo struct {
	createdUserID   string
	createdToken    string
	createdExpiry   string
	revokedToken    string
	validateReturns struct {
		userID   string
		isActive bool
		expires  string
		err      error
	}
}

func (f *fakeSessionRepo) Create(ctx context.Context, userID string, sessionToken string, expiresAt string) error {
	f.createdUserID = userID
	f.createdToken = sessionToken
	f.createdExpiry = expiresAt
	return nil
}

func (f *fakeSessionRepo) Validate(ctx context.Context, sessionToken string) (string, bool, string, error) {
	return f.validateReturns.userID, f.validateReturns.isActive, f.validateReturns.expires, f.validateReturns.err
}

func (f *fakeSessionRepo) Revoke(ctx context.Context, sessionToken string) error {
	f.revokedToken = sessionToken
	return nil
}

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

	fakeRepo := &fakeSessionRepo{}

	us := NewUserService(mockUserRepo, mockLogger, tokenService, fakeRepo)

	loginReq := models.UserLogin{Username: &username, PasswordHash: &password}
	resp, err := us.Login(ctx, loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEmpty(t, fakeRepo.createdToken)
	assert.Equal(t, fakeRepo.createdToken, resp.RefreshToken)

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

	fakeRepo := &fakeSessionRepo{}

	us := NewUserService(mockUserRepo, mockLogger, tokenService, fakeRepo)

	testToken := "test-refresh-token"
	err := us.Logout(ctx, testToken)
	assert.NoError(t, err)
	assert.Equal(t, testToken, fakeRepo.revokedToken)
}
