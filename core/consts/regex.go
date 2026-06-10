// Package consts contains constant values used across the application, such as regex patterns for validation.
package consts

const (
	// Regex for endToEndIdentification: string(len 1-35) {prefix}.{YYYYMMDD}.{seq}  (ISO 20022)
	EndToEndIdentificationRegex = `^[a-zA-Z0-9]{1,35}$`
)
