// Package corerr defines custom error types for the core domain
package corerr

import "errors"

// VO errors
var (
	ErrMetadataCreatedAtNegative           = errors.New("metadata created_at cannot be negative")
	ErrMetadataUpdatedAtNegative           = errors.New("metadata updated_at cannot be negative")
	ErrMetadataUpdatedBeforeCreated        = errors.New("metadata updated_at cannot be before created_at")
	ErrCurrencyInvalidLength               = errors.New("currency must be 3 characters long")
	ErrCurrencyInvalidCharacters           = errors.New("currency must be uppercase letters")
	ErrContryOfResidenceInvalidLength      = errors.New("country of residence must be 2 characters long")
	ErrEndToEndIdentificationInvalidFormat = errors.New("endToEndIdentification does not match required format")
)

// Entity errors
var (
	ErrIBANInvalidLength = errors.New("iban must be between 15 and 34 characters long")
	ErrBICInvalidLength  = errors.New("bic must be either 8 or 11 characters long")
)

// Service errors
var (
	ErrInfrastructure  = errors.New("infrastructure error occurred")
	ErrPaymentNotFound = errors.New("payment not found")
)
