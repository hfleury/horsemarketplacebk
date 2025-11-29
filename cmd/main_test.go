package main

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInitializeApp(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockConfig := mockconfig.NewMockConfiguration(ctrl)

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
	}).Times(3)

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

	mockConfig := mockconfig.NewMockConfiguration(ctrl)

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

func TestLauncher_Run(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockConfig := mockconfig.NewMockConfiguration(ctrl)

	// No expectations on mockConfig because run() just passes it to initializeAppFunc
	// and our mock initializeAppFunc doesn't use it.

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	mockServer := &MockServer{
		RunFunc: func(addr ...string) error {
			return nil
		},
	}

	launcher := &Launcher{
		AppInitializer: func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
			return mockServer, nil
		},
	}

	err := launcher.Run(ctx, mockConfig, mockDBFactory)

	assert.NoError(t, err)
}

func TestLauncher_Run_InitializeAppError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	ctx := context.Background()
	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	launcher := &Launcher{
		AppInitializer: func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
			return nil, errors.New("init error")
		},
	}

	err := launcher.Run(ctx, mockConfig, mockDBFactory)

	assert.Error(t, err)
	assert.Equal(t, "init error", err.Error())
}

func TestLauncher_Run_ServerRunError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	ctx := context.Background()
	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	mockServer := &MockServer{
		RunFunc: func(addr ...string) error {
			return errors.New("run error")
		},
	}

	launcher := &Launcher{
		AppInitializer: func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, error) {
			return mockServer, nil
		},
	}

	err := launcher.Run(ctx, mockConfig, mockDBFactory)

	assert.Error(t, err)
	assert.Equal(t, "run error", err.Error())
}
