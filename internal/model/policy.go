package model

import "encoding/json"

type Policy struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Enabled        bool            `json:"enabled"`
	Priority       int             `json:"priority"`
	Severity       string          `json:"severity"`
	Effect         string          `json:"effect"`
	ActionType     string          `json:"action_type"`
	PayloadSummary string          `json:"payload_summary"`
	Conditions     json.RawMessage `json:"conditions"`
}

type PolicyEvalResult struct {
	Allowed       bool   `json:"allowed"`
	MatchedPolicy string `json:"matched_policy"`
}

