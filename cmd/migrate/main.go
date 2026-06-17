package main

import (
	"errors"
	"flag"
	"os"
	"strconv"

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
	command := flag.String("cmd", "up", "command: up | down | force")
	version := flag.String("version", "", "target version for force command")
	flag.Parse()

	dbURL := *database
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		log.Fatal("database URL is not set")
	}

	m, err := migrate.New("file://"+*path, dbURL)
	if err != nil {
		log.Fatal("failed to init migrate", zap.Error(err))
	}
	defer m.Close()

	switch *command {
	case "up":
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no migrations to apply")
			return
		}
		if err != nil {
			log.Fatal("migration up failed", zap.Error(err))
		}
		log.Info("migrations applied successfully")

	case "down":
		err = m.Steps(-1)
		if err != nil {
			log.Fatal("migration down failed", zap.Error(err))
		}
		log.Info("rolled back one migration")

	case "force":
		if *version == "" {
			log.Fatal("force requires -version flag")
		}
		v, err := strconv.Atoi(*version)
		if err != nil {
			log.Fatal("invalid version", zap.String("version", *version))
		}
		if err := m.Force(v); err != nil {
			log.Fatal("force failed", zap.Error(err))
		}
		log.Info("forced migration version", zap.Int("version", v))

	default:
		log.Fatal("unknown command", zap.String("cmd", *command))
	}
}
