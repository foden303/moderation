package data

import (
	"context"
	"database/sql"
	"moderation/internal/conf"
	"moderation/internal/data/postgres/sqlc"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewRedisCache,
	NewBadwordRepo,
	NewBadImageRepo,
	NewTextModerator,
	NewImageModerator,
	NewVideoModerator,
)

// Data struct for db client
type Data struct {
	Pool    *pgxpool.Pool // pgxpool for sqlc (pgx/v5)
	Queries *sqlc.Queries // sqlc generated queries
	DB      *sql.DB       // database/sql for migrations
}

// NewData new a data instance
func NewData(conf *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)
	ctx := context.Background()
	// config pool
	pgxConfig, err := newPgxPoolConfig(conf)
	if err != nil {
		return nil, nil, err
	}
	// Connect with pgxpool for sqlc
	pool, err := pgxpool.New(ctx, pgxConfig.ConnString())
	if err != nil {
		return nil, nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, nil, err
	}

	// Also open database/sql for migrations
	db, err := sql.Open(conf.Database.Driver, conf.Database.Source)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}

	// auto migrate
	if err := RunMigrate(conf, db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Create sqlc queries instance
	queries := sqlc.New(pool)

	cleanup := func() {
		log.Info("closing db connections")
		pool.Close()
		db.Close()
	}

	return &Data{
		Pool:    pool,
		Queries: queries,
		DB:      db,
	}, cleanup, nil
}

// newPgxPoolConfig creates a pgxpool.Config from conf.Data
func newPgxPoolConfig(conf *conf.Data) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(conf.Database.Source)
	if err != nil {
		return nil, err
	}
	// Configure connection pool settings
	pool := conf.Database.Pool
	if pool.MaxOpenConns > 0 {
		cfg.MaxConns = pool.MaxOpenConns
	}
	if pool.MinIdleConns > 0 {
		cfg.MinConns = pool.MinIdleConns
	}
	if pool.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = time.Duration(pool.MaxConnLifetime) * time.Minute
	}
	if pool.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = time.Duration(pool.MaxConnIdleTime) * time.Minute
	}

	return cfg, nil
}
