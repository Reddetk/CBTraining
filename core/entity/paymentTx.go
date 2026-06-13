package entity

import (
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
	"github.com/shopspring/decimal"
)

type PaymentTX struct {
	TXID                   string
	Status                 valobj.Status
	Amount                 decimal.Decimal
	Currency               valobj.Currency
	EndToEndIdentification valobj.EndToEndIdentification
	TransactionType        string
	DebitorPacc            *PaymentAccount
	CreditorPacc           *PaymentAccount
	Metadata               valobj.Metadata
}

func NewPaymentTX(
	TXID string, status valobj.Status, amount decimal.Decimal, currency valobj.Currency,
	endToEndIdentification valobj.EndToEndIdentification, transactionType string, debitorPacc *PaymentAccount,
	creditorPacc *PaymentAccount, metadata valobj.Metadata,
) (*PaymentTX, error) {
	return &PaymentTX{
		TXID:                   TXID,
		Status:                 status,
		Amount:                 amount,
		Currency:               currency,
		EndToEndIdentification: endToEndIdentification,
		TransactionType:        transactionType,
		DebitorPacc:            debitorPacc,
		CreditorPacc:           creditorPacc,
		Metadata:               metadata,
	}, nil
}

func NewPaymentTXFromDTO(pReq inport.PaymentRequest) (*PaymentTX, error) {
	st, err := valobj.ValStatus(pReq.Status)
	if err != nil {
		return nil, err
	}
	am, err := decimal.NewFromString(pReq.Amount)
	if err != nil {
		return nil, err
	}
	cur, err := valobj.ParseCurrency(pReq.Currency)
	if err != nil {
		return nil, err
	}
	eteID, err := valobj.ParseEndToEndIdentification(pReq.EndToEndIdentification)
	if err != nil {
		return nil, err
	}

	credPA, err := NewPAFromDTO(*pReq.CreditorPacc, *pReq.Creditor)
	if err != nil {
		return nil, err
	}
	debPA, err := NewPAFromDTO(*pReq.DebitorPacc, *pReq.Debitor)
	if err != nil {
		return nil, err
	}

	return NewPaymentTX(
		pReq.TXID,
		st,
		am,
		cur,
		eteID,
		pReq.TransactionType,
		debPA,
		credPA,
		valobj.NewMetadataNow(),
	)
}

func RecoverPaymentTXFromRec(pRec outport.TXRecord) (*PaymentTX, error) {
	st, err := valobj.ValStatus(pRec.Status)
	if err != nil {
		return nil, err
	}
	am, err := decimal.NewFromString(pRec.Amount)
	if err != nil {
		return nil, err
	}
	cur, err := valobj.ParseCurrency(pRec.Currency)
	if err != nil {
		return nil, err
	}
	eteID, err := valobj.ParseEndToEndIdentification(pRec.EndToEndIdentification)
	if err != nil {
		return nil, err
	}

	credPA, err := RecoverPAFromRec(*pRec.CreditorPacc, *pRec.CreditorPacc.Party)
	if err != nil {
		return nil, err
	}
	debPA, err := RecoverPAFromRec(*pRec.DebitorPacc, *pRec.DebitorPacc.Party)
	if err != nil {
		return nil, err
	}

	md, err := valobj.NewMetadata(pRec.Metadata)
	if err != nil {
		return nil, err
	}

	return NewPaymentTX(
		pRec.TXID,
		st,
		am,
		cur,
		eteID,
		pRec.TransactionType,
		debPA,
		credPA,
		md,
	)
}

func (tx *PaymentTX) ToRecord() outport.TXRecord {
	return outport.TXRecord{
		TXID:                   tx.TXID,
		Status:                 string(tx.Status),
		Amount:                 tx.Amount.String(),
		Currency:               tx.Currency.String(),
		EndToEndIdentification: tx.EndToEndIdentification.String(),
		TransactionType:        tx.TransactionType,
		DebitorPacc:            tx.DebitorPacc.ToPaRecord(),
		CreditorPacc:           tx.CreditorPacc.ToPaRecord(),
	}
}

func (tx *PaymentTX) ToTxRequest() outport.TXRequest {
	return outport.TXRequest{
		TXID:                   tx.TXID,
		Status:                 string(tx.Status),
		Amount:                 tx.Amount,
		Currency:               tx.Currency.String(),
		EndToEndIdentification: tx.EndToEndIdentification.String(),
		TransactionType:        tx.TransactionType,
		DebitorIBAN:            tx.DebitorPacc.IBAN,
		CreditorIBAN:           tx.CreditorPacc.IBAN,
	}
}

func (tx *PaymentTX) CredIBAN() string {
	return tx.CreditorPacc.IBAN
}

func (tx *PaymentTX) DebIBAN() string {
	return tx.DebitorPacc.IBAN
}
