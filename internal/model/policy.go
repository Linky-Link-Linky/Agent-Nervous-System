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

type PolicyEvalResult struct {
    Allowed       bool   `json:"allowed"`
    DenyingPolicy string `json:"denying_policy_id,omitempty"`
    DenyReason    string `json:"deny_reason,omitempty"`
    ErrorType     string `json:"error_type,omitempty"`
    ErrorCode     string `json:"error_code,omitempty"`
    Evaluated     int    `json:"policies_evaluated"`
}

func (p *Policy) ShortID() string {
    if len(p.ID) >= 20 { return p.ID[:20] }
    return p.ID
}

func (p *Policy) ShortName() string {
    r := []rune(p.Name)
    if len(r) > 26 { return string(r[:26]) }
    return p.Name
}
