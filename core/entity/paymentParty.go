package entity

import valobj "github.com/Reddetk/CBTraining/core/valObj"

type PaymentParty struct {
	BIC               string
	role              valobj.Role
	contryOfResidence valobj.CountryOfResidence
	name              string
}
