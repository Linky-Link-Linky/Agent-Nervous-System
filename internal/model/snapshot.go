package model

import (
    "fmt"
    "time"
)

type Snapshot struct {
    ID         string    `json:"snapshot_id"`
    AgentID    string    `json:"agent_id"`
    ChainIndex int       `json:"chain_index"`
    Type       string    `json:"snapshot_type"`
    SizeBytes  int64     `json:"size_bytes"`
    Timestamp  time.Time `json:"timestamp"`
    IsDiff     bool      `json:"is_diff"`
    BaseID     string    `json:"base_snapshot_id"`
}

func (s *Snapshot) ShortID() string {
    if len(s.ID) >= 12 { return s.ID[:12] }
    return s.ID
}

func (s *Snapshot) SizeStr() string {
    b := s.SizeBytes
    switch {
    case b < 1024:            return fmt.Sprintf("%dB", b)
    case b < 1024*1024:       return fmt.Sprintf("%.1fKB", float64(b)/1024)
    default:                  return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
    }
}
