// Package valobj implments the value object pattern.
package valobj

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/Reddetk/CBTraining/core/consts"
	corerr "github.com/Reddetk/CBTraining/core/coreErrors"
)

type Role string

const (
	Debtor   Role = "debtor"
	Creditor Role = "creditor"
)

func ValRole(s string) (Role, error) {
	if s == "debtor" {
		return Debtor, nil
	}
	if s == "creditor" {
		return Creditor, nil
	}
	return "", corerr.ErrNotValidRole
}

// Metadata represents creation/update timestamps -------------
type Metadata struct {
	CreatedAt time.Time `json:"created_at"` // Unix RFC3339 timestamp in milliseconds
	UpdatedAt time.Time `json:"updated_at"` // Unix RFC3339 timestamp in milliseconds
}

// NewMetadata creates Metadata with explicit timestamps
func NewMetadata(input string) (Metadata, error) {
	var mt Metadata
	err := json.Unmarshal([]byte(input), &mt)
	if err != nil {
		return Metadata{}, fmt.Errorf("%w - '%s'", corerr.ErrMetadataNotValid, input)
	}
	if mt.UpdatedAt.Before(mt.CreatedAt) {
		return Metadata{}, fmt.Errorf("%w - '%s'", corerr.ErrMetadataUpdatedBeforeCreated, input)
	}

	return mt, nil
}

// NewMetadataNow creates Metadata with current time for both timestamps
func NewMetadataNow() Metadata {
	return Metadata{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now()}
}

// Touch returns new Metadata with updated updatedAt -- immutable update
func (m Metadata) Touch() Metadata {
	return Metadata{
		CreatedAt: m.CreatedAt,
		UpdatedAt: time.Now(),
	}
}

func (m Metadata) String() string {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// Status represents the status of a transaction -------------
type Status string

const (
	Pending    Status = "pending"
	Processing Status = "processing"
	Completed  Status = "completed"
	Failed     Status = "failed"
)

func ValStatus(st string) (Status, error) {
	switch {
	case st == string(Pending):
		return Pending, nil
	case st == string(Processing):
		return Processing, nil
	case st == string(Completed):
		return Processing, nil
	case st == string(Failed):
		return Processing, nil
	default:
		return "", corerr.ErrNotValidStatus
	}
}

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
