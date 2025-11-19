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

func (ts *TokenService) CreateToken(username string, duration time.Duration) (string, error) {
	payload := paseto.JSONToken{
		Subject:    username,
		IssuedAt:   time.Now(),
		Expiration: time.Now().Add(duration),
	}

	token, err := ts.paseto.Encrypt(ts.symmetricKey, payload, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}
