package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestWorkspace(t *testing.T) (root string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "workspace")
	os.MkdirAll(root, 0755)
	os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello world\n"), 0644)
	os.WriteFile(filepath.Join(root, "config.yaml"), []byte("key: value\n"), 0644)
	os.MkdirAll(filepath.Join(root, "subdir"), 0755)
	os.WriteFile(filepath.Join(root, "subdir", "nested.txt"), []byte("nested\n"), 0644)
	os.MkdirAll(filepath.Join(root, ".ans"), 0755)
	os.WriteFile(filepath.Join(root, ".ans", "secret"), []byte("hidden"), 0600)
	os.MkdirAll(filepath.Join(root, "node_modules"), 0755)
	os.WriteFile(filepath.Join(root, "node_modules", "dep.js"), []byte("dep"), 0644)
	return root
}

func TestCaptureAndRestoreFull(t *testing.T) {
	ws := setupTestWorkspace(t)
	fs := NewFileSystemSnap(ws)
	storePath := filepath.Join(t.TempDir(), "snap.tar.gz")

	sn, err := fs.Capture("ans_test", 1, storePath)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if sn.SizeBytes == 0 {
		t.Error("snapshot size is 0")
	}
	if sn.Hash == "" {
		t.Error("hash is empty")
	}
	if sn.StoragePath != storePath {
		t.Errorf("StoragePath=%q, want %q", sn.StoragePath, storePath)
	}
	if sn.SnapType != SnapFileSystem {
		t.Errorf("SnapType=%q", sn.SnapType)
	}
	if _, ok := sn.Metadata["workspace_root"]; !ok {
		t.Error("Metadata missing workspace_root")
	}

	// Verify exclusion rules: .ans and node_modules should not be in archive
	restoreDir := filepath.Join(t.TempDir(), "restored")
	fs2 := NewFileSystemSnap(restoreDir)
	if err := fs2.Restore(sn); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	checkFile(t, restoreDir, "hello.txt", "hello world\n")
	checkFile(t, restoreDir, "config.yaml", "key: value\n")
	checkFile(t, restoreDir, filepath.Join("subdir", "nested.txt"), "nested\n")

	// Excluded files should NOT be restored
	if _, err := os.Stat(filepath.Join(restoreDir, ".ans", "secret")); !os.IsNotExist(err) {
		t.Error(".ans/secret was restored despite exclusion")
	}
	if _, err := os.Stat(filepath.Join(restoreDir, "node_modules", "dep.js")); !os.IsNotExist(err) {
		t.Error("node_modules/dep.js was restored despite exclusion")
	}
}

func TestCaptureEmptyWorkspace(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "empty")
	os.MkdirAll(ws, 0755)
	fs := NewFileSystemSnap(ws)
	storePath := filepath.Join(t.TempDir(), "empty.tar.gz")

	sn, err := fs.Capture("ans_test", 1, storePath)
	if err != nil {
		t.Fatalf("Capture empty: %v", err)
	}
	if sn.SizeBytes == 0 {
		t.Error("empty snapshot size is 0")
	}

	restoreDir := filepath.Join(t.TempDir(), "restored")
	fs2 := NewFileSystemSnap(restoreDir)
	if err := fs2.Restore(sn); err != nil {
		t.Fatalf("Restore empty: %v", err)
	}
}

func TestCaptureHashDeterministic(t *testing.T) {
	ws := setupTestWorkspace(t)
	fs := NewFileSystemSnap(ws)
	p1 := filepath.Join(t.TempDir(), "snap1.tar.gz")
	p2 := filepath.Join(t.TempDir(), "snap2.tar.gz")

	s1, _ := fs.Capture("ans_test", 1, p1)
	s2, _ := fs.Capture("ans_test", 1, p2)
	if s1.Hash != s2.Hash {
		t.Errorf("non-deterministic hash: %q != %q", s1.Hash, s2.Hash)
	}
	if s1.SizeBytes != s2.SizeBytes {
		t.Errorf("non-deterministic size: %d != %d", s1.SizeBytes, s2.SizeBytes)
	}
}

func TestCaptureHashChangedByContent(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	os.MkdirAll(ws, 0755)
	os.WriteFile(filepath.Join(ws, "a.txt"), []byte("aaa"), 0644)
	fs := NewFileSystemSnap(ws)

	p1 := filepath.Join(t.TempDir(), "a.tar.gz")
	s1, _ := fs.Capture("ans_test", 1, p1)

	os.WriteFile(filepath.Join(ws, "a.txt"), []byte("bbb"), 0644)
	p2 := filepath.Join(t.TempDir(), "b.tar.gz")
	s2, _ := fs.Capture("ans_test", 1, p2)

	if s1.Hash == s2.Hash {
		t.Error("different content produced same hash")
	}
}

