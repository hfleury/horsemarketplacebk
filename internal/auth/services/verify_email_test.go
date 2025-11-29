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
)

func TestVerifyEmail_Success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockEmailVerif := repositories.NewMockEmailVerificationRepository(ctrl)

	us := NewUserService(mockUserRepo, mockLogger, nil, nil)
	us.SetEmailVerificationRepo(mockEmailVerif)

	token := "test-token"
	uid := uuid.New()
	now := time.Now().UTC()
	ev := &models.EmailVerification{
		UserId:    &uid,
		ExpiresAt: func() *time.Time { t := now.Add(1 * time.Hour); return &t }(),
	}

	mockEmailVerif.EXPECT().SelectByToken(ctx, token).Return(ev, nil)
	mockEmailVerif.EXPECT().MarkVerified(ctx, token).Return(nil)
	mockUserRepo.EXPECT().SetVerified(ctx, uid.String(), true).Return(nil)

	err := us.VerifyEmail(ctx, token)
	assert.NoError(t, err)
}

func TestVerifyEmail_Expired(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockEmailVerif := repositories.NewMockEmailVerificationRepository(ctrl)

	us := NewUserService(mockUserRepo, mockLogger, nil, nil)
	us.SetEmailVerificationRepo(mockEmailVerif)

	token := "expired-token"
	uid := uuid.New()
	past := time.Now().Add(-2 * time.Hour)
	ev := &models.EmailVerification{
		UserId:    &uid,
		ExpiresAt: &past,
	}

	mockEmailVerif.EXPECT().SelectByToken(ctx, token).Return(ev, nil)

	err := us.VerifyEmail(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}
