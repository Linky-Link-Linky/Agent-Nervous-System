// Package snapshot provides state capture and restore for agent time-travel.
// SPDX-License-Identifier: Apache-2.0
package snapshot

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SnapType categorises what kind of state a snapshot captures.
type SnapType string

const (
	SnapFileSystem     SnapType = "filesystem"
	SnapFileSystemDiff SnapType = "filesystem-diff"
	SnapMemory         SnapType = "memory"
	SnapDatabase       SnapType = "database"
)

// Snapshot is one point-in-time capture of agent state at a given chain index.
// The snapshot itself is stored externally (tar archive, JSON blob, etc.);
// this struct holds the metadata needed to locate and verify it.
type Snapshot struct {
	ID          string            `json:"snapshot_id"`
	ChainIndex  uint64            `json:"chain_index"`
	AgentID     string            `json:"agent_id"`
	ReceiptID   string            `json:"receipt_id,omitempty"`
	SnapType    SnapType          `json:"snap_type"`
	StoragePath string            `json:"storage_path"`
	SizeBytes   int64             `json:"size_bytes"`
	Hash        string            `json:"hash"`
	TimestampNS int64             `json:"timestamp_ns"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Snapshotter is the interface for state capture and restore.
type Snapshotter interface {
	Type() SnapType
	Capture(agentID string, chainIndex uint64, storePath string) (*Snapshot, error)
	Restore(snap *Snapshot) error
}

// Store manages snapshot metadata persistence and orchestrates capture/restore.
type Store struct {
	mu      sync.RWMutex
	db      *sql.DB
	snaps   map[SnapType]Snapshotter
	baseDir string
}

// NewStore opens (or creates) the snapshot store in the chain SQLite DB.
// baseDir is where snapshot archives are stored on disk.
func NewStore(db *sql.DB, baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("creating snapshot base dir: %w", err)
	}
	if _, err := db.Exec(snapshotSchema); err != nil {
		return nil, fmt.Errorf("creating snapshots table: %w", err)
	}
	return &Store{db: db, snaps: make(map[SnapType]Snapshotter), baseDir: baseDir}, nil
}

// RegisterSnapshotter registers a Snapshotter for a given type.
func (s *Store) RegisterSnapshotter(snap Snapshotter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snaps[snap.Type()] = snap
}

// getLatestFullSnapshot returns the most recent full (non-diff) filesystem snapshot for an agent.
func (s *Store) getLatestFullSnapshot(agentID string) (*Snapshot, error) {
	snaps, err := s.list(agentID, SnapFileSystem, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("no full snapshot found for agent %s", agentID)
	}
	return snaps[0], nil
}

// CaptureDifferential captures only files that changed since the latest full snapshot.
// Falls back to full capture if no prior snapshot exists.
func (s *Store) CaptureDifferential(snapType SnapType, agentID string, chainIndex uint64, receiptID string) (*Snapshot, error) {
	s.mu.Lock()
	snap, ok := s.snaps[snapType]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no snapshotter registered for type %q", snapType)
	}
	fsSnap, ok := snap.(*FileSystemSnap)
	if !ok {
		return s.Capture(snapType, agentID, chainIndex, receiptID)
	}

	// Find the latest full snapshot as the diff base
	baseSnap, err := s.getLatestFullSnapshot(agentID)
	if err != nil {
		return s.Capture(snapType, agentID, chainIndex, receiptID)
	}

	storePath := s.snapshotPath(agentID, chainIndex, SnapFileSystemDiff)
	sn, _, err := fsSnap.CaptureDiff(agentID, chainIndex, storePath, baseSnap.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("differential capture: %w", err)
	}
	sn.ReceiptID = receiptID
	sn.TimestampNS = time.Now().UnixNano()
	if sn.Metadata == nil {
		sn.Metadata = make(map[string]string)
	}
	sn.Metadata["agent_id"] = agentID
	// Generate ID
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d-%d", sn.SnapType, agentID, chainIndex, sn.TimestampNS)))
	sn.ID = fmt.Sprintf("%x", h[:16])
	if err := s.save(sn); err != nil {
		return nil, fmt.Errorf("saving diff snapshot metadata: %w", err)
	}
	return sn, nil
}

// Capture creates a snapshot for the given type and agent at chainIndex.
func (s *Store) Capture(snapType SnapType, agentID string, chainIndex uint64, receiptID string) (*Snapshot, error) {
	s.mu.Lock()
	snap, ok := s.snaps[snapType]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no snapshotter registered for type %q", snapType)
	}
	storePath := s.snapshotPath(agentID, chainIndex, snapType)
	sn, err := snap.Capture(agentID, chainIndex, storePath)
	if err != nil {
		return nil, fmt.Errorf("capture %s: %w", snapType, err)
	}
	sn.ChainIndex = chainIndex
	sn.ReceiptID = receiptID
	sn.TimestampNS = time.Now().UnixNano()
	if sn.Metadata == nil {
		sn.Metadata = make(map[string]string)
	}
	sn.Metadata["agent_id"] = agentID
	// Include snap type in the hash input to prevent ID collision between types
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d-%d", snapType, agentID, chainIndex, sn.TimestampNS)))
	sn.ID = fmt.Sprintf("%x", h[:16])
	if err := s.save(sn); err != nil {
		return nil, fmt.Errorf("saving snapshot metadata: %w", err)
	}
	return sn, nil
}

// Restore restores state to the snapshot at the given chain index.
func (s *Store) Restore(snapType SnapType, chainIndex uint64) error {
	sn, err := s.getByChainIndex(snapType, chainIndex)
	if err != nil {
		return fmt.Errorf("snapshot at index %d not found: %w", chainIndex, err)
	}
	s.mu.Lock()
	snap, ok := s.snaps[snapType]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no snapshotter registered for type %q", snapType)
	}
	return snap.Restore(sn)
}

// RestoreAll restores all registered snapshot types to the given chain index.
func (s *Store) RestoreAll(chainIndex uint64) error {
	order := []SnapType{SnapDatabase, SnapFileSystem, SnapMemory}
	for _, st := range order {
		s.mu.Lock()
		_, ok := s.snaps[st]
		s.mu.Unlock()
		if !ok {
			continue
		}
		if err := s.Restore(st, chainIndex); err != nil {
			return fmt.Errorf("restore %s: %w", st, err)
		}
	}
	return nil
}

// List returns snapshots for a given agent, ordered by chain_index DESC.
func (s *Store) List(agentID string, snapType SnapType, limit, offset int) ([]*Snapshot, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	return s.list(agentID, snapType, limit, offset)
}

// Delete removes all snapshots at or above the given chain index.
func (s *Store) Delete(fromIndex uint64) error {
	_, err := s.db.Exec(`DELETE FROM snapshots WHERE chain_index >= ?`, fromIndex)
	return err
}

func (s *Store) snapshotPath(agentID string, chainIndex uint64, snapType SnapType) string {
	// Replace path separators and dots in agentID to prevent path traversal
	safeID := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '.' || r == ':' {
			return '_'
		}
		return r
	}, agentID)
	return filepath.Join(s.baseDir, fmt.Sprintf("%s-%s-%d.snap", safeID, snapType, chainIndex))
}
