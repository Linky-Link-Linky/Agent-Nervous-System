// Package receipt — signer handles Ed25519 signing and verification.
// SPDX-License-Identifier: MIT
package receipt

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Signer wraps an Ed25519 private key.
type Signer struct {
	priv ed25519.PrivateKey
}

// NewSigner creates a Signer from an Ed25519 private key.
func NewSigner(priv ed25519.PrivateKey) *Signer {
	return &Signer{priv: priv}
}

// Sign calls SetReceiptID (if not already set), then signs SignableBytes
// and stores the hex signature in r.Signature.
// Computes SignableBytes once and reuses it for both the hash and signature.
func (s *Signer) Sign(r *Receipt) error {
	r.cachedHash = ""
	r.cachedRaw = nil
	msg, err := r.SignableBytes()
	if err != nil {
		return fmt.Errorf("computing signable bytes: %w", err)
	}
	if r.ReceiptID == "" {
		h := sha256.Sum256(msg)
		r.ReceiptID = fmt.Sprintf("%x", h[:16])
	}
	r.Signature = hex.EncodeToString(ed25519.Sign(s.priv, msg))
	return nil
}

// Verify checks r.Signature against pub over r.SignableBytes().
func Verify(r *Receipt, pub ed25519.PublicKey) error {
	if r.Signature == "" {
		return fmt.Errorf("receipt %s has no signature", r.ReceiptID)
	}
	sig, err := hex.DecodeString(r.Signature)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}
	msg, err := r.SignableBytes()
	if err != nil {
		return fmt.Errorf("computing signable bytes: %w", err)
	}
	if !ed25519.Verify(pub, msg, sig) {
		return fmt.Errorf("signature invalid for receipt %s", r.ReceiptID)
	}
	return nil
}
