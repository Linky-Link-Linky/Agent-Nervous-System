package policy

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRE  = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ssnRE    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	ccRE     = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	phoneRE  = regexp.MustCompile(`\b\+?1?\d{10,15}\b`)
	ipRE     = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	apiKeyRE = regexp.MustCompile(`\b(?:sk-|pk-)[a-zA-Z0-9_-]{8,}\b`)
)

// isValidCreditCard checks the Luhn algorithm on a digit-only string.
func isValidCreditCard(s string) bool {
	var digits []int
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits = append(digits, int(r-'0'))
		}
	}
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// PIIClassification describes what kinds of PII were found.
type PIIClassification struct {
	HasEmail      bool `json:"has_email"`
	HasSSN        bool `json:"has_ssn"`
	HasCreditCard bool `json:"has_credit_card"`
	HasPhone      bool `json:"has_phone"`
	HasIP         bool `json:"has_ip"`
	HasAPIKey     bool `json:"has_api_key"`
	HasPII        bool `json:"has_pii"`
}

// DetectPII scans a string for common PII patterns.
// Uses cheap string pre-checks before running regex to reject clean strings fast.
func DetectPII(s string) PIIClassification {
	var p PIIClassification
	// Cheap pre-filter: check for trigger chars before running regex
	if strings.ContainsAny(s, "@") && emailRE.MatchString(s) {
		p.HasEmail = true
	}
	if strings.ContainsAny(s, "-") && ssnRE.MatchString(s) {
		p.HasSSN = true
	}
	if m := ccRE.FindString(s); m != "" && isValidCreditCard(m) {
		p.HasCreditCard = true
	}
	if containsDigit(s) && phoneRE.MatchString(s) {
		p.HasPhone = true
	}
	if containsDigit(s) && strings.ContainsAny(s, ".") && ipRE.MatchString(s) {
		p.HasIP = true
	}
	if strings.ContainsAny(s, "-_") && apiKeyRE.MatchString(s) {
		p.HasAPIKey = true
	}
	p.HasPII = p.HasEmail || p.HasSSN || p.HasCreditCard || p.HasPhone || p.HasIP || p.HasAPIKey
	return p
}

// containsDigit is a fast digit check without regex.
func containsDigit(s string) bool {
	for i := range s {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}
