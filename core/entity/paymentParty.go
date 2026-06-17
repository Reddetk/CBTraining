package entity

import (
	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
	valobj "github.com/Reddetk/CBTraining/core/valObj"
	inport "github.com/Reddetk/CBTraining/ports/inports"
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

func NewPaymentPartyFromDTO(ppDTO inport.PaymentPartyDTO) (*PaymentParty, error) {
	cor, err := valobj.ParseCountryOfResidence(ppDTO.ContryOfResidence)
	if err != nil {
		return nil, err
	}
	role, err := valobj.ValRole(ppDTO.Role)
	if err != nil {
		return nil, err
	}
	return NewPaymentParty(ppDTO.BIC, role, cor, ppDTO.Name)
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

func  RecoverPPFromRec(ppr *outport.PartyRecord) (*PaymentParty, error) {
	cor, err := valobj.ParseCountryOfResidence(ppr.ContryOfResidence)
	if err != nil {
		return nil, err
	}
	role, err := valobj.ValRole(ppr.Role)
	if err != nil {
		return nil, err
	}
	return NewPaymentParty(ppr.BIC, role, cor, ppr.Name)
}
