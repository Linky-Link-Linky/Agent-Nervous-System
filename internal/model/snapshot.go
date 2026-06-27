package model

import "time"

type Snapshot struct {
	ID         string    `json:"snapshot_id"`
	AgentID    string    `json:"agent_id"`
	ChainIndex int       `json:"chain_index"`
	Slot       int       `json:"slot"`
	Type       string    `json:"snapshot_type"`
	Paths      string    `json:"paths"`
	SizeBytes  int64     `json:"size_bytes"`
	Timestamp  time.Time `json:"timestamp"`
	IsDiff     bool      `json:"is_diff"`
	BaseID     string    `json:"base_snapshot_id"`
}

type SnapshotDiff struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