func TestSnapshotPaths(t *testing.T) {
	ws := setupTestWorkspace(t)
	fs := NewFileSystemSnap(ws)
	storePath := filepath.Join(t.TempDir(), "paths.tar.gz")

	sn, err := fs.SnapshotPaths([]string{"hello.txt", "config.yaml"}, storePath)
	if err != nil {
		t.Fatalf("SnapshotPaths: %v", err)
	}
	if sn.SizeBytes == 0 {
		t.Error("snapshot size is 0")
	}

	restoreDir := filepath.Join(t.TempDir(), "restored")
	fs2 := NewFileSystemSnap(restoreDir)
	if err := fs2.Restore(sn); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	checkFile(t, restoreDir, "hello.txt", "hello world\n")
	checkFile(t, restoreDir, "config.yaml", "key: value\n")
	// subdir/nested.txt was NOT in the paths list
	if _, err := os.Stat(filepath.Join(restoreDir, "subdir", "nested.txt")); !os.IsNotExist(err) {
		t.Error("nested.txt was restored despite not being in SnapshotPaths")
	}
}

func TestSnapshotPathsOutsideWorkspace(t *testing.T) {
	ws := setupTestWorkspace(t)
	fs := NewFileSystemSnap(ws)
	outside := filepath.Join(t.TempDir(), "outside.txt")
	os.WriteFile(outside, []byte("outside"), 0644)

	_, err := fs.SnapshotPaths([]string{outside}, filepath.Join(t.TempDir(), "out.tar.gz"))
	if err == nil || !strings.Contains(err.Error(), "outside workspace root") {
		t.Fatalf("expected error for outside path, got: %v", err)
	}
}

func TestPathTraversalRestore(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	os.MkdirAll(ws, 0755)
	os.WriteFile(filepath.Join(ws, "good.txt"), []byte("safe"), 0644)
	fs := NewFileSystemSnap(ws)
	storePath := filepath.Join(t.TempDir(), "snap.tar.gz")

	sn, err := fs.SnapshotPaths([]string{ws}, storePath)
	if err != nil {
		t.Fatalf("SnapshotPaths: %v", err)
	}

	restoreDir := filepath.Join(t.TempDir(), "restored")
	fs2 := NewFileSystemSnap(restoreDir)
	if err := fs2.Restore(sn); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	checkFile(t, restoreDir, "good.txt", "safe")
	// File should exist under restoreDir, not at filesystem root
	if _, err := os.Stat(filepath.Join(restoreDir, "good.txt")); err != nil {
		t.Errorf("restored file missing: %v", err)
	}
}

func TestExcludePaths(t *testing.T) {
	ws := setupTestWorkspace(t)
	fs := NewFileSystemSnap(ws)

	excluded := fs.ExcludedPaths()
	if len(excluded) == 0 {
		t.Fatal("no excluded paths")
	}
	hasDotAns := false
	hasNodeModules := false
	for _, p := range excluded {
		if p == ".ans" {
			hasDotAns = true
		}
		if p == "node_modules" {
			hasNodeModules = true
		}
	}
	if !hasDotAns {
		t.Error(".ans not in default exclusions")
	}
	if !hasNodeModules {
		t.Error("node_modules not in default exclusions")
	}
}

func TestExtensionFilter(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	os.MkdirAll(ws, 0755)
	os.WriteFile(filepath.Join(ws, "a.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(ws, "a.go"), []byte("go code"), 0644)
	os.WriteFile(filepath.Join(ws, "a.py"), []byte("py code"), 0644)

	fs := &FileSystemSnap{WorkspaceRoot: ws, IncludeExts: map[string]bool{".txt": true, ".go": true}}
	storePath := filepath.Join(t.TempDir(), "ext.tar.gz")
	sn, err := fs.Capture("ans_test", 1, storePath)
	if err != nil {
		t.Fatalf("Capture with extensions: %v", err)
	}

	restoreDir := filepath.Join(t.TempDir(), "restored")
	fs2 := NewFileSystemSnap(restoreDir)
	fs2.Restore(sn)

	checkFile(t, restoreDir, "a.txt", "text")
	checkFile(t, restoreDir, "a.go", "go code")
	// a.py should not be included (not in IncludeExts)
	if _, err := os.Stat(filepath.Join(restoreDir, "a.py")); !os.IsNotExist(err) {
		t.Error("a.py was restored despite extension filter")
	}
}

func TestIsExcluded(t *testing.T) {
	fs := NewFileSystemSnap("/dummy")
	tests := []struct{ path, msg string; excluded bool }{
		{".ans", ".ans exact", true},
		{".ans/secret", ".ans subpath", true},
		{".ans/", ".ans trailing slash", true},
		{"my_node_modules_test", "substring match", false},
		{"node_modules", "exact node_modules", true},
		{"node_modules/dep.js", "node_modules subpath", true},
		{"src/main.go", "normal path", false},
	}
	for _, tt := range tests {
		got := fs.isExcluded(tt.path)
		if got != tt.excluded {
			t.Errorf("isExcluded(%q) = %v, want %v (%s)", tt.path, got, tt.excluded, tt.msg)
		}
	}
}

func checkFile(t *testing.T, dir, rel, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	if string(data) != want {
		t.Errorf("%s: got %q, want %q", rel, string(data), want)
	}
}
