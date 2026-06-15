// Package outbox
package outbox

import (
	"context"
	"time"

	"github.com/Reddetk/CBTraining/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Event struct {
	TXID      string
	Topic     string
	Payload   []byte
	CreatedAt time.Time
	SentAt    *time.Time
}

type Store struct {
	db     *pgxpool.Pool
	logger logger.Logger
}

func NewStore(db *pgxpool.Pool, logger logger.Logger) *Store {
	return &Store{db: db, logger: logger}
}

func (s *Store) FetchUnsent(ctx context.Context, limit int) ([]Event, error) {
	rows, err := s.db.Query(ctx, `
        SELECT txid, topic, payload, created_at, sent_at
        FROM outbox
        WHERE sent_at IS NULL
        ORDER BY created_at ASC
        LIMIT $1
		FOR UPDATE SKIP LOCKED
    `, limit)
	if err != nil {
		s.logger.Error("failed to fetch unsent events", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var sentAt *time.Time

		if err := rows.Scan(&e.TXID, &e.Topic, &e.Payload, &e.CreatedAt, &sentAt); err != nil {
			s.logger.Error("failed to scan event", zap.Error(err))
			return nil, err
		}

		e.SentAt = sentAt
		events = append(events, e)
	}

	return events, rows.Err()
}

// MarkSent помечает события как отправленные, выставляя sent_at = NOW().
func (s *Store) MarkSent(ctx context.Context, TXids []string) error {
	if len(TXids) == 0 {
		return nil
	}

	_, err := s.db.Exec(ctx, `
        UPDATE outbox
        SET sent_at = NOW()
        WHERE txid = ANY($1)
    `, TXids)
	if err != nil {
		s.logger.Error("failed to mark events as sent", zap.Error(err))
	}
	return err
}
