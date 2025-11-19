package main

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInitializeApp(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockConfig := config.NewMockConfiguration(ctrl)

	mockConfig.EXPECT().LoadConfiguration().Times(1)
	mockConfig.EXPECT().GetConfig().Return(&config.AllConfiguration{
		Psql: config.PostgresConfig{
			Username: "test_user",
			Password: "test_pass",
			DdName:   "test_db",
			Host:     "localhost",
			Port:     "5432",
			SSLMode:  "disable",
		},
	}).Times(2)

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	server, err := initializeApp(ctx, mockConfig, mockDBFactory)

	assert.NoError(t, err)
	assert.NotNil(t, server)
}

func TestInitializeApp_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockConfig := config.NewMockConfiguration(ctrl)

	mockConfig.EXPECT().LoadConfiguration().Times(1)
	mockConfig.EXPECT().GetConfig().Return(&config.AllConfiguration{
		Psql: config.PostgresConfig{
			Username: "test_user",
			Password: "test_pass",
			DdName:   "test_db",
			Host:     "localhost",
			Port:     "5432",
			SSLMode:  "disable",
		},
	}).Times(1)

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return nil, errors.New("db error")
	}

	server, err := initializeApp(ctx, mockConfig, mockDBFactory)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Equal(t, "db error", err.Error())
}

// MockServer is a mock of Server interface
type MockServer struct {
	RunFunc func(addr ...string) error
}

func (m *MockServer) Run(addr ...string) error {
	return m.RunFunc(addr...)
}

func TestRun(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockConfig := config.NewMockConfiguration(ctrl)

	mockConfig.EXPECT().LoadConfiguration().Times(1)
	mockConfig.EXPECT().GetConfig().Return(&config.AllConfiguration{
		Psql: config.PostgresConfig{
			Username: "test_user",
			Password: "test_pass",
			DdName:   "test_db",
			Host:     "localhost",
			Port:     "5432",
			SSLMode:  "disable",
		},
	}).Times(2)

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	// Mock initializeAppFunc
	originalInitializeAppFunc := initializeAppFunc
	defer func() { initializeAppFunc = originalInitializeAppFunc }()

	mockServer := &MockServer{
		RunFunc: func(addr ...string) error {
			return nil
		},
	}

	initializeAppFunc = func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
		return mockServer, nil
	}

	err := run(ctx, mockConfig, mockDBFactory)

	assert.NoError(t, err)
}
