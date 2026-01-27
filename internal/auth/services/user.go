package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo       repositories.UserRepository
	logger         config.Logging
	tokenService   *TokenService
	sessionRepo    repositories.SessionRepository
	emailSender    email.Sender
	emailVerifRepo repositories.EmailVerificationRepository
}

func NewUserService(userRepo repositories.UserRepository, logger config.Logging, tokenService *TokenService, sessionRepo repositories.SessionRepository) *UserService {
	return &UserService{
		userRepo:     userRepo,
		logger:       logger,
		tokenService: tokenService,
		sessionRepo:  sessionRepo,
	}
}

// SetEmailSender allows wiring an email.Sender after construction without
// changing existing constructor call sites.
func (us *UserService) SetEmailSender(s email.Sender) {
	us.emailSender = s
}

// SetEmailVerificationRepo wires the EmailVerification repository.
func (us *UserService) SetEmailVerificationRepo(r repositories.EmailVerificationRepository) {
	us.emailVerifRepo = r
}

func (us *UserService) CreateUser(ctx context.Context, userRequest models.UserCreateResquest) (*models.User, error) {
	user := models.User{}
	// Normalize inputs: trim username and normalize email to lowercase
	if userRequest.Username != nil {
		u := strings.TrimSpace(*userRequest.Username)
		userRequest.Username = &u
	}
	if userRequest.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*userRequest.Email))
		userRequest.Email = &e
	}
	if userRequest.Username == nil || userRequest.PasswordHash == nil {
		us.logger.Log(ctx, config.InfoLevel, "Username or email missing", map[string]any{
			"Message": "Username or email missing",
		})
		return nil, errors.New("username and email cannot be empty")
	} else {
		exist, err := us.userRepo.IsUsernameTaken(ctx, *userRequest.Username)
		if err != nil {
			us.logger.Log(ctx, config.ErrorLevel, "Error checking if username is taken", map[string]any{
				"Error": err.Error(),
			})
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
		us.logger.Log(ctx, config.InfoLevel, "Username or email missing", map[string]any{
			"Message": "Username or email missing",
		})
		return nil, errors.New("username and email cannot be empty")
	} else {
		exist, err := us.userRepo.IsEmailTaken(ctx, *userRequest.Email)
		if err != nil {
			us.logger.Log(ctx, config.ErrorLevel, "Error checking if email is taken", map[string]any{
				"Error": err.Error(),
			})
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
		us.logger.Log(ctx, config.InfoLevel, "Invalid password", map[string]any{
			"Error": err.Error(),
		})
		return nil, err
	}

	passHashed, err := us.hashPassword(ctx, *userRequest.PasswordHash)
	if err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "Error hashing password", map[string]any{
			"Error": err.Error(),
		})
		return nil, err
	}

	user.PasswordHash = &passHashed

	userCreated, err := us.userRepo.Insert(ctx, &user)
	if err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "Error inserting user", map[string]any{
			"Error": err.Error(),
		})
		return nil, err
	}

	// If an email sender and verification repository are configured, persist a verification token and send the email.
	if us.emailSender != nil && us.emailVerifRepo != nil && userCreated.Email != nil {
		verificationToken := uuid.New().String()
		now := time.Now().UTC()
		expiry := now.Add(48 * time.Hour)
		ev := &models.EmailVerification{
			UserId:            userCreated.Id,
			VerificationToken: &verificationToken,
			Email:             userCreated.Email,
			RequestedAt:       &now,
			ExpiresAt:         &expiry,
		}
		if _, err := us.emailVerifRepo.Create(ctx, ev); err != nil {
			us.logger.Log(ctx, config.ErrorLevel, "failed to persist email verification", map[string]any{"error": err.Error()})
		}

		verifyLink := fmt.Sprintf("/api/v1/auth/verify?token=%s", verificationToken)
		body := fmt.Sprintf("Hello %s,\n\nPlease verify your email by visiting the following link:\n%s\n\nIf you did not sign up, ignore this message.", func() string {
			if userCreated.Username != nil {
				us.logger.Log(ctx, config.InfoLevel, "Preparing verification email", map[string]any{
					"email": *userCreated.Email,
				})
				return *userCreated.Username
			}
			return "user"
		}(), verifyLink)

		// Log attempt to send verification email with token and link for debugging
		us.logger.Log(ctx, config.InfoLevel, "attempting to send verification email", map[string]any{
			"email":              *userCreated.Email,
			"verification_token": verificationToken,
			"verification_link":  verifyLink,
		})

		if err := us.emailSender.Send(ctx, *userCreated.Email, "Verify your HorseMarketplace email", body); err != nil {
			us.logger.Log(ctx, config.ErrorLevel, "failed to send verification email", map[string]any{"error": err.Error()})
		} else {
			us.logger.Log(ctx, config.InfoLevel, "sent verification email", map[string]any{"email": *userCreated.Email})
		}
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
		us.logger.Log(ctx, config.InfoLevel, "Username or password missing", map[string]any{
			"Message": "Username or password missing",
		})
		return nil, errors.New("username and password must be provided")
	}

	// Support login by either username or email. The frontend may submit a username
	// that is actually an email address. First attempt lookup by username; if not
	// found and the input looks like an email address, try lookup by email. This
	// avoids requiring the client to know which one the user provided.
	input := strings.TrimSpace(*userLogin.Username)
	var (
		user *models.User
		err  error
	)

	// If input contains an @, prefer email lookup
	if strings.Contains(input, "@") {
		// normalize email before lookup
		e := strings.ToLower(input)
		user, err = us.userRepo.SelectUserByEmail(ctx, &models.User{Email: &e})
		if err != nil {
			us.logger.Log(ctx, config.InfoLevel, "Invalid credentials", map[string]any{"Message": "Invalid credentials"})
			return nil, errors.New("invalid credentials")
		}
	} else {
		// try username first
		u := input
		user, err = us.userRepo.SelectUserByUsername(ctx, &models.User{Username: &u})
		if err != nil {
			// fallback: try email lookup in case the client provided an email without @ (unlikely)
			e := strings.ToLower(input)
			user, err = us.userRepo.SelectUserByEmail(ctx, &models.User{Email: &e})
			if err != nil {
				us.logger.Log(ctx, config.InfoLevel, "Invalid credentials", map[string]any{"Message": "Invalid credentials"})
				return nil, errors.New("invalid credentials")
			}
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*userLogin.PasswordHash))
	if err != nil {
		us.logger.Log(ctx, config.InfoLevel, "Invalid credentials", map[string]any{
			"Message": "Invalid credentials",
		})
		return nil, errors.New("invalid credentials")
	}

	// Create access token (short lived) and refresh session
	accessTTL := 24 * time.Hour
	role := "user"
	if user.Role != nil {
		role = *user.Role
	}
	accessToken, err := us.tokenService.CreateToken(user.Id.String(), *user.Username, *user.Email, role, accessTTL)
	if err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "Failed to create access token", map[string]any{
			"Error": err.Error(),
		})
		return nil, err
	}

	// create refresh token via sessionRepo
	if us.sessionRepo == nil {
		us.logger.Log(ctx, config.ErrorLevel, "Session repository not configured", map[string]any{
			"Error": "session repository is nil",
		})
		return nil, errors.New("session repository not configured")
	}

	refreshToken := uuid.New().String()
	refreshExpiry := time.Now().Add(7 * 24 * time.Hour)
	// sessionRepo.Create expects RFC3339 expiry string
	if err := us.sessionRepo.Create(ctx, user.Id.String(), refreshToken, refreshExpiry.Format(time.RFC3339)); err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "Failed to create refresh session", map[string]any{
			"Error": err.Error(),
		})
		return nil, err
	}

	// Build response
	loginResponse := &models.LoginResponse{
		Token: accessToken,
		User: models.UserResponse{
			Username: *user.Username,
			Email:    *user.Email,
		},
		ExpiresAt:        time.Now().Add(accessTTL).Format(time.RFC3339),
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiry.Format(time.RFC3339),
	}

	return loginResponse, nil
}

