package main

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
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

	server, err := initializeApp(ctx, mockConfig)

	assert.NoError(t, err)
	assert.NotNil(t, server)
}
