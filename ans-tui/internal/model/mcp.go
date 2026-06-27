package model

import "time"

type MCPStatus struct {
	Running        bool    `json:"running"`
	UptimeSeconds  int64   `json:"uptime_seconds"`
	ListenAddr     string  `json:"listen_addr"`
	TargetURL      string  `json:"target_url"`
	TotalMessages  int64   `json:"total_messages"`
	TotalTokens    int64   `json:"total_tokens"`
	BurnRate       float64 `json:"burn_rate_per_sec"`
	Injections     int     `json:"injection_count"`
	Pruned         int     `json:"pruned_count"`
	RateLimited    int     `json:"rate_limited_count"`
	BudgetExceeded int     `json:"budget_exceeded_count"`
	PolicyDenied   int     `json:"policy_denied_count"`
	ToolsDenied    int     `json:"tools_denied_count"`
	RecentLog      []*MCPLogEntry `json:"recent_log"`
	ReqHistory     []float64      `json:"req_history"`
	TokHistory     []float64      `json:"tok_history"`
}

type MCPLogEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	Direction     string    `json:"direction"`
	Method        string    `json:"method"`
	TokenEstimate int       `json:"token_estimate"`
	InjDetected   bool      `json:"injection_detected"`
	PIIFound      bool      `json:"pii_found"`
	PolicyResult  string    `json:"policy_result"`
	ContentPreview string   `json:"content_preview"`
}