// Logout invalidates a refresh token (session)
func (us *UserService) Logout(ctx context.Context, refreshToken string) error {
	if us.sessionRepo == nil {
		return errors.New("session repository not configured")
	}
	return us.sessionRepo.Revoke(ctx, refreshToken)
}

// Refresh issues a new access token given a valid refresh token
func (us *UserService) Refresh(ctx context.Context, refreshToken string) (string, string, string, error) {
	if us.sessionRepo == nil {
		return "", "", "", errors.New("session repository not configured")
	}

	userID, isActive, expiresAt, err := us.sessionRepo.Validate(ctx, refreshToken)
	if err != nil {
		return "", "", "", errors.New("invalid refresh token")
	}
	if !isActive {
		// possible token reuse: revoke all sessions for this user and require re-login
		if revokeErr := us.sessionRepo.RevokeAllForUser(ctx, userID); revokeErr != nil {
			us.logger.Log(ctx, config.ErrorLevel, "failed to revoke all sessions after token reuse", map[string]any{"error": revokeErr.Error()})
		}
		return "", "", "", errors.New("refresh token reuse detected; all sessions revoked")
	}

	// check expiry
	if expiresAt != "" {
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			if t.Before(time.Now().UTC()) {
				return "", "", "", errors.New("refresh token expired")
			}
		}
	}

	// Load user details to include in new token
	user, err := us.userRepo.SelectUserByID(ctx, userID)
	if err != nil {
		return "", "", "", errors.New("failed to load user")
	}

	role := "user"
	if user.Role != nil {
		role = *user.Role
	}
	accessToken, err := us.tokenService.CreateToken(user.Id.String(), *user.Username, *user.Email, role, 15*time.Minute)
	if err != nil {
		return "", "", "", err
	}

	// Rotate refresh token transactionally: create a new one and revoke the old in a single operation.
	newRefresh := uuid.New().String()
	newExpiry := time.Now().Add(7 * 24 * time.Hour)
	if err := us.sessionRepo.Rotate(ctx, userID, refreshToken, newRefresh, newExpiry.Format(time.RFC3339)); err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "failed to rotate refresh token", map[string]any{"error": err.Error()})
		return "", "", "", errors.New("failed to rotate refresh token")
	}

	return accessToken, newRefresh, newExpiry.Format(time.RFC3339), nil
}

