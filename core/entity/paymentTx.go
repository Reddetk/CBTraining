package entity

import (
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	"github.com/shopspring/decimal"
)

type PaymentTX struct {
	TXID                   string
	status                 valobj.Status
	amount                 decimal.Decimal
	currency               valobj.Currency
	endToEndIdentification valobj.EndToEndIdentification
	transactionType        string
	debitorIBAN            string
	creditorIBAN           string
	metadata               valobj.Metadata
}

func NewPaymentTX(
	TXID string, status valobj.Status, amount decimal.Decimal, currency valobj.Currency,
	endToEndIdentification valobj.EndToEndIdentification, transactionType string, debitorIBAN string,
	creditorIBAN string, metadata valobj.Metadata,
) (*PaymentTX, error) {
	if err := validateIBAN(debitorIBAN); err != nil {
		return nil, err
	}
	if err := validateIBAN(creditorIBAN); err != nil {
		return nil, err
	}
	return &PaymentTX{
		TXID:                   TXID,
		status:                 status,
		amount:                 amount,
		currency:               currency,
		endToEndIdentification: endToEndIdentification,
		transactionType:        transactionType,
		debitorIBAN:            debitorIBAN,
		creditorIBAN:           creditorIBAN,
		metadata:               metadata,
	}, nil
}
