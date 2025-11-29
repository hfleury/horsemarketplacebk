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

// fakeEmailVerifRepo implements a minimal EmailVerificationRepository for tests
type fakeEmailVerifRepo struct {
	lastCreated *models.EmailVerification
	latest      *models.EmailVerification
}

func (f *fakeEmailVerifRepo) Create(ctx context.Context, ev *models.EmailVerification) (*models.EmailVerification, error) {
	f.lastCreated = ev
	return ev, nil
}

func (f *fakeEmailVerifRepo) SelectByToken(ctx context.Context, token string) (*models.EmailVerification, error) {
	return nil, nil
}

func (f *fakeEmailVerifRepo) MarkVerified(ctx context.Context, token string) error {
	return nil
}

func (f *fakeEmailVerifRepo) GetLatestByEmail(ctx context.Context, email string) (*models.EmailVerification, error) {
	return f.latest, nil
}

// simpleFakeSender records the last sent email
type simpleFakeSender struct {
	LastTo   string
	LastBody string
}

func (s *simpleFakeSender) Send(ctx context.Context, to, subject, body string) error {
	s.LastTo = to
	s.LastBody = body
	return nil
}

func TestLoginByEmailAndUsername(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockSession := repositories.NewMockSessionRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// Prepare user with hashed password
	uid := uuid.New()
	username := "bob"
	email := "bob@example.com"
	password := "P4ssw0rd!"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	hashStr := string(hash)
	user := &models.User{Id: &uid, Username: &username, Email: &email, PasswordHash: &hashStr}

	// token service
	cfg := &config.AllConfiguration{PasetoKey: "01234567890123456789012345678901"}
	tokenService := NewTokenService(cfg, mockLogger)

	// Expect session creation when login succeeds
	mockSession.EXPECT().Create(gomock.Any(), uid.String(), gomock.Any(), gomock.Any()).Return(nil)

	// Case A: login by email (input contains @)
	mockUserRepo.EXPECT().SelectUserByEmail(gomock.Any(), gomock.Any()).Return(user, nil)
	us := NewUserService(mockUserRepo, mockLogger, tokenService, mockSession)

	resp, err := us.Login(ctx, models.UserLogin{Username: &email, PasswordHash: &password})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, email, resp.User.Email)
	assert.NotEmpty(t, resp.Token)

	// Case B: login by username (no @)
	// Reset expectations: Expect username lookup
	mockUserRepo.EXPECT().SelectUserByUsername(gomock.Any(), gomock.Any()).Return(user, nil)
	// session create expectation for second login
	mockSession.EXPECT().Create(gomock.Any(), uid.String(), gomock.Any(), gomock.Any()).Return(nil)

	resp2, err2 := us.Login(ctx, models.UserLogin{Username: &username, PasswordHash: &password})
	assert.NoError(t, err2)
	assert.NotNil(t, resp2)
	assert.Equal(t, username, resp2.User.Username)
}

func TestLoginInvalidPassword(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockSession := repositories.NewMockSessionRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	uid := uuid.New()
	username := "alice"
	email := "alice@example.com"
	hash, _ := bcrypt.GenerateFromPassword([]byte("RightPass1!"), bcrypt.DefaultCost)
	hashStr := string(hash)
	user := &models.User{Id: &uid, Username: &username, Email: &email, PasswordHash: &hashStr}

	// token service not needed because login will fail before token creation
	mockUserRepo.EXPECT().SelectUserByUsername(gomock.Any(), gomock.Any()).Return(user, nil)

	us := NewUserService(mockUserRepo, mockLogger, nil, mockSession)

	_, err := us.Login(ctx, models.UserLogin{Username: &username, PasswordHash: func() *string { s := "wrongpass"; return &s }()})
	assert.Error(t, err)
}

func TestResendVerification_RateLimitAndSuccess(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repositories.NewMockUserRepository(ctrl)
	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	uid := uuid.New()
	email := "rate@test.com"
	username := "ratetest"
	user := &models.User{Id: &uid, Username: &username, Email: &email}

	// When looking up user by email, return the user
	mockUserRepo.EXPECT().SelectUserByEmail(gomock.Any(), gomock.Any()).Return(user, nil).Times(2)

	// Prepare fake repo and sender
	fakeRepo := &fakeEmailVerifRepo{}
	fakeSender := &simpleFakeSender{}

	us := NewUserService(mockUserRepo, mockLogger, nil, nil)
	us.SetEmailVerificationRepo(fakeRepo)
	us.SetEmailSender(fakeSender)

	// Case A: rate limited — set latest requested_at to now
	now := time.Now().UTC()
	fakeRepo.latest = &models.EmailVerification{RequestedAt: &now}

	err := us.ResendVerification(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "please wait")

	// Case B: success — set latest to old time
	old := now.Add(-2 * time.Minute)
	fakeRepo.latest = &models.EmailVerification{RequestedAt: &old}

	err2 := us.ResendVerification(ctx, email)
	assert.NoError(t, err2)
	assert.Equal(t, email, fakeSender.LastTo)
	assert.NotNil(t, fakeRepo.lastCreated)
}
