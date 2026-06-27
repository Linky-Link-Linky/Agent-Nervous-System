package identity

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	agent, err := New("test-agent", "1.0.0", "test-owner")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if !strings.HasPrefix(agent.ID, AgentIDPrefix) {
		t.Errorf("ID does not start with %q: got %q", AgentIDPrefix, agent.ID)
	}
	if len(agent.ID) < 8 {
		t.Errorf("ID too short: %q (len=%d)", agent.ID, len(agent.ID))
	}
	if agent.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", agent.Name, "test-agent")
	}
	if len(agent.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("PublicKey size = %d, want %d", len(agent.PublicKey), ed25519.PublicKeySize)
	}
	if len(agent.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("PrivateKey size = %d, want %d", len(agent.PrivateKey), ed25519.PrivateKeySize)
	}
}

func TestSignVerify(t *testing.T) {
	agent, err := New("signer", "1.0.0", "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	message := make([]byte, 32)
	for i := range message {
		message[i] = byte(i)
	}
	sig := agent.Sign(message)
	if !agent.Verify(message, sig) {
		t.Error("Verify() returned false for valid signature")
	}
	// Mutate message
	message[0] ^= 0xFF
	if agent.Verify(message, sig) {
		t.Error("Verify() returned true for tampered message")
	}
}

func TestKeystoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	agent, err := New("save-load-test", "1.0.0", "owner1")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if err := ks.Save(agent); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	loaded, err := ks.Load(agent.ID)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if loaded.ID != agent.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, agent.ID)
	}
	if loaded.Name != agent.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, agent.Name)
	}
	if loaded.Version != agent.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, agent.Version)
	}
	if string(loaded.PublicKey) != string(agent.PublicKey) {
		t.Error("PublicKey mismatch")
	}
	if string(loaded.PrivateKey) != string(agent.PrivateKey) {
		t.Error("PrivateKey mismatch")
	}
}

func TestKeystoreList(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	a1, _ := New("agent1", "1.0.0", "")
	a2, _ := New("agent2", "1.0.0", "")
	a3, _ := New("agent3", "1.0.0", "")
	ks.Save(a1)
	ks.Save(a2)
	ks.Save(a3)
	ids, err := ks.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("List() returned %d agents, want 3", len(ids))
	}
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	for _, want := range []string{a1.ID, a2.ID, a3.ID} {
		if !idMap[want] {
			t.Errorf("List() missing agent ID %q", want)
		}
	}
}

func TestDeriveIDDeterministic(t *testing.T) {
	a1, err := New("deterministic", "1.0.0", "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	a2 := NewFromKeys(a1.Name, a1.Version, a1.Owner, a1.PublicKey, a1.PrivateKey)
	if a2.ID != a1.ID {
		t.Errorf("Reconstructed agent ID = %q, want %q", a2.ID, a1.ID)
	}
}

func TestKeystoreDelete(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	agent, _ := New("delete-test", "1.0.0", "")
	ks.Save(agent)
	if err := ks.Delete(agent.ID); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}
	_, err = ks.Load(agent.ID)
	if err == nil {
		t.Error("Load() succeeded after Delete(), want error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Load() error = %v, want 'not found'", err)
	}
}

func TestKeystoreListEmpty(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	ids, err := ks.List()
	if err != nil {
		t.Fatalf("List() failed on empty keystore: %v", err)
	}
	if ids != nil && len(ids) != 0 {
		t.Errorf("List() = %v, want nil or empty slice", ids)
	}
}

func TestRotate(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	agent, _ := New("rotate-test", "1.0.0", "owner")
	ks.Save(agent)
	oldID := agent.ID

	newAgent, rec, err := ks.Rotate(agent.ID)
	if err != nil {
		t.Fatalf("Rotate() failed: %v", err)
	}
	if newAgent.ID == oldID {
		t.Error("New agent ID is same as old ID")
	}
	if err := VerifyRotation(rec); err != nil {
		t.Errorf("VerifyRotation() failed: %v", err)
	}

	// Verify old key file still present
	oldPath := filepath.Join(dir, oldID+".key")
	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("Old key file missing: %v", err)
	}

	// Verify new key file present
	newPath := filepath.Join(dir, newAgent.ID+".key")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("New key file missing: %v", err)
	}
}

func TestVerifyRotationTampered(t *testing.T) {
	dir := t.TempDir()
	ks, err := NewKeystore(dir)
	if err != nil {
		t.Fatalf("NewKeystore() failed: %v", err)
	}
	agent, _ := New("tamper-test", "1.0.0", "")
	ks.Save(agent)

	_, rec, err := ks.Rotate(agent.ID)
	if err != nil {
		t.Fatalf("Rotate() failed: %v", err)
	}

	// Tamper with new public key
	rec.NewPublicKey = rec.NewPublicKey[:len(rec.NewPublicKey)-2] + "FF"

	err = VerifyRotation(rec)
	if err == nil {
		t.Error("VerifyRotation() succeeded on tampered record, want error")
	}
}
