// Package inport defines the interfaces for the use cases related to payment management
package inport

import "context"

type PayManager interface {
	PaymentCMD(ctx context.Context, requ *PaymentRequest) (*TXConfirmation, error)
	StorePaymentResult(ctx context.Context, payRes *PaymentResult) error
	Shutdown()
}

type PaymentRequest struct {
	Amount                 string             `json:"amount" binding:"required"`
	Currency               string             `json:"currency" binding:"required"`
	EndToEndIdentification string             `json:"endToEndIdentification" binding:"required"`
	TransactionType        string             `json:"transactionType" binding:"required"`
	Debtor                 *PaymentPartyDTO   `json:"debtor" binding:"required"`
	DebtorPacc             *PaymentAccountDTO `json:"debtorPacc" binding:"required"`
	Creditor               *PaymentPartyDTO   `json:"creditor" binding:"required"`
	CreditorPacc           *PaymentAccountDTO `json:"creditorPacc" binding:"required"`
}

type PaymentPartyDTO struct {
	BIC               string `json:"bic" binding:"required"`
	Role              string `json:"role" binding:"required"`
	ContryOfResidence string `json:"contryOfResidence" binding:"required"`
	Name              string `json:"name" binding:"required"`
}

type PaymentAccountDTO struct {
	IBAN       string `json:"iban" binding:"required"`
	AccCurency string `json:"accCurency" binding:"required"`
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
