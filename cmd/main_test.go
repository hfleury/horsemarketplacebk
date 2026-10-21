package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

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
		Frontend: config.FrontendConfig{
			URL: "http://localhost:5173",
		},
	}).Times(9) // matches initializeApp's 9 configService.GetConfig() call sites (cmd/main.go)

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return &db.PsqlDB{}, nil
	}

	// releaseResources is not called: the zero-value PsqlDB has a nil Conn.
	server, releaseResources, err := initializeApp(ctx, mockConfig, mockDBFactory)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, releaseResources)
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
	}).Times(2) // InitSentry's cfg.Env, then the DB factory call

	ctx := context.Background()

	mockDBFactory := func(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
		return nil, errors.New("db error")
	}

	server, releaseResources, err := initializeApp(ctx, mockConfig, mockDBFactory)

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Nil(t, releaseResources)
	assert.Equal(t, "db error", err.Error())
}

// MockServer is a mock of Server interface
type MockServer struct {
	ListenAndServeFunc func() error
	ShutdownFunc       func(ctx context.Context) error
}

func (m *MockServer) ListenAndServe() error {
	return m.ListenAndServeFunc()
}

func (m *MockServer) Shutdown(ctx context.Context) error {
	return m.ShutdownFunc(ctx)
}

func noopDBFactory(config *config.AllConfiguration, logger zerolog.Logger) (*db.PsqlDB, error) {
	return &db.PsqlDB{}, nil
}

func launcherFor(server Server, releaseResources func()) *Launcher {
	return &Launcher{
		AppInitializer: func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, func(), error) {
			return server, releaseResources, nil
		},
	}
}

func TestLauncher_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	released := false
	mockServer := &MockServer{
		ListenAndServeFunc: func() error { return http.ErrServerClosed },
	}

	err := launcherFor(mockServer, func() { released = true }).Run(context.Background(), mockConfig, noopDBFactory)

	assert.NoError(t, err)
	assert.True(t, released)
}

func TestLauncher_Run_InitializeAppError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)

	launcher := &Launcher{
		AppInitializer: func(ctx context.Context, configService config.Configuration, newDB dbFactory) (Server, func(), error) {
			return nil, nil, errors.New("init error")
		},
	}

	err := launcher.Run(context.Background(), mockConfig, noopDBFactory)

	assert.Error(t, err)
	assert.Equal(t, "init error", err.Error())
}

func TestLauncher_Run_ServerRunError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	released := false
	mockServer := &MockServer{
		ListenAndServeFunc: func() error { return errors.New("run error") },
	}

	err := launcherFor(mockServer, func() { released = true }).Run(context.Background(), mockConfig, noopDBFactory)

	assert.Error(t, err)
	assert.Equal(t, "run error", err.Error())
	assert.True(t, released)
}

func TestLauncher_Run_ShutsDownWhenContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	var events []string
	serverClosed := make(chan struct{})
	mockServer := &MockServer{
		ListenAndServeFunc: func() error {
			<-serverClosed
			return http.ErrServerClosed
		},
		ShutdownFunc: func(ctx context.Context) error {
			events = append(events, "shutdown")
			close(serverClosed)
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := launcherFor(mockServer, func() { events = append(events, "release") }).Run(ctx, mockConfig, noopDBFactory)

	assert.NoError(t, err)
	assert.Equal(t, []string{"shutdown", "release"}, events)
}

func TestLauncher_Run_ReturnsShutdownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockConfig := mockconfig.NewMockConfiguration(ctrl)
	released := false
	serverBlocked := make(chan struct{})
	t.Cleanup(func() { close(serverBlocked) })
	mockServer := &MockServer{
		ListenAndServeFunc: func() error {
			<-serverBlocked
			return nil
		},
		ShutdownFunc: func(ctx context.Context) error {
			return errors.New("shutdown error")
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := launcherFor(mockServer, func() { released = true }).Run(ctx, mockConfig, noopDBFactory)

	assert.EqualError(t, err, "shutdown error")
	assert.True(t, released)
}

func TestNewHTTPServer_SetsTimeouts(t *testing.T) {
	server := newHTTPServer(http.NewServeMux())

	assert.Equal(t, ":8080", server.Addr)
	assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	assert.Equal(t, 60*time.Second, server.ReadTimeout)
	assert.Equal(t, 90*time.Second, server.WriteTimeout)
	assert.Equal(t, 120*time.Second, server.IdleTimeout)
}

func TestNewPaymentRegistry(t *testing.T) {
	tests := []struct {
		name   string
		config config.PaymentsConfig
		want   []string
	}{
		{name: "sandbox disabled", config: config.PaymentsConfig{SandboxSecret: "s3cret"}, want: []string{}},
		{name: "sandbox enabled without a secret", config: config.PaymentsConfig{SandboxEnabled: true}, want: []string{}},
		{name: "sandbox enabled with a secret", config: config.PaymentsConfig{SandboxEnabled: true, SandboxSecret: "s3cret"}, want: []string{"sandbox"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := newPaymentRegistry(context.Background(), tt.config, config.NewZerologService())
			assert.Equal(t, tt.want, registry.Codes())
		})
	}
}
