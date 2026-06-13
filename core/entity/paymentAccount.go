// Package entity defines the core entities used in the payment processing system, such as PaymentAccount and PaymentTransaction. These entities encapsulate the essential data and behaviors related to payments, including account details, transaction information, and associated metadata.
package entity

import (
	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	inport "github.com/Reddetk/CBTraining/ports/inports"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

type PaymentAccount struct {
	IBAN         string
	accCurency   valobj.Currency
	paymentParty *PaymentParty
}

func NewPaymentAccount(IBAN string, currency valobj.Currency) (*PaymentAccount, error) {
	if err := validateIBAN(IBAN); err != nil {
		return nil, err
	}
	return &PaymentAccount{IBAN: IBAN, accCurency: currency}, nil
}

func NewPAFromDTO(paDTO inport.PaymentAccountDTO, ppDTO inport.PaymentPartyDTO) (*PaymentAccount, error) {
	cur, err := valobj.ParseCurrency(paDTO.AccCurency)
	if err != nil {
		return nil, err
	}
	return NewPaymentAccount(paDTO.IBAN, cur)
}

func validateIBAN(IBAN string) error {
	if len(IBAN) < 15 || len(IBAN) > consts.IBANLength {
		return corerr.ErrIBANInvalidLength
	}
	return nil
}

func (pa *PaymentAccount) ToPaRecord() *outport.PARecord {
	return &outport.PARecord{
		IBAN:       pa.IBAN,
		AccCurency: pa.accCurency.String(),
		Party:      pa.paymentParty.ToPartyRecord(),
	}
}
