ENV        ?= local
MIGRATIONS := ./migrations

# Load env file
ifneq (,$(wildcard .env.$(ENV)))
  include .env.$(ENV)
  export
endif

# ─────────────────────────────────────────
# Dev lifecycle
# ─────────────────────────────────────────

.PHONY: dev
dev: infra-up migrate-up swagger dev-run  ## local: infra -> migrate -> swagger -> go run server

.PHONY: dev-run
dev-run:  ## Run server with 'go run' (requires infra running)
	go run ./cmd/server

# ─────────────────────────────────────────
# Test lifecycle
# ─────────────────────────────────────────

.PHONY: test
test: infra-up migrate-up swagger unitest-run dev-run  ## local: infra -> migrate -> swagger -> tests -> server

.PHONY: unitest-run
unitest-run:  ## Run unit tests
	-go test -tags=testing ./core -v

.PHONY: mantest-run
mantest-run:  ## Generate and send curl request
	go run ./Test/main.go

# ─────────────────────────────────────────
# Up
# ─────────────────────────────────────────

.PHONY: up
up: infra-up migrate-up server-up  ## docker infra -> migrate -> docker server

# ─────────────────────────────────────────
# Build
# ─────────────────────────────────────────

.PHONY: build
build:  ## Compile server and migrate binaries
	go build -o .\bin\migrate.exe ./cmd/migrate
	go build -o .\bin\server.exe  ./cmd/server

.PHONY: build-stub
build-stub:  ## Compile stub exe
	go build -o bin/ProcessService.exe ./ProcessService/main.go

.PHONY: build-all
build-all: build build-stub  ## Compile server, migrate and stub binaries

.PHONY: build-migration
build-migration:  ## Compile migrate binary
	go build -o .\bin\migrate.exe ./cmd/migrate

# ─────────────────────────────────────────
# Migrations
# ─────────────────────────────────────────

.PHONY: migrate-up
migrate-up: build-migration  ## Apply all migrations
	.\bin\migrate.exe -path $(MIGRATIONS) \
	  -database "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)" \
	  -cmd up

.PHONY: migrate-down
migrate-down: build-migration  ## Roll back last migration
	.\bin\migrate.exe -path $(MIGRATIONS) \
	  -database "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)" \
	  -cmd down

.PHONY: migrate-force
migrate-force: build-migration  ## Force migration version: make migrate-force VERSION=1
	.\bin\migrate.exe -path $(MIGRATIONS) \
	  -database "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)" \
	  -cmd force -version $(VERSION)

# ─────────────────────────────────────────
# Swagger
# ─────────────────────────────────────────

.PHONY: swagger
swagger:  ## Generate docs/ from annotations in cmd/server/main.go
	swag init -g cmd/server/main.go -o docs/

# ─────────────────────────────────────────
# Run
# ─────────────────────────────────────────

.PHONY: run
run: build  ## Start server binary
	.\bin\server.exe

# ─────────────────────────────────────────
# Stub
# ─────────────────────────────────────────

BROKERS      ?= localhost:9092
TOPIC_REQ    ?= payment.processing.request
TOPIC_RESP   ?= payment.processing.response
FAILURE_RATE ?= 0.10
DELAY_MIN    ?= 2
DELAY_MAX    ?= 7

.PHONY: stub-run
stub-run: build-stub  ## Run stub locally (override vars via make stub-run BROKERS=...)
	bin/ProcessService.exe \
	  -brokers          $(BROKERS) \
	  -topic-request    $(TOPIC_REQ) \
	  -topic-response   $(TOPIC_RESP) \
	  -failure-rate     $(FAILURE_RATE) \
	  -delay-min        $(DELAY_MIN) \
	  -delay-max        $(DELAY_MAX)

.PHONY: stub-logs
stub-logs:  ## Tail stub container logs
	docker-compose --env-file .env.$(ENV) logs -f stub

# ─────────────────────────────────────────
# Server
# ─────────────────────────────────────────

.PHONY: server-up
server-up:  ## Start server container
	docker-compose --env-file .env.$(ENV) up -d --build server

# ─────────────────────────────────────────
# Infra
# ─────────────────────────────────────────

.PHONY: infra-up
infra-up:  ## Start infrastructure containers (Postgres + Kafka + Stub)
	docker-compose --env-file .env.$(ENV) up -d --build postgres kafka stub

.PHONY: infra-down
infra-down:  ## Stop infrastructure containers
	docker-compose --env-file .env.$(ENV) down

# ─────────────────────────────────────────
# Docker (full)
# ─────────────────────────────────────────

.PHONY: docker-up
docker-up:  ## Full Docker environment: infra + migrate + server
	docker-compose --env-file .env.$(ENV) up --build

.PHONY: docker-down
docker-down:  ## Stop and remove all Docker containers
	docker-compose --env-file .env.$(ENV) down

# ─────────────────────────────────────────
# Clean
# ─────────────────────────────────────────

.PHONY: clean
clean:  ## Remove build artifacts
	rm -rf bin/ coverage.out

# ─────────────────────────────────────────
# Help
# ─────────────────────────────────────────

.PHONY: help
help:  ## Show this help
	@powershell -Command " \
	  Get-Content 'Makefile' | \
	  Select-String '^[a-zA-Z_-]+:.*##' | \
	  ForEach-Object { \
	    $$parts = $$_ -split ':.*##'; \
	    Write-Host ('  {0,-20} {1}' -f $$parts[0].Trim(), $$parts[1].Trim()) \
	  }"