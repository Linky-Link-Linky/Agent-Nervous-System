package model

import "time"

type Agent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Owner        string    `json:"owner"`
	PublicKey    string    `json:"public_key"`
	RegisteredAt time.Time `json:"registered_at"`
}
