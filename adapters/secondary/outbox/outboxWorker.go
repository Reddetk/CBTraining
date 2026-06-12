package outbox

import (
	"context"
	"encoding/json"
	"time"

	kfproducer "github.com/Reddetk/CBTraining/adapters/secondary/kafka"
	"github.com/Reddetk/CBTraining/logger"
	"go.uber.org/zap"
)

type Worker struct {
	store    *Store
	producer kfproducer.WriterProducer
	batch    int
	interval time.Duration
	logger   logger.Logger
}

func New(store *Store, producer kfproducer.WriterProducer, batch int, interval time.Duration, logger logger.Logger) *Worker {
	return &Worker{
		store:    store,
		producer: producer,
		batch:    batch,
		interval: interval,
		logger:   logger,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	events, err := w.store.FetchUnsent(ctx, w.batch)
	if err != nil {
		w.logger.Error("failed to fetch unsent events", zap.Error(err))
		return
	}
	if len(events) == 0 {
		return
	}

	var sentIDs []string

	for _, e := range events {
		var payload struct {
			TXID string `json:"txid"`
		}
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			w.logger.Error("invalid payload", zap.String("txid", e.TXID), zap.Error(err))
			continue
		}

		msg := kfproducer.Message{
			Topic: e.Topic,
			Key:   []byte(payload.TXID),
			Value: e.Payload,
		}

		if err := w.producer.Send(ctx, msg); err != nil {
			w.logger.Error("failed to send to kafka", zap.String("txid", e.TXID), zap.Error(err))
			continue
		}

		sentIDs = append(sentIDs, e.TXID)
	}

	if err := w.store.MarkSent(ctx, sentIDs); err != nil {
		w.logger.Error("failed to mark events as sent", zap.Error(err))
	}
}
