package snapshot

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// mockSnap is a Snapshotter implementation for testing Store logic.
type mockSnap struct {
	typ      SnapType
	captures int
}

func (m *mockSnap) Type() SnapType { return m.typ }
func (m *mockSnap) Capture(agentID string, chainIndex uint64, storePath string) (*Snapshot, error) {
	m.captures++
	return &Snapshot{
		AgentID:    agentID,
		ChainIndex: chainIndex,
		SnapType:   m.typ,
		StoragePath: storePath,
		SizeBytes:  100,
		Hash:       "mockhash",
		TimestampNS: 1,
		Metadata:   map[string]string{"mock": "true"},
	}, nil
}
func (m *mockSnap) Restore(snap *Snapshot) error { return nil }

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewStore(t *testing.T) {
	db := openTestDB(t)
	baseDir := t.TempDir()
	s, err := NewStore(db, baseDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
}

func TestRegisterAndCaptureMock(t *testing.T) {
	db := openTestDB(t)
	s, err := NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	m := &mockSnap{typ: SnapMemory}
	s.RegisterSnapshotter(m)

	sn, err := s.Capture(SnapMemory, "ans_test", 1, "")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if sn.AgentID != "ans_test" {
		t.Errorf("AgentID=%q, want %q", sn.AgentID, "ans_test")
	}
	if sn.ChainIndex != 1 {
		t.Errorf("ChainIndex=%d, want 1", sn.ChainIndex)
	}
	if sn.SnapType != SnapMemory {
		t.Errorf("SnapType=%q, want %q", sn.SnapType, SnapMemory)
	}
	if len(sn.ID) != 32 {
		t.Errorf("ID length=%d, want 32 hex chars", len(sn.ID))
	}
	if m.captures != 1 {
		t.Errorf("mock.Captures=%d, want 1", m.captures)
	}
}

func TestCaptureUnregisteredType(t *testing.T) {
	db := openTestDB(t)
	s, err := NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_, err = s.Capture(SnapDatabase, "x", 1, "")
	if err == nil || !strings.Contains(err.Error(), "no snapshotter registered") {
		t.Fatalf("expected error for unregistered type, got: %v", err)
	}
}

func TestListSnapshots(t *testing.T) {
	db := openTestDB(t)
	s, err := NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	for i := 1; i <= 5; i++ {
		_, err := s.Capture(SnapFileSystem, "ans_agent", uint64(i), "")
		if err != nil {
			t.Fatalf("Capture %d: %v", i, err)
		}
	}

	snaps, err := s.List("ans_agent", SnapFileSystem, 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 5 {
		t.Errorf("List returned %d, want 5", len(snaps))
	}
	// Should be DESC order
	for i := 1; i < len(snaps); i++ {
		if snaps[i-1].ChainIndex < snaps[i].ChainIndex {
			t.Errorf("not in DESC order: %d < %d", snaps[i-1].ChainIndex, snaps[i].ChainIndex)
		}
	}
}

func TestListWithLimits(t *testing.T) {
	db := openTestDB(t)
	s, err := NewStore(db, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	for i := 1; i <= 10; i++ {
		s.Capture(SnapFileSystem, "ans_agent", uint64(i), "")
	}

	tests := []struct {
		limit, offset, want int
	}{
		{3, 0, 3},
		{5, 2, 5},
		{100, 0, 10},
		{-1, 0, 10},
		{5, -1, 5},
		{0, 0, 10}, // 0 limit = no limit, returns all
	}
	for _, tt := range tests {
		snaps, err := s.List("ans_agent", SnapFileSystem, tt.limit, tt.offset)
		if err != nil {
			t.Fatalf("List(%d,%d): %v", tt.limit, tt.offset, err)
		}
		if len(snaps) != tt.want {
			t.Errorf("List(%d,%d) = %d, want %d", tt.limit, tt.offset, len(snaps), tt.want)
		}
	}
}

func TestListByAgent(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	s.Capture(SnapFileSystem, "ans_foo", 1, "")
	s.Capture(SnapFileSystem, "ans_bar", 2, "")

	snaps, _ := s.List("ans_foo", SnapFileSystem, 10, 0)
	if len(snaps) != 1 || snaps[0].AgentID != "ans_foo" {
		t.Errorf("expected 1 snap for ans_foo, got %d", len(snaps))
	}
	snaps, _ = s.List("ans_baz", SnapFileSystem, 10, 0)
	if len(snaps) != 0 {
		t.Errorf("expected 0 snaps for ans_baz, got %d", len(snaps))
	}
}

func TestRestoreByIndex(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	s.Capture(SnapFileSystem, "ans_test", 1, "")
	err := s.Restore(SnapFileSystem, 1)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
}

func TestRestoreByIndexNotFound(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	err := s.Restore(SnapFileSystem, 99)
	if err == nil || !strings.Contains(err.Error(), "no filesystem snapshot at chain index 99") {
		t.Fatalf("expected error for missing snapshot, got: %v", err)
	}
}

func TestRestoreAllOrder(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())

	restoreOrder := make([]SnapType, 0)
	dbSnap := &mockSnap{typ: SnapDatabase}
	fsSnap := &mockSnap{typ: SnapFileSystem}
	memSnap := &mockSnap{typ: SnapMemory}
	s.RegisterSnapshotter(dbSnap)
	s.RegisterSnapshotter(fsSnap)
	s.RegisterSnapshotter(memSnap)

	s.Capture(SnapDatabase, "ans_test", 1, "")
	s.Capture(SnapFileSystem, "ans_test", 1, "")
	s.Capture(SnapMemory, "ans_test", 1, "")

	// Override Restore to track order
	snaps := s.snaps
	for _, st := range []SnapType{SnapDatabase, SnapFileSystem, SnapMemory} {
		orig := snaps[st]
		snaps[st] = &restoreTracker{Snapshotter: orig, order: &restoreOrder}
	}

	err := s.RestoreAll(1)
	if err != nil {
		t.Fatalf("RestoreAll: %v", err)
	}
	if len(restoreOrder) != 3 {
		t.Fatalf("expected 3 restores, got %d", len(restoreOrder))
	}
	// Must be Database first, then Filesystem, then Memory
	if restoreOrder[0] != SnapDatabase {
		t.Errorf("restore order[0]=%q, want %q", restoreOrder[0], SnapDatabase)
	}
	if restoreOrder[1] != SnapFileSystem {
		t.Errorf("restore order[1]=%q, want %q", restoreOrder[1], SnapFileSystem)
	}
	if restoreOrder[2] != SnapMemory {
		t.Errorf("restore order[2]=%q, want %q", restoreOrder[2], SnapMemory)
	}
}

type restoreTracker struct {
	Snapshotter
	order *[]SnapType
}

func (r *restoreTracker) Restore(snap *Snapshot) error {
	*r.order = append(*r.order, snap.SnapType)
	return r.Snapshotter.Restore(snap)
}

func TestDelete(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	for i := 1; i <= 5; i++ {
		s.Capture(SnapFileSystem, "ans_test", uint64(i), "")
	}
	if err := s.Delete(3); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	snaps, _ := s.List("ans_test", SnapFileSystem, 10, 0)
	if len(snaps) != 2 {
		t.Errorf("after Delete(3): %d snaps, want 2", len(snaps))
	}
}

func TestCaptureSetsID(t *testing.T) {
	db := openTestDB(t)
	s, _ := NewStore(db, t.TempDir())
	m := &mockSnap{typ: SnapFileSystem}
	s.RegisterSnapshotter(m)

	s1, _ := s.Capture(SnapFileSystem, "ans_a", 1, "")
	s2, _ := s.Capture(SnapFileSystem, "ans_a", 2, "")
	if s1.ID == s2.ID {
		t.Error("different snapshots have the same ID")
	}
	// Same agent+index+type but different time should produce different IDs
	s3, _ := s.Capture(SnapFileSystem, "ans_b", 1, "")
	if s1.ID == s3.ID {
		t.Error("different agents produced same snapshot ID")
	}
}
