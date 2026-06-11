# ---- build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN swag init -g cmd/server/main.go -o docs/
RUN go build -o bin/server  ./cmd/server
RUN go build -o bin/migrate ./cmd/migrate

# ---- runtime stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/        ./
COPY --from=builder /app/migrations/ ./migrations/
COPY --from=builder /app/config/     ./config/
COPY --from=builder /app/docs/       ./docs/