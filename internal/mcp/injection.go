package mcp

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	systemOverrideRE  = regexp.MustCompile(`(?i)(?:\b(?:ignore|override|disregard|forget|discard)\s+(?:all\s+)?(?:previous|above|prior|system|instructions))`)
	delimiterBreakRE  = regexp.MustCompile(`(?i)(?:"""|---|<\|im_end\|>|<\|im_start\|>|<\|sys\|>|<\|assistant\|>|<\|user\|>|\[INST\]|\[\/INST\])`)
	roleInjectionRE   = regexp.MustCompile(`(?i)\b(?:from now on|you are now|act as|pretend to be|you will now|your new role|your new system)\b`)
	dataExfilRE       = regexp.MustCompile(`(?i)(?:send\s+(?:this|the|all)\s+(?:data|info|file|document)(?:\s+to|to\s+https?)|exfiltrate|leak\s+this)`)
	contextOverflowRE = regexp.MustCompile(`(?i)(?:repeat\s+(?:after|this|the|above)|say\s+the\s+word|print\s+the\s+word)`)
	payloadObfuscationRE = regexp.MustCompile(`(?i)(?:base64|hex[\s_]*encode|rot13|cipher|encoded[\s_]*string|obfuscated)`)
)

// InjectionType describes the kind of injection found.
type InjectionType string

const (
	InjSystemOverride  InjectionType = "system_override"
	InjDelimiterBreak  InjectionType = "delimiter_break"
	InjRoleInjection   InjectionType = "role_injection"
	InjDataExfil       InjectionType = "data_exfiltration"
	InjContextOverflow InjectionType = "context_overflow"
	InjObfuscation     InjectionType = "payload_obfuscation"
)

// CheckInjection scans content for prompt injection patterns.
func CheckInjection(content string) (InjectionType, bool) {
	if systemOverrideRE.MatchString(content) {
		return InjSystemOverride, true
	}
	if delimiterBreakRE.MatchString(content) {
		return InjDelimiterBreak, true
	}
	if roleInjectionRE.MatchString(content) {
		return InjRoleInjection, true
	}
	if dataExfilRE.MatchString(content) {
		return InjDataExfil, true
	}
	if contextOverflowRE.MatchString(content) {
		return InjContextOverflow, true
	}
	if payloadObfuscationRE.MatchString(content) {
		return InjObfuscation, true
	}
	return "", false
}

// ScanParams extracts readable text from JSON-RPC params for injection scanning.
func ScanParams(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		return string(raw)
	}
	var parts []string
	for _, key := range []string{"text", "content", "prompt", "query", "input", "arguments", "uri"} {
		if v, ok := params[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				parts = append(parts, s)
			} else {
				var nested map[string]json.RawMessage
				if json.Unmarshal(v, &nested) == nil {
					for _, nk := range []string{"text", "content", "data"} {
						if nv, ok := nested[nk]; ok {
							var ns string
							if json.Unmarshal(nv, &ns) == nil {
								parts = append(parts, ns)
							}
						}
					}
				}
			}
		}
	}
	return strings.Join(parts, " ")
}
