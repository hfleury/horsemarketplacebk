package db_test

import (
	"testing"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPsqlDB_ConfiguresConnectionPool(t *testing.T) {
	cfg := &config.AllConfiguration{
		Psql: config.PostgresConfig{
			Username: "test_user",
			Password: "test_pass",
			DdName:   "test_db",
			Host:     "localhost",
			Port:     "5432",
		},
	}

	psqlDB, err := db.NewPsqlDB(cfg, zerolog.Nop())
	require.NoError(t, err)
	defer psqlDB.Close()

	assert.Equal(t, 25, psqlDB.Conn.Stats().MaxOpenConnections)
}
