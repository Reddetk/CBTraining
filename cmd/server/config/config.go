// Package config
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Reddetk/CBTraining/logger"
)

type Config struct {
	HTTP     HTTPConfig
	DB       DBConfig
	Kafka    KafkaConfig
	Outbox   OutboxConfig
	Dispatch DispatchConfig
	Shutdown ShutdownConfig
	log      logger.Logger
}

type HTTPConfig struct {
	Addr string // APP_PORT, default ":8080"
}

type DBConfig struct {
	Host           string // DB_HOST
	Port           string // DB_PORT, default "5432"
	User           string // DB_USER
	Password       string // DB_PASS
	Name           string // DB_NAME
	SSL            string // DB_SSL, default "disable"
	MaxConnections int    // DB_MAX_CONNECTIONS, default 100
	SharedBuffers  string // DB_SHARED_BUFFERS, default "128MB"
	MaxWalSize     string // DB_MAX_WAL_SIZE, default "1GB"
	LogMinDuration int    // DB_LOG_MIN_DURATION ms, default 200
	LocalPort      string // DB_LOCAL_PORT
}

// DSN собирает строку подключения из полей структуры.
// Пример: postgres://user:pass@localhost:5432/payments?sslmode=disable
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password,
		d.Host, d.Port,
		d.Name, d.SSL,
	)
}

type KafkaConfig struct {
	Host              string        // KAFKA_HOST (берём из KAFKA_BROKERS или отдельно)
	Port              string        // KAFKA_PORT, default "9092"
	ControllerPort    string        // KAFKA_CONTROLLER_PORT, default "9093"
	ClusterID         string        // KAFKA_CLUSTER_ID
	Partitions        int           // KAFKA_PARTITIONS, default 3
	ReplicationFactor int           // KAFKA_REPLICATION_FACTOR, default 1
	AutoCreateTopics  bool          // KAFKA_AUTO_CREATE_TOPICS, default true
	RequestTopic      string        // KAFKA_REQUEST_TOPIC
	ResponseTopic     string        // KAFKA_RESPONSE_TOPIC
	ConsumerGroup     string        // KAFKA_CONSUMER_GROUP
	StartOffset       int64         // kafka.LastOffset = -1
	CommitInterval    time.Duration // KAFKA_COMMIT_INTERVAL, default 1s
}

// Brokers возвращает slice адресов брокеров для kafka-go.
func (k KafkaConfig) Brokers() []string {
	return []string{fmt.Sprintf("%s:%s", k.Host, k.Port)}
}

type OutboxConfig struct {
	BatchSize int           // OUTBOX_BATCH_SIZE, default 10
	Interval  time.Duration // OUTBOX_INTERVAL, default 5s
	Workers   int           // OUTBOX_WORKERS, default 1
}

type DispatchConfig struct {
	MaxWorkers int // MAX_WORKERS, default 5
}

type ShutdownConfig struct {
	Timeout time.Duration // SHUTDOWN_TIMEOUT, default 30s
}

// Load читает все переменные окружения и собирает Config.
// Падает с паникой если обязательные переменные не заданы.
func Load() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Addr: ":" + getEnv("APP_PORT", "8080"),
		},
		DB: DBConfig{
			Host:           requireEnv("DB_HOST"),
			Port:           getEnv("DB_PORT", "5432"),
			User:           requireEnv("DB_USER"),
			Password:       requireEnv("DB_PASS"),
			Name:           requireEnv("DB_NAME"),
			SSL:            getEnv("DB_SSL", "disable"),
			MaxConnections: getInt("DB_MAX_CONNECTIONS", 100),
			SharedBuffers:  getEnv("DB_SHARED_BUFFERS", "128MB"),
			MaxWalSize:     getEnv("DB_MAX_WAL_SIZE", "1GB"),
			LogMinDuration: getInt("DB_LOG_MIN_DURATION", 200),
			LocalPort:      getEnv("DB_LOCAL_PORT", "5432"),
		},
		Kafka: KafkaConfig{
			Host:              requireEnv("KAFKA_HOST"),
			Port:              getEnv("KAFKA_PORT", "9092"),
			ControllerPort:    getEnv("KAFKA_CONTROLLER_PORT", "9093"),
			ClusterID:         getEnv("KAFKA_CLUSTER_ID", ""),
			Partitions:        getInt("KAFKA_PARTITIONS", 3),
			ReplicationFactor: getInt("KAFKA_REPLICATION_FACTOR", 1),
			AutoCreateTopics:  getBool("KAFKA_AUTO_CREATE_TOPICS", true),
			RequestTopic:      getEnv("KAFKA_REQUEST_TOPIC", "payment.processing.request"),
			ResponseTopic:     getEnv("KAFKA_RESPONSE_TOPIC", "payment.processing.response"),
			ConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP", "payment-service-consumer"),
			StartOffset:       -1, // kafka.LastOffset
			CommitInterval:    getDuration("KAFKA_COMMIT_INTERVAL", 1*time.Second),
		},
		Outbox: OutboxConfig{
			BatchSize: getInt("OUTBOX_BATCH_SIZE", 10),
			Interval:  getDuration("OUTBOX_INTERVAL", 5*time.Second),
			Workers:   getInt("OUTBOX_WORKERS", 1),
		},
		Dispatch: DispatchConfig{
			MaxWorkers: getInt("MAX_WORKERS", 5),
		},
		Shutdown: ShutdownConfig{
			Timeout: getDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
	}
}

// ──────────────────────────────────────────────
// helpers
// ──────────────────────────────────────────────

func requireEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		panic(fmt.Sprintf("config: required env variable %q is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
