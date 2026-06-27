package model

import (
	"encoding/json"
	"time"
)

type Receipt struct {
	Index        int       `json:"index"`
	ID           string    `json:"receipt_id"`
	PrevHash     string    `json:"prev_hash"`
	AgentID      string    `json:"agent_id"`
	ActionType   string    `json:"action_type"`
	Phase        string    `json:"phase"`
	Outcome      string    `json:"outcome"`
	DurationMS   int64     `json:"duration_ms"`
	Timestamp    time.Time `json:"-"`
	PolicyDecision string  `json:"policy_decision"`
	PayloadSummary string  `json:"payload_summary"`
	SnapshotID   string    `json:"snapshot_id"`
	Signature    string    `json:"signature"`
}

func (r *Receipt) UnmarshalJSON(b []byte) error {
	type raw Receipt
	var r2 raw
	if err := json.Unmarshal(b, &r2); err != nil {
		return err
	}
	*r = Receipt(r2)

	// Try nanosecond int64 or RFC3339
	var rawMap map[string]any
	if err := json.Unmarshal(b, &rawMap); err != nil {
		return err
	}
	if ts, ok := rawMap["timestamp_ns"]; ok {
		switch v := ts.(type) {
		case float64:
			r.Timestamp = time.Unix(0, int64(v))
		case int64:
			r.Timestamp = time.Unix(0, v)
		case json.Number:
			n, _ := v.Int64()
			r.Timestamp = time.Unix(0, n)
		}
	} else if ts, ok := rawMap["timestamp"]; ok {
		if s, ok := ts.(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				r.Timestamp = t
			}
		}
	}
	return nil
}

type DaemonStatus struct {
	Running       bool    `json:"running"`
	Uptime        string  `json:"uptime"`
	ChainLength   int     `json:"chain_length"`
	AgentCount    int     `json:"agent_count"`
	DBSizeMB      float64 `json:"db_size_mb"`
	ChainVerified bool    `json:"chain_verified"`
	Version       string  `json:"version"`
}
