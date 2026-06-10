package outports

import "context"

type PaymentRepo interface {
	UpserPayment(ctx context.Context, paymentData TXRecord) error
}

type TXRecord struct {
	TXID                   string
	Status                 string
	Amount                 string
	Currency               string
	EndToEndIdentification string
	DebitorPacc            *PARecord
	CreditorPacc           *PARecord
	Metadata               string
}

type PartyRecord struct {
	BIC               string
	Role              string
	ContryOfResidence string
	Name              string
}

type PARecord struct {
	IBAN       string
	AccCurency string
	Party      *PartyRecord
}
