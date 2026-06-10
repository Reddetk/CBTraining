// Package entity defines the core entities used in the payment processing system, such as PaymentAccount and PaymentTransaction. These entities encapsulate the essential data and behaviors related to payments, including account details, transaction information, and associated metadata.
package entity

import valobj "github.com/Reddetk/CBTraining/core/valObj"

type PaymentAccount struct {
	IBAN       string
	AccCurency valobj.Currency
}
