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

func TestRefreshRotatesSession(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockSession := repositories.NewMockSessionRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)

	// prepare a user
	uid := uuid.New()
	username := "rtest"
	email := "rtest@example.com"
	user := &models.User{Id: &uid, Username: &username, Email: &email}

	// token service
	cfg := &config.AllConfiguration{PasetoKey: "01234567890123456789012345678901"}
	tokenService := NewTokenService(cfg, mockLogger)

	// inputs
	oldRefresh := "old-refresh-token"
	userID := uid.String()
	// validate returns active session
	expiry := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	mockSession.EXPECT().Validate(gomock.Any(), oldRefresh).Return(userID, true, expiry, nil)

	// Select user
	mockUserRepo.EXPECT().SelectUserByID(gomock.Any(), userID).Return(user, nil)

	// Expect Rotate to be called; capture the new token
	var capturedNew string
	mockSession.EXPECT().Rotate(gomock.Any(), userID, oldRefresh, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _userID, _old, newToken, newExpiry string) error {
			capturedNew = newToken
			return nil
		},
	)

	us := NewUserService(mockUserRepo, mockLogger, tokenService, mockSession)

	access, newRefresh, newExpiry, err := us.Refresh(ctx, oldRefresh)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, newRefresh)
	assert.Equal(t, capturedNew, newRefresh)
	// expiry should be parseable RFC3339
	_, perr := time.Parse(time.RFC3339, newExpiry)
	assert.NoError(t, perr)
}

func TestRefreshDetectsReuseAndRevokesAll(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockSession := repositories.NewMockSessionRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)

	cfg := &config.AllConfiguration{PasetoKey: "01234567890123456789012345678901"}
	tokenService := NewTokenService(cfg, mockLogger)

	// simulate inactive token returned from Validate
	uid := uuid.New()
	userID := uid.String()
	mockSession.EXPECT().Validate(gomock.Any(), "reused-token").Return(userID, false, "", nil)
	mockSession.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)

	us := NewUserService(mockUserRepo, mockLogger, tokenService, mockSession)

	_, _, _, err := us.Refresh(ctx, "reused-token")
	assert.Error(t, err)
}
