package inport

import "context"

type PayAuditor interface {
	GetPaymentInfo(ctx context.Context, TXID string) (*PaymentInfo, error)
}

type PaymentInfo struct {
	TXID            string
	Status          string
	CreditorAccount *PaymentAccountDTO
	DebtorAccount  *PaymentAccountDTO
	Metadata        string
}
