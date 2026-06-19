package mcp

import "regexp"

var (
	emailRE  = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ssnRE    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	ccRE     = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	phoneRE  = regexp.MustCompile(`\b\+?1?\d{10,15}\b`)
	ipRE     = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	apiKeyRE = regexp.MustCompile(`\b(?:sk-|pk-)[a-zA-Z0-9_-]{8,}\b`)
)

func RedactPII(s string) string {
	s = emailRE.ReplaceAllString(s, "[REDACTED_EMAIL]")
	s = ssnRE.ReplaceAllString(s, "[REDACTED_SSN]")
	s = ccRE.ReplaceAllString(s, "[REDACTED_CC]")
	s = phoneRE.ReplaceAllString(s, "[REDACTED_PHONE]")
	s = ipRE.ReplaceAllString(s, "[REDACTED_IP]")
	s = apiKeyRE.ReplaceAllString(s, "[REDACTED_API_KEY]")
	return s
}
