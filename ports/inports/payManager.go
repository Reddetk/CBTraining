// Package inport defines the interfaces for the use cases related to payment management
package inport

import "context"

type PayManager interface {
	PaymentCMD(ctx context.Context, requ *PaymentRequest) (*TXConfirmation, error)
	StorePaymentResult(ctx context.Context, payRes *PaymentResult) error
	Shutdown()
}

type PaymentRequest struct {
	Amount                 string             `json:"amount"`
	Currency               string             `json:"currency"`
	EndToEndIdentification string             `json:"endToEndIdentification"`
	TransactionType        string             `json:"transactionType"`
	Debitor                *PaymentPartyDTO   `json:"debitor"`
	DebitorPacc            *PaymentAccountDTO `json:"debitorPacc"`
	Creditor               *PaymentPartyDTO   `json:"creditor"`
	CreditorPacc           *PaymentAccountDTO `json:"creditorPacc"`
}

type PaymentPartyDTO struct {
	BIC               string `json:"bic"`
	Role              string `json:"role"`
	ContryOfResidence string `json:"contryOfResidence"`
	Name              string `json:"name"`
}

type PaymentAccountDTO struct {
	IBAN       string `json:"iban"`
	AccCurency string `json:"accCurency"`
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