// VerifyEmail validates a verification token, marks the email verification record
// as verified and updates the user's is_verified flag.
func (us *UserService) VerifyEmail(ctx context.Context, token string) error {
	if us.emailVerifRepo == nil {
		return errors.New("email verification repository not configured")
	}

	ev, err := us.emailVerifRepo.SelectByToken(ctx, token)
	if err != nil {
		return errors.New("invalid verification token")
	}

	// check expiry
	if ev.ExpiresAt != nil {
		if ev.ExpiresAt.Before(time.Now().UTC()) {
			return errors.New("verification token expired")
		}
	}

	// mark email verification record as verified
	if err := us.emailVerifRepo.MarkVerified(ctx, token); err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "failed to mark verification record verified", map[string]any{"error": err.Error()})
		return errors.New("failed to verify email")
	}

	// update user record
	if ev.UserId != nil {
		if err := us.userRepo.SetVerified(ctx, ev.UserId.String(), true); err != nil {
			us.logger.Log(ctx, config.ErrorLevel, "failed to set user verified", map[string]any{"error": err.Error()})
			return errors.New("failed to verify user")
		}
	}

	return nil
}

// ResendVerification creates a new verification token for the given email and sends
// a verification email. To avoid leaking which emails exist, the handler may
// return a generic response; this method logs useful details for operators.
func (us *UserService) ResendVerification(ctx context.Context, email string) error {
	// normalize email
	email = strings.ToLower(strings.TrimSpace(email))

	if us.emailSender == nil || us.emailVerifRepo == nil {
		us.logger.Log(ctx, config.ErrorLevel, "email sender or verification repo not configured", nil)
		return errors.New("email sending not configured")
	}

	// look up user by email
	user, err := us.userRepo.SelectUserByEmail(ctx, &models.User{Email: &email})
	if err != nil || user == nil {
		// Do not reveal existence to callers, just log and return nil to indicate request accepted
		us.logger.Log(ctx, config.InfoLevel, "resend verification requested for unknown email", map[string]any{"email": email, "error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}()})
		return nil
	}

	// if already verified, nothing to do
	if user.IsVerified != nil && *user.IsVerified {
		us.logger.Log(ctx, config.InfoLevel, "resend verification requested for already verified user", map[string]any{"email": email, "user_id": user.Id})
		return nil
	}

	// optional rate-limit: check last request time and refuse if too recent (e.g., within 1 minute)
	if last, err := us.emailVerifRepo.GetLatestByEmail(ctx, email); err == nil && last != nil && last.RequestedAt != nil {
		if time.Since(*last.RequestedAt) < time.Minute {
			us.logger.Log(ctx, config.InfoLevel, "resend verification rate limited", map[string]any{"email": email})
			return errors.New("please wait a moment before requesting another verification email")
		}
	}

	verificationToken := uuid.New().String()
	now := time.Now().UTC()
	expiry := now.Add(48 * time.Hour)
	ev := &models.EmailVerification{
		UserId:            user.Id,
		VerificationToken: &verificationToken,
		Email:             user.Email,
		RequestedAt:       &now,
		ExpiresAt:         &expiry,
	}

	if _, err := us.emailVerifRepo.Create(ctx, ev); err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "failed to persist email verification (resend)", map[string]any{"error": err.Error(), "email": email})
		return err
	}

	verifyLink := fmt.Sprintf("/api/v1/auth/verify?token=%s", verificationToken)
	body := fmt.Sprintf("Hello %s,\n\nPlease verify your email by visiting the following link:\n%s\n\nIf you did not request this, ignore this message.", func() string {
		if user.Username != nil {
			return *user.Username
		}
		return "user"
	}(), verifyLink)

	us.logger.Log(ctx, config.InfoLevel, "attempting to send verification email (resend)", map[string]any{"email": email, "verification_token": verificationToken, "verification_link": verifyLink})

	if err := us.emailSender.Send(ctx, email, "Verify your HorseMarketplace email", body); err != nil {
		us.logger.Log(ctx, config.ErrorLevel, "failed to send verification email (resend)", map[string]any{"error": err.Error(), "email": email})
		return err
	}

	us.logger.Log(ctx, config.InfoLevel, "sent verification email (resend)", map[string]any{"email": email})
	return nil
}
