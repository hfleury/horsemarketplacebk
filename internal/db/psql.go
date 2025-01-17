package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hfleury/horsemarketplacebk/config"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type PsqlDB struct {
	Conn *sql.DB
	Logg zerolog.Logger
}

func NewPsqlDB(config *config.AllConfiguration, logger zerolog.Logger) (*PsqlDB, error) {
	connStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		config.Psql.Username,
		config.Psql.Password,
		config.Psql.DdName,
		config.Psql.Host,
		config.Psql.Port,
	)

	fmt.Println("Connecting to database with connection string:", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error().Err(err).Msg("Error pinging database")
		return nil, err
	}

	return &PsqlDB{
		Conn: db,
		Logg: logger,
	}, nil
}

func (p *PsqlDB) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	p.Logg.Trace().Msg("Starting new transaction")
	tx, err := p.Conn.BeginTx(ctx, nil)
	if err != nil {
		p.Logg.Error().Err(err).Msg("Error starting transaction")
		return nil, err
	}
	p.Logg.Trace().Msg("Transaction started successfully")
	return tx, nil
}

func (p *PsqlDB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	p.Logg.Debug().Str("query", query).Msg("Executing query")
	rows, err := p.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		p.Logg.Error().Err(err).Str("query", query).Msg("Error executing query")
		return nil, err
	}
	p.Logg.Debug().Str("query", query).Msg("Query executed successfully")
	return rows, nil
}

func (p *PsqlDB) Execute(ctx context.Context, query string, args ...any) (sql.Result, error) {
	p.Logg.Debug().Str("query", query).Msg("Executing execute query")
	result, err := p.Conn.ExecContext(ctx, query, args...)
	if err != nil {
		p.Logg.Error().Err(err).Str("query", query).Msg("Error executing execute query")
		return nil, err
	}
	p.Logg.Debug().Str("query", query).Msg("Execute query executed successfully")
	return result, nil
}

func (p *PsqlDB) Close() error {
	p.Logg.Trace().Msg("Closing database connection")
	err := p.Conn.Close()
	if err != nil {
		p.Logg.Error().Err(err).Msg("Error closing database connection")
		return err
	}
	p.Logg.Trace().Msg("Database connection closed successfully")
	return nil
}

func (p *PsqlDB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	p.Logg.Debug().Str("query", query).Msg("Executing query row")
	return p.Conn.QueryRowContext(ctx, query, args...)
}
