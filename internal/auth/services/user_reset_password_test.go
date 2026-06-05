package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	mockrepositories "github.com/hfleury/horsemarketplacebk/internal/mocks/auth/repositories"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

type mockEmailSender struct {
	sendFunc func(ctx context.Context, to, subject, body string) error
}

func (m *mockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	return m.sendFunc(ctx, to, subject, body)
}

func TestRequestPasswordReset_UserNotFound(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	// User not found should return nil (no error) for security reasons
	mockUserRepo.EXPECT().
		SelectUserByEmail(ctx, gomock.Any()).
		Return(nil, nil).
		Times(1)

	mockLogger.EXPECT().Log(ctx, config.InfoLevel, "password reset requested for unknown email", gomock.Any()).Times(1)

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)
	userService.SetEmailSender(&mockEmailSender{})

	err := userService.RequestPasswordReset(ctx, "nonexistent@test.com")
	assert.NoError(t, err)
}

func TestRequestPasswordReset_Success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	userID := uuid.New()
	username := "testuser"
	emailAddr := "test@example.com"
	user := &models.User{
		Id:       &userID,
		Username: &username,
		Email:    &emailAddr,
	}

	mockUserRepo.EXPECT().
		SelectUserByEmail(ctx, gomock.Any()).
		Return(user, nil).
		Times(1)

	mockResetRepo.EXPECT().
		Create(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, pr *models.PasswordReset) (*models.PasswordReset, error) {
			assert.Equal(t, &userID, pr.UserId)
			assert.False(t, *pr.IsUsed)
			assert.True(t, pr.ExpiresAt.After(time.Now()))
			return pr, nil
		}).
		Times(1)

	emailSent := false
	sender := &mockEmailSender{
		sendFunc: func(ctx context.Context, to, subject, body string) error {
			assert.Equal(t, emailAddr, to)
			assert.Contains(t, subject, "Reset your HorseMarketplace password")
			assert.Contains(t, body, "reset-password?token=")
			emailSent = true
			return nil
		},
	}

	mockLogger.EXPECT().Log(ctx, config.InfoLevel, "sending password reset email", gomock.Any()).Times(1)

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)
	userService.SetEmailSender(sender)

	err := userService.RequestPasswordReset(ctx, "test@example.com")
	assert.NoError(t, err)
	assert.True(t, emailSent)
}

func TestResetPassword_TokenInvalid(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	mockResetRepo.EXPECT().
		SelectByToken(ctx, "invalid-token").
		Return(nil, errors.New("not found")).
		Times(1)

	mockLogger.EXPECT().Log(ctx, config.InfoLevel, "invalid password reset token used", gomock.Any()).Times(1)

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)

	err := userService.ResetPassword(ctx, "invalid-token", "NewPass123!")
	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestResetPassword_TokenAlreadyUsed(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	isUsed := true
	pr := &models.PasswordReset{
		IsUsed: &isUsed,
	}

	mockResetRepo.EXPECT().
		SelectByToken(ctx, "used-token").
		Return(pr, nil).
		Times(1)

	mockLogger.EXPECT().Log(ctx, config.InfoLevel, "password reset token already used", gomock.Any()).Times(1)

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)

	err := userService.ResetPassword(ctx, "used-token", "NewPass123!")
	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestResetPassword_TokenExpired(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	isUsed := false
	expiredAt := time.Now().Add(-1 * time.Hour)
	pr := &models.PasswordReset{
		IsUsed:    &isUsed,
		ExpiresAt: &expiredAt,
	}

	mockResetRepo.EXPECT().
		SelectByToken(ctx, "expired-token").
		Return(pr, nil).
		Times(1)

	mockLogger.EXPECT().Log(ctx, config.InfoLevel, "password reset token expired", gomock.Any()).Times(1)

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)

	err := userService.ResetPassword(ctx, "expired-token", "NewPass123!")
	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestResetPassword_ValidationFail(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	isUsed := false
	expiresAt := time.Now().Add(1 * time.Hour)
	userID := uuid.New()
	pr := &models.PasswordReset{
		UserId:    &userID,
		IsUsed:    &isUsed,
		ExpiresAt: &expiresAt,
	}

	mockResetRepo.EXPECT().
		SelectByToken(ctx, "valid-token").
		Return(pr, nil).
		Times(1)

	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)

	// Weak password - missing special char
	err := userService.ResetPassword(ctx, "valid-token", "WeakPass123")
	assert.Error(t, err)
	assert.Equal(t, "password must contain at least one special character", err.Error())
}

func TestResetPassword_Success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mockrepositories.NewMockUserRepository(ctrl)
	mockResetRepo := mockrepositories.NewMockPasswordResetRepository(ctrl)
	mockLogger := mockconfig.NewMockLogging(ctrl)

	isUsed := false
	expiresAt := time.Now().Add(1 * time.Hour)
	userID := uuid.New()
	pr := &models.PasswordReset{
		UserId:    &userID,
		IsUsed:    &isUsed,
		ExpiresAt: &expiresAt,
	}

	mockResetRepo.EXPECT().
		SelectByToken(ctx, "valid-token").
		Return(pr, nil).
		Times(1)

	mockUserRepo.EXPECT().
		UpdatePassword(ctx, userID.String(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id string, hash string) error {
			err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("ValidPass123!"))
			assert.NoError(t, err)
			return nil
		}).
		Times(1)

	mockResetRepo.EXPECT().
		MarkAsUsed(ctx, "valid-token").
		Return(nil).
		Times(1)

	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	userService := NewUserService(mockUserRepo, mockLogger, nil, nil)
	userService.SetPasswordResetRepo(mockResetRepo)

	err := userService.ResetPassword(ctx, "valid-token", "ValidPass123!")
	assert.NoError(t, err)
}
