// Package identity manages cryptographic agent identities for ANS.
// Each agent gets a unique Ed25519 keypair. The public key hash is the agent ID.
// Base58 encoding is implemented inline — no external dependency.
// SPDX-License-Identifier: Apache-2.0
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

const AgentIDPrefix = "ans_"

// base58Encode encodes bytes using the Bitcoin base58 alphabet.
// Implemented inline to eliminate external dependencies.
func base58Encode(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	leadingZeros := 0
	for _, b := range input {
		if b != 0 {
			break
		}
		leadingZeros++
	}
	// Work on a copy as big-endian integer
	num := make([]int, len(input))
	for i, b := range input {
		num[i] = int(b)
	}
	var digits []byte
	for len(num) > 0 {
		var remainder int
		var next []int
		for _, v := range num {
			cur := remainder*256 + v
			q := cur / 58
			remainder = cur % 58
			if len(next) > 0 || q > 0 {
				next = append(next, q)
			}
		}
		digits = append(digits, alphabet[remainder])
		num = next
	}
	for i := 0; i < leadingZeros; i++ {
		digits = append(digits, alphabet[0])
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// Agent holds all identity information for a single agent.
type Agent struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Owner        string             `json:"owner"`
	PublicKey    ed25519.PublicKey  `json:"public_key"`
	PrivateKey   ed25519.PrivateKey `json:"-"`
	RegisteredAt time.Time          `json:"registered_at"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
}

// AgentExport is the JSON-serializable form of an Agent (no private key).
type AgentExport struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Owner        string            `json:"owner"`
	PublicKeyHex string            `json:"public_key_hex"`
	RegisteredAt time.Time         `json:"registered_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// New generates a new Agent with a fresh Ed25519 keypair.
func New(name, version, owner string) (*Agent, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}
	return &Agent{
		ID:           deriveID(pub),
		Name:         name,
		Version:      version,
		Owner:        owner,
		PublicKey:    pub,
		PrivateKey:   priv,
		RegisteredAt: time.Now().UTC(),
		Metadata:     make(map[string]string),
	}, nil
}

// NewFromKeys reconstructs an Agent from an existing keypair.
func NewFromKeys(name, version, owner string, pub ed25519.PublicKey, priv ed25519.PrivateKey) *Agent {
	return &Agent{
		ID:           deriveID(pub),
		Name:         name,
		Version:      version,
		Owner:        owner,
		PublicKey:    pub,
		PrivateKey:   priv,
		RegisteredAt: time.Now().UTC(),
		Metadata:     make(map[string]string),
	}
}

// deriveID computes "ans_" + base58(sha256(pubkey)[:10]).
func deriveID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return AgentIDPrefix + base58Encode(h[:10])
}

// Sign signs a message with the agent's private key.
func (a *Agent) Sign(message []byte) []byte {
	return ed25519.Sign(a.PrivateKey, message)
}

// Verify checks a signature against the agent's public key.
func (a *Agent) Verify(message, sig []byte) bool {
	return ed25519.Verify(a.PublicKey, message, sig)
}

// Export returns a JSON-safe representation (no private key).
func (a *Agent) Export() AgentExport {
	return AgentExport{
		ID:           a.ID,
		Name:         a.Name,
		Version:      a.Version,
		Owner:        a.Owner,
		PublicKeyHex: fmt.Sprintf("%x", a.PublicKey),
		RegisteredAt: a.RegisteredAt,
		Metadata:     a.Metadata,
	}
}

// MarshalJSON never serializes the private key.
func (a *Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Export())
}
