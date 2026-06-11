// Package outport defines the output ports for the payment processing system
package outport

import (
	"context"

	"github.com/shopspring/decimal"
)

type PaymentProcessor interface {
	ProcessTX(ctx context.Context, paymentData TXRequest) error
}

type TXRequest struct {
	TXID                   string
	Status                 string
	Amount                 decimal.Decimal
	Currency               string
	EndToEndIdentification string
	TransactionType        string
	DebitorIBAN            string
	CreditorIBAN           string
	Metadata               string
}
