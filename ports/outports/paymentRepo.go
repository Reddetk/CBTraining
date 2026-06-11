package outport

import "context"

type PaymentRepo interface {
	InsertTX(ctx context.Context, paymentData TXRecord) error
	GetTXByID(ctx context.Context, TXID string) (*TXRecord, error)
	PersistProcessResult(ctx context.Context, TXID, result string) error
}

type TXRecord struct {
	TXID                   string
	Status                 string
	Amount                 string
	Currency               string
	EndToEndIdentification string
	TransactionType        string
	DebitorPacc            *PARecord
	CreditorPacc           *PARecord
	Metadata               string
}

type PARecord struct {
	IBAN       string
	AccCurency string
	Party      *PartyRecord
}

type PartyRecord struct {
	BIC               string
	Role              string
	ContryOfResidence string
	Name              string
}
