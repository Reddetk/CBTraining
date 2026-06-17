// Package kfproducer implements the Kafka producer adapter for sending messages to Kafka topics
package kfproducer

import (
	"context"

	"github.com/Reddetk/CBTraining/logger"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

type WriterProducer struct {
	w      *kafka.Writer
	logger logger.Logger
}

func NewWriterProducer(w *kafka.Writer, logger logger.Logger) *WriterProducer {
	return &WriterProducer{
		w:      w,
		logger: logger,
	}
}

func (p *WriterProducer) Send(ctx context.Context, msg Message) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: msg.Topic,
		Key:   msg.Key,
		Value: msg.Value,
	})
}

func (p *WriterProducer) Close() error {
	return p.w.Close()
}
