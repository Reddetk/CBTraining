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
	debitorPaccID          string
	creditorPaccID         string
	metadata               valobj.Metadata
}
