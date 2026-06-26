package model

import "time"

type Token struct {
	ID          string    `json:"credential_id"`
	Type        string    `json:"type"`
	AgentID     string    `json:"agent_id"`
	Resource    string    `json:"resource"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expires_at"`
	IssuedAt    time.Time `json:"issued_at"`
	Revoked     bool      `json:"revoked"`
}

func (t *Token) TTLSeconds() int {
	remaining := time.Until(t.ExpiresAt).Seconds()
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}
