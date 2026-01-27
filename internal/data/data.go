package data

import (
	"database/sql"
	"storage/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

type Data struct {
	DB *sql.DB
}

func NewData(conf *conf.Data, logger log.Logger) (*Data, func(), error) {
	log := log.NewHelper(logger)

	db, err := sql.Open("postgres", conf.Database.Dsn)
	if err != nil {
		return nil, nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, nil, err
	}

	// AUTO MIGRATE
	if err := RunMigrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	cleanup := func() {
		log.Info("closing db")
		db.Close()
	}

	return &Data{DB: db}, cleanup, nil
}
