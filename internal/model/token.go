package model

import (
    "fmt"
    "time"
)

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

func (t *Token) ShortID() string {
    if len(t.ID) >= 12 { return t.ID[:12] }
    return t.ID
}

func (t *Token) TTLSeconds() int {
    r := int(time.Until(t.ExpiresAt).Seconds())
    if r < 0 { return 0 }
    return r
}

func (t *Token) TypeIcon() string {
    switch t.Type {
    case "aws-sts":  return "☁"
    case "vault":    return "⌇"
    case "gcp-iam":  return "◆"
    case "azure-ad": return "▲"
    case "oauth2":   return "◎"
    default:         return "⬡"
    }
}

func (t *Token) TTLColorClass() string {
    s := t.TTLSeconds()
    switch {
    case s > 30: return "ok"
    case s > 10: return "warn"
    default:     return "fail"
    }
}

func (t *Token) ShortResource() string {
    r := []rune(t.Resource)
    if len(r) > 32 { return string(r[:32]) }
    return t.Resource
}

func (t *Token) PermStr() string {
    if len(t.Permissions) == 0 { return "[]" }
    return fmt.Sprint(t.Permissions)
}
