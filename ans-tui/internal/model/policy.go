package model

import "encoding/json"

type Policy struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Enabled    bool            `json:"enabled"`
	Priority   int             `json:"priority"`
	Severity   string          `json:"severity"`
	Effect     string          `json:"effect"`
	Conditions json.RawMessage `json:"conditions"`
}
