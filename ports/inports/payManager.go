// Package inport defines the interfaces for the use cases related to payment management
package inport

import "context"

type PayManager interface {
	PaymentCMD(ctx context.Context, requ *PaymentRequest) (*TXConfirmation, error)
	StorePaymentResult(ctx context.Context, payRes *PaymentResult) error
	Shutdown()
}

type PaymentRequest struct {
	TXID                   string
	Status                 string
	Amount                 string
	Currency               string
	EndToEndIdentification string
	TransactionType        string
	Debitor                *PaymentPartyDTO
	DebitorPacc            *PaymentAccountDTO
	Creditor               *PaymentPartyDTO
	CreditorPacc           *PaymentAccountDTO
}

type PaymentPartyDTO struct {
	BIC               string
	Role              string
	ContryOfResidence string
	Name              string
}

type PaymentAccountDTO struct {
	IBAN       string
	AccCurency string
}

type PaymentResult struct {
	TXID     string
	Result   string
	Metadata string
}

type TXConfirmation struct {
	TXID     string
	Status   string
	Metadata string
}
