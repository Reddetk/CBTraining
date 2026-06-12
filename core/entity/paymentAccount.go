// Package entity defines the core entities used in the payment processing system, such as PaymentAccount and PaymentTransaction. These entities encapsulate the essential data and behaviors related to payments, including account details, transaction information, and associated metadata.
package entity

import (
	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

type PaymentAccount struct {
	IBAN       string
	AccCurency valobj.Currency
}

func NewPaymentAccount(IBAN string, currency valobj.Currency) (*PaymentAccount, error) {
	if err := validateIBAN(IBAN); err != nil {
		return nil, err
	}
	return &PaymentAccount{IBAN: IBAN, AccCurency: currency}, nil
}

func validateIBAN(IBAN string) error {
	if len(IBAN) < 15 || len(IBAN) > consts.IBANLength {
		return corerr.ErrIBANInvalidLength
	}
	return nil
}

func (pa *PaymentAccount) ToPaRecord(BIC string, role valobj.Role, countryOfResidence valobj.CountryOfResidence, name string) (*outport.PARecord, error) {
	party, err := NewPaymentParty(BIC, role, countryOfResidence, name)
	if err != nil {
		return &outport.PARecord{}, err
	}
	return &outport.PARecord{
		IBAN:       pa.IBAN,
		AccCurency: pa.AccCurency.String(),
		Party:      party.ToPartyRecord(),
	}, nil
}
