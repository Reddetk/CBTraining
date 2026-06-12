package entity

import (
	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	outport "github.com/Reddetk/CBTraining/ports/outports"
)

type PaymentParty struct {
	BIC               string
	role              valobj.Role
	contryOfResidence valobj.CountryOfResidence
	name              string
}

func NewPaymentParty(
	BIC string, role valobj.Role, countryOfResidence valobj.CountryOfResidence, name string,
) (*PaymentParty, error) {
	if err := validateBIC(BIC); err != nil {
		return nil, err
	}
	return &PaymentParty{
		BIC:               BIC,
		role:              role,
		contryOfResidence: countryOfResidence,
		name:              name,
	}, nil
}

func validateBIC(BIC string) error {
	if len(BIC) != consts.BICLengthA && len(BIC) != consts.BICLengthB {
		return corerr.ErrBICInvalidLength
	}
	return nil
}

func (pp *PaymentParty) ToPartyRecord() *outport.PartyRecord {
	return &outport.PartyRecord{
		BIC:               pp.BIC,
		Role:              string(pp.role),
		ContryOfResidence: pp.contryOfResidence.CountryCode,
		Name:              pp.name,
	}
}
