// Package kfconsumer implements the Kafka consumer adapter for consuming messages from Kafka topics
package kfconsumer

import (
	"context"
	"encoding/json"

	"github.com/Reddetk/CBTraining/logger"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	payManager inport.PayManager
	consumer   *kafka.Reader
	logger     logger.Logger
}

func NewConsumer(pM inport.PayManager, reader *kafka.Reader, log logger.Logger) *Consumer {
	return &Consumer{
		payManager: pM,
		consumer:   reader,
		logger:     log,
	}
}

// Run starts the Kafka consumer loop to read messages and process payment results
func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("kfconsumer: started payment.processing.response consumer")

	for {
		m, err := c.consumer.FetchMessage(ctx)
		if err != nil {

			if ctx.Err() != nil {
				c.logger.Info("kfconsumer: context cancelled, stopping consumer")
				return nil
			}
			c.logger.Error("kfconsumer: read message error", zap.Error(err))
			continue
		}

		var pr inport.PaymentResult
		if err := json.Unmarshal(m.Value, &pr); err != nil {
			c.logger.Error("kfconsumer: invalid payload",
				zap.Error(err),
			)
			continue
		}

		c.logger.Info("kfconsumer: received payment result",
			zap.String("txid", pr.TXID), zap.String("result", pr.Result),
		)

		if err := c.payManager.StorePaymentResult(ctx, &pr); err != nil {
			c.logger.Error("kfconsumer: StorePaymentResult failed",
				zap.String("txid", pr.TXID),
				zap.Error(err),
			)
			continue
		}

		if err := c.consumer.CommitMessages(ctx, m); err != nil {
			c.logger.Error("failed to commit", zap.Error(err))
		}

		c.logger.Info("kfconsumer: payment result stored",
			zap.String("txid", pr.TXID),
		)
	}
}

// INB191683861
