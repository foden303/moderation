package data

import (
	"database/sql"
	"moderation/internal/conf"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrate(conf *conf.Data, db *sql.DB) error {
	// Create an instance of the Postgres driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	// Create a new migrate instance
	src, err := (&file.File{}).Open("internal/data/migrations")
	if err != nil {
		return err
	}
	// Create the migrate instance
	m, err := migrate.NewWithInstance(
		"file",
		src,
		conf.Database.Driver,
		driver,
	)
	if err != nil {
		return err
	}
	// Run the migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
