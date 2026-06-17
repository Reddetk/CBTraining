// Package corerr defines custom error types for the core domain
package corerr

import "errors"

// VO errors
var (
	ErrMetadataNotValid                    = errors.New("metadata cnot valid format")
	ErrMetadataUpdatedBeforeCreated        = errors.New("metadata updated_at cannot be before created_at")
	ErrCurrencyInvalidLength               = errors.New("currency must be 3 characters long")
	ErrCurrencyInvalidCharacters           = errors.New("currency must be uppercase letters")
	ErrContryOfResidenceInvalidLength      = errors.New("country of residence must be 2 characters long")
	ErrEndToEndIdentificationInvalidFormat = errors.New("endToEndIdentification does not match required format")
	ErrNotValidRole                        = errors.New("not valid role")
	ErrNotValidStatus                      = errors.New("not valid status")
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

	ErrDispatcherShuting      = errors.New("dispatcher is shutting down")
	ErrFailedTorecoverPanding = errors.New("failed to recover panding tx-s")
)
