package outport

import "context"

type PendingRepo interface {
	LoadPendingPayments(ctx context.Context) ([]TXRecord, error)
}
