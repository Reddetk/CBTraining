package main

import (
	"context"
	"errors"
	nhttp "net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Reddetk/CBTraining/adapters/primary/http"
	kfconsumer "github.com/Reddetk/CBTraining/adapters/primary/kafka"
	kfproducer "github.com/Reddetk/CBTraining/adapters/secondary/kafka"
	"github.com/Reddetk/CBTraining/adapters/secondary/outbox"
	"github.com/Reddetk/CBTraining/adapters/secondary/postgres"
	"github.com/Reddetk/CBTraining/cmd/server/config"
	"github.com/Reddetk/CBTraining/core"
	"github.com/Reddetk/CBTraining/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// @title           Payment Service API
// @version         1.0
// @description     Async payment processing service
// @host            localhost:8080
// @BasePath        /
func main() {
	// ===== 0. Загрузка конфигурации =====
	cfg := config.Load()

	// ===== 1. Инициализация логгера =====
	log := logger.New()
	log.Info("payment service starting")

	// ===== 2. Инициализация контекста =====
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ===== 3. Инициализация БД =====
	dbURL := cfg.DB.DSN()
	log.Info("connecting to database", zap.String("dsn", cfg.DB.Host+":"+cfg.DB.Port))

	dbPool, err := pgxpool.New(rootCtx, dbURL)
	if err != nil {
		log.Error("failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	defer dbPool.Close()

	// Проверка соединения
	if err := dbPool.Ping(rootCtx); err != nil {
		log.Error("database ping failed", zap.Error(err))
		os.Exit(1)
	}
	log.Info("database connected successfully")

	// ===== 4. Инициализация вторичных адаптеров =====

	// PostgreSQL Repository (реализует PaymentRepo и PendingRepo)
	log.Info("initializing PostgreSQL repository")
	paymentRepo := postgres.New(dbPool, log)

	// Kafka Producer для outbox worker
	kafkaBrokers := cfg.Kafka.Brokers()
	log.Info("initializing Kafka producer", zap.String("brokers", cfg.Kafka.Host+":"+cfg.Kafka.Port))

	kafkaWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers: kafkaBrokers,
	})
	defer kafkaWriter.Close()

	kafkaProducer := kfproducer.NewWriterProducer(kafkaWriter, log)

	// Outbox Store
	log.Info("initializing outbox store")
	outboxStore := outbox.NewStore(dbPool, log)

	// Outbox Worker
	log.Info("initializing outbox worker",
		zap.Int("batch_size", cfg.Outbox.BatchSize),
		zap.Duration("interval", cfg.Outbox.Interval))

	outboxWorker := outbox.New(outboxStore, *kafkaProducer, cfg.Outbox.BatchSize, cfg.Outbox.Interval, log)

	// ===== 5. Инициализация сервисов =====

	// PaymentManagerService
	log.Info("initializing PaymentManagerService", zap.Int("max_workers", cfg.Dispatch.MaxWorkers))

	paymentManagerSvc, err := core.NewPaymentManagerService(
		paymentRepo,
		paymentRepo,
		paymentRepo,
		rootCtx,
		cfg.Dispatch.MaxWorkers,
		log,
	)
	if err != nil {
		log.Error("failed to create PaymentManagerService", zap.Error(err))
		os.Exit(1)
	}

	// PaymentAuditService
	log.Info("initializing PaymentAuditService")
	auditSvc := core.NewPaymentAuditService(paymentRepo, log)

	// ===== 6. Инициализация первичных адаптеров =====

	// HTTP Handler с внедрением зависимостей
	log.Info("initializing HTTP handler")
	httpHandlers := http.NewHandler(paymentManagerSvc, auditSvc)

	// HTTP Router
	ginEngine := gin.Default()
	http.RegisterRoutes(ginEngine, httpHandlers)

	// HTTP Server
	log.Info("initializing HTTP server", zap.String("addr", cfg.HTTP.Addr))

	// Kafka Consumer Reader
	log.Info("initializing Kafka consumer",
		zap.String("group", cfg.Kafka.ConsumerGroup),
		zap.String("topic", cfg.Kafka.ResponseTopic))

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaBrokers,
		Topic:          cfg.Kafka.ResponseTopic,
		GroupID:        cfg.Kafka.ConsumerGroup,
		StartOffset:    kafka.LastOffset,
		CommitInterval: cfg.Kafka.CommitInterval,
	})
	defer kafkaReader.Close()

	// Kafka Consumer Adapter
	kafkaConsumer := kfconsumer.NewConsumer(paymentManagerSvc, kafkaReader, log)

	// ===== 7. Запуск фоновых воркеров =====

	// Запуск outbox worker
	log.Info("starting outbox worker")
	go func() {
		outboxWorker.Run(rootCtx)
	}()

	// Запуск Kafka consumer
	log.Info("starting Kafka consumer")
	go func() {
		if err := kafkaConsumer.Run(rootCtx); err != nil {
			log.Error("Kafka consumer error", zap.Error(err))
		}
	}()

	err = paymentManagerSvc.PandingRecovery(rootCtx)
	if err != nil {
		log.Error("Panding recovery error", zap.Error(err))
	}

	// ===== 8. Запуск HTTP сервера =====
	httpSrv := &nhttp.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: ginEngine,
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, nhttp.ErrServerClosed) {
			log.Error("HTTP server error", zap.Error(err))
		}
	}()

	// ===== 9. Graceful Shutdown =====
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
	defer shutdownCancel()

	// останавливаем HTTP — новые запросы не принимаем,
	log.Info("shutting down HTTP server")
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("closing Kafka consumer")
	_ = kafkaReader.Close()

	log.Info("cancelling root context")
	cancel()

	// 4. Ждём всех воркеров Dispatcher через Wg.Wait()
	log.Info("waiting for all dispatcher workers")
	paymentManagerSvc.Shutdown() // → ps.dispatcher.Wg.Wait()

	log.Info("payment service stopped cleanly")
}
