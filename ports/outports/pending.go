package outports

import "context"

type PendingRepo interface {
	LoadPendingPayments(ctx context.Context) ([]TXRecord, error)
	Pub(ctx context.Context, TXID string) error
	Del(ctx context.Context, TXID string) error
}
