package inports

import "context"

type PayAuditor interface {
	GetPayment(ctx context.Context, TXID string) (*PaymentInfo, error)
}

type PaymentInfo struct {
	TXID            string
	Status          string
	CreditorAccount *PaymentAccountDTO
	DebitorAccount  *PaymentAccountDTO
	Metadata        string
}
