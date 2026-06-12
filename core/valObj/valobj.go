// Package valobj implments the value object pattern.
package valobj

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
)

type Role string

const (
	Debitor  Role = "debitor"
	Creditor Role = "creditor"
)

// Metadata represents creation/update timestamps -------------
type Metadata struct {
	createdAt int64 // Unix timestamp in milliseconds
	updatedAt int64 // Unix timestamp in milliseconds
}

func validateMetadata(createdAt, updatedAt int64) error {
	if createdAt < 0 {
		return corerr.ErrMetadataCreatedAtNegative
	}
	if updatedAt < 0 {
		return corerr.ErrMetadataUpdatedAtNegative
	}
	if updatedAt < createdAt {
		return corerr.ErrMetadataUpdatedBeforeCreated
	}
	return nil
}

// NewMetadata creates Metadata with explicit timestamps
func NewMetadata(createdAt, updatedAt int64) (Metadata, error) {
	if err := validateMetadata(createdAt, updatedAt); err != nil {
		return Metadata{}, err
	}
	return Metadata{createdAt: createdAt, updatedAt: updatedAt}, nil
}

// NewMetadataNow creates Metadata with current time for both timestamps
func NewMetadataNow() Metadata {
	now := time.Now().UnixMilli()
	return Metadata{createdAt: now, updatedAt: now}
}

// Touch returns new Metadata with updated updatedAt -- immutable update
func (m Metadata) Touch() Metadata {
	return Metadata{
		createdAt: m.createdAt,
		updatedAt: time.Now().UnixMilli(),
	}
}

func (m Metadata) Equals(other Metadata) bool {
	return m.createdAt == other.createdAt && m.updatedAt == other.updatedAt
}

// String returns a created_at=%s updated_at=%s representation of Metadata
func (m Metadata) String() string {
	cA := strconv.FormatInt(m.createdAt, 10)
	uA := strconv.FormatInt(m.updatedAt, 10)
	return fmt.Sprintf("created_at=%s updated_at=%s", cA, uA)
}

// Status represents the status of a transaction -------------
type Status string

const (
	Pending    Status = "pending"
	Processing Status = "processing"
	Completed  Status = "completed"
	Failed     Status = "failed"
)

// Currency represents a 3-letter ISO currency code ------------
type Currency struct {
	Currency string `validate:"required,len=3"`
}

func ParseCurrency(s string) (Currency, error) {
	if len(s) != 3 {
		return Currency{}, corerr.ErrCurrencyInvalidLength
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return Currency{}, corerr.ErrCurrencyInvalidCharacters
		}
	}
	return Currency{Currency: s}, nil
}

func (c *Currency) String() string {
	return c.Currency
}

// EndToEndIdentification represents a string that must match the regex pattern defined in consts -------
type EndToEndIdentification struct {
	Identification string `validate:"required,regex=^[a-zA-Z0-9]{1,35}$"`
}

func ParseEndToEndIdentification(s string) (EndToEndIdentification, error) {
	if regex := consts.EndToEndIdentificationRegex; !regexp.MustCompile(regex).MatchString(s) {
		return EndToEndIdentification{}, corerr.ErrEndToEndIdentificationInvalidFormat
	}
	return EndToEndIdentification{Identification: s}, nil
}

func (etei *EndToEndIdentification) String() string {
	return etei.Identification
}

// CountryOfResidence char(len 2)(ContryCode  ISO 3166-1 alpha-2.)
type CountryOfResidence struct {
	CountryCode string `validate:"required,len=2"`
}

func ParseCountryOfResidence(s string) (CountryOfResidence, error) {
	if len(s) != 2 {
		return CountryOfResidence{}, corerr.ErrContryOfResidenceInvalidLength
	}
	return CountryOfResidence{CountryCode: s}, nil
}
