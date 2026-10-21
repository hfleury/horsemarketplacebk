package services

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	mockrepositories "github.com/hfleury/horsemarketplacebk/internal/mocks/auth/repositories"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newPermissiveLogger(ctrl *gomock.Controller) *mockconfig.MockLogging {
	logger := mockconfig.NewMockLogging(ctrl)
	logger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	return logger
}

func TestCreateUser_TracksUserSignedUp(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	userID := uuid.New()
	request := models.UserCreateResquest{
		Username:     rtnStringPointer("tracked"),
		Email:        rtnStringPointer("tracked@test.com"),
		PasswordHash: rtnStringPointer("p4ssw[]rd"),
	}

	userRepo := mockrepositories.NewMockUserRepository(ctrl)
	userRepo.EXPECT().IsUsernameTaken(ctx, "tracked").Return(false, nil)
	userRepo.EXPECT().IsEmailTaken(ctx, "tracked@test.com").Return(false, nil)
	userRepo.EXPECT().Insert(ctx, gomock.Any()).Return(&models.User{Id: &userID, Username: request.Username, Email: request.Email}, nil)
	tracker := mockanalytics.NewFakeTracker()
	userService := NewUserService(userRepo, newPermissiveLogger(ctrl), nil, nil)
	userService.SetTracker(tracker)

	_, err := userService.CreateUser(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, []analytics.Event{{Name: analytics.EventUserSignedUp, DistinctID: userID.String()}}, tracker.Events())
}

func TestCreateUser_FailureTracksNothing(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	userRepo := mockrepositories.NewMockUserRepository(ctrl)
	userRepo.EXPECT().IsUsernameTaken(ctx, "taken").Return(true, nil)
	tracker := mockanalytics.NewFakeTracker()
	userService := NewUserService(userRepo, newPermissiveLogger(ctrl), nil, nil)
	userService.SetTracker(tracker)

	_, err := userService.CreateUser(ctx, models.UserCreateResquest{Username: rtnStringPointer("taken"), PasswordHash: rtnStringPointer("p4ssw[]rd")})

	assert.Error(t, err)
	assert.Empty(t, tracker.Events())
}

func TestVerifyEmail_TracksEmailVerifiedOnlyWhenValid(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn time.Duration
		tracked   bool
	}{
		{name: "valid token", expiresIn: time.Hour, tracked: true},
		{name: "expired token", expiresIn: -time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			userID := uuid.New()
			expiresAt := time.Now().UTC().Add(tt.expiresIn)
			userRepo := mockrepositories.NewMockUserRepository(ctrl)
			verificationRepo := mockrepositories.NewMockEmailVerificationRepository(ctrl)
			verificationRepo.EXPECT().SelectByToken(ctx, "token").Return(&models.EmailVerification{UserId: &userID, ExpiresAt: &expiresAt}, nil)
			if tt.tracked {
				verificationRepo.EXPECT().MarkVerified(ctx, "token").Return(nil)
				userRepo.EXPECT().SetVerified(ctx, userID.String(), true).Return(nil)
			}
			tracker := mockanalytics.NewFakeTracker()
			userService := NewUserService(userRepo, newPermissiveLogger(ctrl), nil, nil)
			userService.SetEmailVerificationRepo(verificationRepo)
			userService.SetTracker(tracker)

			_ = userService.VerifyEmail(ctx, "token")

			if !tt.tracked {
				assert.Empty(t, tracker.Events())
				return
			}
			assert.Equal(t, []analytics.Event{{Name: analytics.EventEmailVerified, DistinctID: userID.String()}}, tracker.Events())
		})
	}
}

func TestLogin_TracksUserLoggedInOnlyOnSuccess(t *testing.T) {
	tests := []struct {
		name     string
		password string
		tracked  bool
	}{
		{name: "correct password", password: "RightPass1!", tracked: true},
		{name: "wrong password", password: "wrongpass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			userID := uuid.New()
			username, email := "alice", "alice@example.com"
			hash, _ := bcrypt.GenerateFromPassword([]byte("RightPass1!"), bcrypt.MinCost)
			hashString := string(hash)
			user := &models.User{Id: &userID, Username: &username, Email: &email, PasswordHash: &hashString}

			logger := newPermissiveLogger(ctrl)
			userRepo := mockrepositories.NewMockUserRepository(ctrl)
			userRepo.EXPECT().SelectUserByUsername(gomock.Any(), gomock.Any()).Return(user, nil)
			sessionRepo := mockrepositories.NewMockSessionRepository(ctrl)
			sessionRepo.EXPECT().Create(gomock.Any(), userID.String(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			tokenService := NewTokenService(&config.AllConfiguration{PasetoKey: "01234567890123456789012345678901"}, logger)
			tracker := mockanalytics.NewFakeTracker()
			userService := NewUserService(userRepo, logger, tokenService, sessionRepo)
			userService.SetTracker(tracker)

			_, _ = userService.Login(ctx, models.UserLogin{Username: &username, PasswordHash: &tt.password})

			if !tt.tracked {
				assert.Empty(t, tracker.Events())
				return
			}
			assert.Equal(t, []analytics.Event{{
				Name:       analytics.EventUserLoggedIn,
				DistinctID: userID.String(),
				Properties: map[string]any{"role": "user"},
			}}, tracker.Events())
		})
	}
}
