package model

import "time"

type Agent struct {
    ID        string    `json:"agent_id"`
    Name      string    `json:"name"`
    Version   string    `json:"version"`
    Owner     string    `json:"owner"`
    PublicKey string    `json:"public_key"`
    CreatedAt time.Time `json:"created_at"`
}

type DaemonStatus struct {
    Running       bool    `json:"running"`
    PID           int     `json:"pid"`
    UptimeSeconds int64   `json:"uptime_seconds"`
    ChainLength   int     `json:"chain_length"`
    AgentCount    int     `json:"agent_count"`
    DBSizeMB      float64 `json:"db_size_mb"`
    ChainVerified bool    `json:"chain_verified"`
    Version       string  `json:"version"`
}

func (a *Agent) ShortID() string {
    if len(a.ID) >= 13 { return a.ID[:13] }
    return a.ID
}

func (a *Agent) ShortPubKey() string {
    if len(a.PublicKey) >= 16 { return a.PublicKey[:16] + "…" }
    return a.PublicKey
}
