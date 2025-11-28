package services

import (
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/o1egl/paseto"
)

type TokenService struct {
	paseto       *paseto.V2
	symmetricKey []byte
	logger       config.Logging
}

func NewTokenService(cfg *config.AllConfiguration, logger config.Logging) *TokenService {
	return &TokenService{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(cfg.PasetoKey),
		logger:       logger,
	}
}

func (ts *TokenService) CreateToken(userID, username, email string, duration time.Duration) (string, error) {
	now := time.Now()

	payload := paseto.JSONToken{
		Subject:    userID,            // User ID as subject (standard claim)
		IssuedAt:   now,               // Token creation time
		Expiration: now.Add(duration), // Token expiry
		NotBefore:  now,               // Token valid from now
	}

	// Add custom claims for username and email
	payload.Set("username", username)
	payload.Set("email", email)

	token, err := ts.paseto.Encrypt(ts.symmetricKey, payload, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}
