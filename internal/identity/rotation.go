// Package identity — rotation.go implements key rotation with a signed rotation receipt.
// Rotating a key generates a new keypair, signs a rotation record with BOTH the old
// and new private keys, and saves the new keypair. The rotation record is returned so
// the caller can append it to the chain as a CustomAction receipt.
// SPDX-License-Identifier: MIT
package identity

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// RotationRecord is the payload appended to the chain when a key is rotated.
// It is signed by both the old and new private keys, proving control of both.
type RotationRecord struct {
	AgentID      string `json:"agent_id"`
	OldPublicKey string `json:"old_public_key_hex"`
	NewPublicKey string `json:"new_public_key_hex"`
	RotatedAt    string `json:"rotated_at"`
	OldSignature string `json:"old_signature_hex"` // sign(canonical_json_without_sigs, oldPriv)
	NewSignature string `json:"new_signature_hex"` // sign(canonical_json_without_sigs, newPriv)
}

// Rotate generates a new keypair for the agent, saves it to the keystore, and
// returns a RotationRecord suitable for embedding in a chain receipt payload.
// The old key is NOT deleted — historical receipts remain verifiable with it.
func (ks *Keystore) Rotate(agentID string) (*Agent, *RotationRecord, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	old, err := ks.loadLocked(agentID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading existing agent: %w", err)
	}

	// Generate new keypair
	newAgent, err := New(old.Name, old.Version, old.Owner)
	if err != nil {
		return nil, nil, fmt.Errorf("generating new keypair: %w", err)
	}

	// Build the signable body (without signatures)
	type rotationBody struct {
		AgentID      string `json:"agent_id"`
		OldPublicKey string `json:"old_public_key_hex"`
		NewPublicKey string `json:"new_public_key_hex"`
		RotatedAt    string `json:"rotated_at"`
	}
	body := rotationBody{
		AgentID:      agentID,
		OldPublicKey: hex.EncodeToString(old.PublicKey),
		NewPublicKey: hex.EncodeToString(newAgent.PublicKey),
		RotatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling rotation body: %w", err)
	}

	oldSig := ed25519.Sign(old.PrivateKey, bodyBytes)
	newSig := ed25519.Sign(newAgent.PrivateKey, bodyBytes)

	rec := &RotationRecord{
		AgentID:      body.AgentID,
		OldPublicKey: body.OldPublicKey,
		NewPublicKey: body.NewPublicKey,
		RotatedAt:    body.RotatedAt,
		OldSignature: hex.EncodeToString(oldSig),
		NewSignature: hex.EncodeToString(newSig),
	}

	// Save new agent under its new ID
	if err := ks.saveLocked(newAgent); err != nil {
		return nil, nil, fmt.Errorf("saving new keypair: %w", err)
	}

	return newAgent, rec, nil
}

// VerifyRotation checks that a RotationRecord was signed by both claimed keys.
func VerifyRotation(rec *RotationRecord) error {
	if rec == nil {
		return fmt.Errorf("rotation record is nil")
	}
	type rotationBody struct {
		AgentID      string `json:"agent_id"`
		OldPublicKey string `json:"old_public_key_hex"`
		NewPublicKey string `json:"new_public_key_hex"`
		RotatedAt    string `json:"rotated_at"`
	}
	body := rotationBody{
		AgentID: rec.AgentID, OldPublicKey: rec.OldPublicKey,
		NewPublicKey: rec.NewPublicKey, RotatedAt: rec.RotatedAt,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	oldPub, err := hex.DecodeString(rec.OldPublicKey)
	if err != nil || len(oldPub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid old public key hex")
	}
	newPub, err := hex.DecodeString(rec.NewPublicKey)
	if err != nil || len(newPub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid new public key hex")
	}
	oldSig, err := hex.DecodeString(rec.OldSignature)
	if err != nil {
		return fmt.Errorf("invalid old signature hex")
	}
	newSig, err := hex.DecodeString(rec.NewSignature)
	if err != nil {
		return fmt.Errorf("invalid new signature hex")
	}
	if !ed25519.Verify(ed25519.PublicKey(oldPub), bodyBytes, oldSig) {
		return fmt.Errorf("old key signature invalid")
	}
	if !ed25519.Verify(ed25519.PublicKey(newPub), bodyBytes, newSig) {
		return fmt.Errorf("new key signature invalid")
	}
	return nil
}
