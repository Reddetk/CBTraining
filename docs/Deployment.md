# Deployment

## Project Structure for Deployment

```

cmd/
  server/          ← main binary: HTTP server + Dispatcher + Kafka consumer
    main.go
  migrate/         ← init binary: applies migrations and exits
    main.go
migrations/        ← SQL migration files (golang-migrate)
config/
  config.yaml      ← server configuration (semaphore size, timeouts, etc.)
.env.local         ← local environment (DB credentials, Kafka DSN)
.env.test          ← test environment
.env.example       ← template, committed to git
Taskfile.yml
Dockerfile
docker-compose.yml

```

## Local Development

**Prerequisites:** Go 1.25+, [Task](https://taskfile.dev), [swag](https://github.com/swaggo/swag)

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Postgres and Kafka must be running on ports defined in `.env.local`:

```bash
cp .env.example .env.local  # fill in credentials
```

**Start sequence:**

```bash
task dev          # migrate-up → swagger → run
```

