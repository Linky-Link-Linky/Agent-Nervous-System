package model

import (
    "encoding/json"
    "time"
)

type Receipt struct {
    Index          int       `json:"index"`
    ID             string    `json:"receipt_id"`
    PrevHash       string    `json:"prev_hash"`
    AgentID        string    `json:"agent_id"`
    ActionType     string    `json:"action_type"`
    Phase          string    `json:"phase"`
    Outcome        string    `json:"outcome"`
    DurationMS     int64     `json:"duration_ms"`
    Timestamp      time.Time `json:"timestamp"`
    PolicyDecision string    `json:"policy_decision"`
    PayloadSummary string    `json:"payload_summary"`
    SnapshotID     string    `json:"snapshot_id"`
    Signature      string    `json:"signature"`
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
    type alias Receipt
    aux := &struct {
        Timestamp json.RawMessage `json:"timestamp"`
        *alias
    }{alias: (*alias)(r)}
    if err := json.Unmarshal(data, aux); err != nil {
        return err
    }
    if len(aux.Timestamp) > 0 {
        var s string
        if err := json.Unmarshal(aux.Timestamp, &s); err == nil {
            t, err2 := time.Parse(time.RFC3339Nano, s)
            if err2 == nil { r.Timestamp = t; return nil }
        }
        var ns int64
        if err := json.Unmarshal(aux.Timestamp, &ns); err == nil {
            r.Timestamp = time.Unix(0, ns).UTC()
        }
    }
    return nil
}

func (r *Receipt) ShortID() string {
    if len(r.ID) >= 12 { return r.ID[:12] }
    return r.ID
}

func (r *Receipt) ShortPrevHash() string {
    if len(r.PrevHash) >= 12 { return r.PrevHash[:12] }
    return r.PrevHash
}

func (r *Receipt) ShortSignature() string {
    if len(r.Signature) >= 16 { return r.Signature[:16] + "…" }
    return r.Signature
}

func (r *Receipt) ShortAgent() string {
    if len(r.AgentID) >= 13 { return r.AgentID[:13] }
    return r.AgentID
}
