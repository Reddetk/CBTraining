package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type TXRequest struct {
	TXID                   string `json:"TXID"`
	Status                 string `json:"Status"`
	Amount                 string `json:"Amount"`
	Currency               string `json:"Currency"`
	EndToEndIdentification string `json:"EndToEndIdentification"`
	TransactionType        string `json:"TransactionType"`
	DebtorIBAN             string `json:"DebtorIBAN"`
	CreditorIBAN           string `json:"CreditorIBAN"`
	Metadata               string `json:"Metadata"`
}

type PaymentResult struct {
	TXID     string `json:"TXID"`
	Result   string `json:"Result"`
	Metadata string `json:"Metadata"`
}

func main() {
	brokers := flag.String("brokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka broker address")
	topicReq := flag.String("topic-request", getEnv("KAFKA_TOPIC_REQUEST", "payment.processing.request"), "Input topic")
	topicResp := flag.String("topic-response", getEnv("KAFKA_TOPIC_RESPONSE", "payment.processing.response"), "Output topic")
	groupID := flag.String("group", getEnv("KAFKA_CONSUMER_GROUP", "stub-processing-service"), "Kafka consumer group")
	delayMin := flag.Int("delay-min", 2, "Min processing delay in seconds")
	delayMax := flag.Int("delay-max", 7, "Max processing delay in seconds")
	failureRate := flag.Float64("failure-rate", 0.10, "Failure probability (0.0 - 1.0)")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("config",
		"brokers", *brokers,
		"topic_request", *topicReq,
		"topic_response", *topicResp,
		"group", *groupID,
		"delay_min", *delayMin,
		"delay_max", *delayMax,
		"failure_rate", *failureRate,
	)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{*brokers},
		Topic:   *topicReq,
		GroupID: *groupID,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:     kafka.TCP(*brokers),
		Balancer: &kafka.Hash{},
	}
	defer writer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger.Info("stub processing service started")

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("shutting down")
				return
			}
			logger.Error("failed to fetch message", "err", err)
			continue
		}

		go func(msg kafka.Message) {
			var req TXRequest
			if err := json.Unmarshal(msg.Value, &req); err != nil {
				logger.Error("failed to unmarshal", "err", err)
				return
			}

			delay := time.Duration(*delayMin+rand.Intn(*delayMax-*delayMin+1)) * time.Second
			logger.Info("processing", "txid", req.TXID, "delay", delay)
			time.Sleep(delay)

			result := "completed"
			metadata := ""
			if rand.Float64() < *failureRate {
				result = "failed"
				metadata = "simulated processing failure"
			}

			payload, _ := json.Marshal(PaymentResult{
				TXID:     req.TXID,
				Result:   result,
				Metadata: metadata,
			})

			if err := writer.WriteMessages(ctx, kafka.Message{
				Topic: *topicResp,
				Key:   msg.Key,
				Value: payload,
			}); err != nil {
				logger.Error("failed to write response", "txid", req.TXID, "err", err)
				return
			}

			if err := reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("failed to commit", "txid", req.TXID, "err", err)
				return
			}

			logger.Info("processed", "txid", req.TXID, "result", result)
		}(msg)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
