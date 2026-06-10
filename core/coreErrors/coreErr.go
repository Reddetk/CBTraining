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
