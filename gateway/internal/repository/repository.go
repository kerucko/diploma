package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/kerucko/diploma/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found in database")
)

type repository struct {
	conn    *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewConnection(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	dbPath := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	deadline := time.After(cfg.Timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn, err := pgxpool.New(ctx, dbPath)
			if err != nil {
				continue
			}
			if err = conn.Ping(ctx); err != nil {
				continue
			}
			log.Println("Successful database connection")
			return conn, nil

		case <-deadline:
			return nil, fmt.Errorf("unable to connect to database")
		}
	}
}
