package entity

import (
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	outport "github.com/Reddetk/CBTraining/ports/outports"
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

func (tx *PaymentTX) ToRecord(
	depCur, credCur valobj.Currency,
	dBIC, cBIC string,
	drole, crole valobj.Role,
	dcoR, ccoR valobj.CountryOfResidence,
	dname, cname string,
) (outport.TXRecord, error) {
	dpacc, err := NewPaymentAccount(tx.debitorIBAN, depCur)
	if err != nil {
		return outport.TXRecord{}, err
	}
	cpacc, err := NewPaymentAccount(tx.creditorIBAN, credCur)
	if err != nil {
		return outport.TXRecord{}, err
	}

	cPacRec, err := cpacc.ToPaRecord(cBIC, crole, ccoR, cname)
	if err != nil {
		return outport.TXRecord{}, err
	}
	dPacRec, err := dpacc.ToPaRecord(dBIC, drole, dcoR, dname)
	if err != nil {
		return outport.TXRecord{}, err
	}

	return outport.TXRecord{
		TXID:                   tx.TXID,
		Status:                 string(tx.status),
		Amount:                 tx.amount.String(),
		Currency:               tx.currency.String(),
		EndToEndIdentification: tx.endToEndIdentification.String(),
		TransactionType:        tx.transactionType,
		DebitorPacc:            dPacRec,
		CreditorPacc:           cPacRec,
	}, nil
}
