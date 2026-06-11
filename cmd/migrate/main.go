package main

import (
	"flag"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	path := flag.String("path", "migrations", "path to migrations folder")
	database := flag.String("database", "", "database URL")
	flag.Parse()

	dbURL := *database
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}

	m, err := migrate.New("file://"+*path, dbURL)
	if err != nil {
		log.Fatal("failed to init migrate", zap.Error(err))
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("migration failed", zap.Error(err))
	}

	log.Info("migrations applied successfully")
}
