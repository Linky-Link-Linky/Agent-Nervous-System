// Package identity — keystore persists agent keypairs encrypted with AES-256-GCM.
// Encryption key is derived from a machine-local secret via HKDF-SHA256.
// Security note: protecting ~/.ans/machine.secret protects all stored keys.
// SPDX-License-Identifier: MIT
package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

// agentIDSafe matches only alphanumeric + underscore + hypen (base58 chars in agent IDs).
var agentIDSafe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func sanitizeAgentID(id string) error {
	if len(id) == 0 || len(id) > 128 {
		return fmt.Errorf("invalid agent ID length")
	}
	if !agentIDSafe.MatchString(id) {
		return fmt.Errorf("agent ID contains invalid characters")
	}
	return nil
}

// KeystoreEntry is the plaintext structure serialized before encryption.
type KeystoreEntry struct {
	AgentID    string `json:"agent_id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Owner      string `json:"owner"`
	PublicKey  []byte `json:"public_key"`
	PrivateKey []byte `json:"private_key"`
}

// cacheTTL is how long a decrypted agent stays in memory before re-read from disk.
const cacheTTL = 5 * 60_000_000_000 // 5 minutes in nanoseconds

type cachedAgent struct {
	agent     *Agent
	expiresAt int64 // unix nanos
}

// Keystore manages encrypted agent keys at a local directory.
type Keystore struct {
	mu     sync.Mutex
	dir    string
	encKey []byte // 32-byte AES-256-GCM key
	cache  map[string]*cachedAgent
}

// NewKeystore opens (or creates) a keystore at dir.
// If dir is empty, defaults to ~/.ans/keys.
func NewKeystore(dir string) (*Keystore, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home dir: %w", err)
		}
		dir = filepath.Join(home, ".ans", "keys")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating keystore dir: %w", err)
	}
	encKey, err := deriveMachineKey()
	if err != nil {
		return nil, fmt.Errorf("deriving machine key: %w", err)
	}
	return &Keystore{dir: dir, encKey: encKey, cache: make(map[string]*cachedAgent)}, nil
}

// Save encrypts and writes an agent keypair to disk.
func (ks *Keystore) Save(a *Agent) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return ks.saveLocked(a)
}

// saveLocked is Save with ks.mu already held by caller.
func (ks *Keystore) saveLocked(a *Agent) error {
	if err := sanitizeAgentID(a.ID); err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}
	delete(ks.cache, a.ID) // invalidate cache
	entry := KeystoreEntry{
		AgentID:    a.ID,
		Name:       a.Name,
		Version:    a.Version,
		Owner:      a.Owner,
		PublicKey:  []byte(a.PublicKey),
		PrivateKey: []byte(a.PrivateKey),
	}
	plaintext, err := json.Marshal(entry) // #nosec G117
	if err != nil {
		return fmt.Errorf("marshalling entry: %w", err)
	}
	ciphertext, err := ksEncrypt(ks.encKey, plaintext)
	if err != nil {
		return fmt.Errorf("encrypting entry: %w", err)
	}
	return os.WriteFile(filepath.Join(ks.dir, a.ID+".key"), ciphertext, 0600)
}

// Load decrypts and returns the agent for the given ID.
// Results are cached in memory for cacheTTL to avoid repeated disk I/O + AES decrypt.
func (ks *Keystore) Load(agentID string) (*Agent, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return ks.loadLocked(agentID)
}

// loadLocked is Load with ks.mu already held by caller.
func (ks *Keystore) loadLocked(agentID string) (*Agent, error) {
	if err := sanitizeAgentID(agentID); err != nil {
		return nil, fmt.Errorf("invalid agent ID: %w", err)
	}
	// Check in-memory cache first
	if c, ok := ks.cache[agentID]; ok && time.Now().UnixNano() < c.expiresAt {
		return c.agent, nil
	}
	ciphertext, err := os.ReadFile(filepath.Join(ks.dir, agentID+".key")) // #nosec G304
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("agent %s not found in keystore", agentID)
		}
		return nil, fmt.Errorf("reading keyfile: %w", err)
	}
	plaintext, err := ksDecrypt(ks.encKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypting entry: %w", err)
	}
	var entry KeystoreEntry
	if err := json.Unmarshal(plaintext, &entry); err != nil {
		return nil, fmt.Errorf("unmarshalling entry: %w", err)
	}
	agent := NewFromKeys(
		entry.Name, entry.Version, entry.Owner,
		ed25519.PublicKey(entry.PublicKey),
		ed25519.PrivateKey(entry.PrivateKey),
	)
	// Cache for subsequent lookups
	ks.cache[agentID] = &cachedAgent{agent: agent, expiresAt: time.Now().UnixNano() + cacheTTL}
	return agent, nil
}

// List returns all agent IDs in the keystore directory.
func (ks *Keystore) List() ([]string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	entries, err := os.ReadDir(ks.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading keystore dir: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".key" {
			ids = append(ids, e.Name()[:len(e.Name())-4])
		}
	}
	return ids, nil
}

// Delete removes a keypair from the keystore.
func (ks *Keystore) Delete(agentID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if err := sanitizeAgentID(agentID); err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}
	delete(ks.cache, agentID) // invalidate cache
	return os.Remove(filepath.Join(ks.dir, agentID+".key"))
}

func deriveMachineKey() ([]byte, error) {
	secret, err := machineSecret()
	if err != nil {
		return nil, err
	}
	r := hkdf.New(sha256.New, secret, []byte("ans-keystore-v1"), []byte("keystore-enc-key"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func machineSecret() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home dir: %w", err)
	}
	secretPath := filepath.Join(home, ".ans", "machine.secret")
	if data, readErr := os.ReadFile(secretPath); readErr == nil { // #nosec G304
		if len(data) == 32 {
			return data, nil
		}
		return nil, fmt.Errorf("machine.secret exists but has wrong length (%d bytes, expected 32); rename or delete it manually", len(data))
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
		return nil, err
	}
	// Use O_EXCL to atomically create the file, preventing TOCTOU race with
	// concurrent NewKeystore calls (in-process or cross-process).
	f, err := os.OpenFile(secretPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600) // #nosec G304
	if err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating machine.secret: %w", err)
		}
		// Another process created it first; read their version.
		data, err := os.ReadFile(secretPath) // #nosec G304
		if err != nil {
			return nil, fmt.Errorf("reading machine.secret after race: %w", err)
		}
		if len(data) != 32 {
			return nil, fmt.Errorf("machine.secret has wrong length (%d bytes, expected 32)", len(data))
		}
		return data, nil
	}
	if _, err := f.Write(secret); err != nil {
		_ = f.Close()
		_ = os.Remove(secretPath)
		return nil, fmt.Errorf("writing machine.secret: %w", err)
	}
	_ = f.Close()
	return secret, nil
}

func ksEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func ksDecrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	return gcm.Open(nil, nonce, ciphertext[gcm.NonceSize():], nil)
}
