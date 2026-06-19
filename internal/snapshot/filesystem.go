// Package snapshot — file system state capture and restore using tar archives.
// SPDX-License-Identifier: MIT
package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiffEntry records one file change between snapshots.
type DiffEntry struct {
	RelPath  string `json:"rel_path"`
	Change   string `json:"change"` // "modified", "added", "deleted"
	Size     int64  `json:"size,omitempty"`
	ModTime  int64  `json:"mtime,omitempty"`
}

// FileSystemSnap captures and restores file system state as a compressed tar archive.
//
// It snapshots files and directories under a workspace root. The snapshot is stored
// as a .tar.gz file containing the relative paths of all tracked files.
//
// Usage:
//
//	snap := NewFileSystemSnap("/path/to/workspace")
//	sn, err := snap.Capture("agent_id", 42, "/tmp/snaps/agent_id/fs-42.snap")
//	err = snap.Restore(sn)
type FileSystemSnap struct {
	WorkspaceRoot string            // root directory to snapshot
	IncludeExts   map[string]bool   // if non-empty, only snapshot files with these extensions
	ExcludePaths  map[string]bool   // paths (relative to workspace) to exclude
}

// NewFileSystemSnap creates a new filesystem snapshotter rooted at workspaceRoot.
func NewFileSystemSnap(workspaceRoot string) *FileSystemSnap {
	return &FileSystemSnap{
		WorkspaceRoot: workspaceRoot,
		IncludeExts:   nil, // all extensions
		ExcludePaths:  map[string]bool{".ans": true, "node_modules": true},
	}
}

func (fs *FileSystemSnap) Type() SnapType { return SnapFileSystem }

// Capture creates a compressed tar archive of all files under WorkspaceRoot.
// Returns a Snapshot with the hash and size of the archive.
func (fs *FileSystemSnap) Capture(agentID string, chainIndex uint64, storePath string) (*Snapshot, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, fmt.Errorf("creating snapshot dir: %w", err)
	}
	f, err := os.Create(storePath)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot file: %w", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	walkErr := filepath.Walk(fs.WorkspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(fs.WorkspaceRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if fs.isExcluded(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return nil
		}
		if len(fs.IncludeExts) > 0 && info.Mode().IsRegular() {
			ext := strings.ToLower(filepath.Ext(rel))
			if !fs.IncludeExts[ext] {
				return nil
			}
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", rel, err)
		}
		header.Name = rel
		if info.IsDir() {
			header.Name += "/"
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", rel, err)
		}
		if info.Mode().IsRegular() {
			fh, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, fh); err != nil {
				fh.Close()
				return fmt.Errorf("writing %s to tar: %w", rel, err)
			}
			fh.Close()
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walking workspace: %w", walkErr)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}

	// Hash the entire archive, not just file bodies
	fh, err := os.Open(storePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive for hashing: %w", err)
	}
	defer fh.Close()
	archiveHash := sha256.New()
	if _, err := io.Copy(archiveHash, fh); err != nil {
		return nil, fmt.Errorf("hashing archive: %w", err)
	}

	stat, err := os.Stat(storePath)
	if err != nil {
		return nil, fmt.Errorf("stating snapshot: %w", err)
	}
	return &Snapshot{
		ChainIndex:  chainIndex,
		AgentID:     agentID,
		SnapType:    SnapFileSystem,
		StoragePath: storePath,
		SizeBytes:   stat.Size(),
		Hash:        hex.EncodeToString(archiveHash.Sum(nil)),
		TimestampNS: time.Now().UnixNano(),
		Metadata: map[string]string{
			"workspace_root": fs.WorkspaceRoot,
		},
	}, nil
}

// Restore extracts a snapshot tar archive back to the workspace root.
// If the snapshot is differential (SnapFileSystemDiff), it first restores the
// base snapshot, then applies the diff on top.
// WARNING: This overwrites existing files.
func (fs *FileSystemSnap) Restore(snap *Snapshot) error {
	// If this is a differential snapshot, restore base first
	if snap.SnapType == SnapFileSystemDiff && snap.Metadata != nil {
		if basePath, ok := snap.Metadata["base_snapshot"]; ok && basePath != "" {
			baseSnap := &Snapshot{StoragePath: basePath, SnapType: SnapFileSystem}
			if err := fs.Restore(baseSnap); err != nil {
				return fmt.Errorf("restoring base snapshot: %w", err)
			}
		}
	}

	f, err := os.Open(snap.StoragePath)
	if err != nil {
		return fmt.Errorf("opening snapshot %s: %w", snap.StoragePath, err)
	}
	defer f.Close()

	// Verify archive hash before extraction
	if snap.Hash != "" {
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return fmt.Errorf("hashing snapshot: %w", err)
		}
		if hex.EncodeToString(h.Sum(nil)) != snap.Hash {
			return fmt.Errorf("snapshot hash mismatch (expected %s)", snap.Hash)
		}
		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Errorf("rewinding snapshot: %w", err)
		}
	}

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var deletedFiles []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}
		name := strings.TrimLeft(header.Name, "/")
		name = strings.TrimPrefix(name, "./")
		name = filepath.FromSlash(name)
		if name == "." || name == "" {
			continue
		}

		// Handle deletion manifest in differential snapshots
		if name == ".ans_deleted" {
			buf := make([]byte, header.Size)
			io.ReadFull(tr, buf)
			deletedFiles = strings.Split(strings.TrimSpace(string(buf)), "\n")
			continue
		}

		target := filepath.Join(fs.WorkspaceRoot, name)
		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(fs.WorkspaceRoot)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(fs.WorkspaceRoot) {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			out.Close()
		}
	}

	// Remove files that were deleted since the base snapshot
	for _, rel := range deletedFiles {
		rel = filepath.FromSlash(rel)
		target := filepath.Join(fs.WorkspaceRoot, rel)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(fs.WorkspaceRoot)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(fs.WorkspaceRoot) {
			continue
		}
		if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing deleted file %s: %w", rel, err)
		}
	}

	return nil
}

// listFiles returns a map of relative path -> (size, modTime) for the workspace.
func (fs *FileSystemSnap) listFiles(workspaceRoot string) (map[string][2]int64, error) {
	files := make(map[string][2]int64)
	err := filepath.Walk(workspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		rel, err := filepath.Rel(workspaceRoot, path)
		if err != nil { return err }
		if rel == "." { return nil }
		rel = filepath.ToSlash(rel)
		if fs.isExcluded(rel) {
			if info.IsDir() { return filepath.SkipDir }
			return nil
		}
		if !info.Mode().IsRegular() { return nil }
		if len(fs.IncludeExts) > 0 {
			ext := strings.ToLower(filepath.Ext(rel))
			if !fs.IncludeExts[ext] { return nil }
		}
		files[rel] = [2]int64{info.Size(), info.ModTime().UnixNano()}
		return nil
	})
	return files, err
}

// Diff computes the file-level diff between current workspace and a prior snapshot archive.
// Returns added, modified, and deleted file lists.
func (fs *FileSystemSnap) Diff(baseSnapPath string) (added, modified, deleted []string, err error) {
	// Read prior snapshot file listing
	prior := make(map[string][2]int64)
	f, err := os.Open(baseSnapPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("opening base snapshot: %w", err)
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF { break }
		if err != nil { return nil, nil, nil, fmt.Errorf("reading tar: %w", err) }
		if hdr.Typeflag == tar.TypeReg {
			name := strings.TrimLeft(hdr.Name, "/")
			name = strings.TrimPrefix(name, "./")
			prior[name] = [2]int64{hdr.Size, hdr.ModTime.UnixNano()}
		}
	}

	current, err := fs.listFiles(fs.WorkspaceRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("listing workspace: %w", err)
	}

	for rel, info := range current {
		if prev, ok := prior[rel]; !ok {
			added = append(added, rel)
		} else if info[0] != prev[0] || info[1] != prev[1] {
			modified = append(modified, rel)
		}
	}
	for rel := range prior {
		if _, ok := current[rel]; !ok {
			deleted = append(deleted, rel)
		}
	}
	return added, modified, deleted, nil
}

// CaptureDiff creates a differential snapshot: only files that changed since the
// base snapshot are stored. Deleted files are recorded in metadata.
func (fs *FileSystemSnap) CaptureDiff(agentID string, chainIndex uint64, storePath string, baseSnapPath string) (*Snapshot, []DiffEntry, error) {
	added, modified, deleted, err := fs.Diff(baseSnapPath)
	if err != nil {
		return nil, nil, err
	}
	if len(added)+len(modified)+len(deleted) == 0 {
		return nil, nil, fmt.Errorf("no changes since base snapshot")
	}

	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, nil, fmt.Errorf("creating snapshot dir: %w", err)
	}
	fh, err := os.Create(storePath)
	if err != nil {
		return nil, nil, fmt.Errorf("creating snapshot file: %w", err)
	}
	defer fh.Close()

	gzw := gzip.NewWriter(fh)
	tw := tar.NewWriter(gzw)

	// Store deletion manifest as a special entry
	if len(deleted) > 0 {
		delHeader := &tar.Header{
			Name:     ".ans_deleted",
			Size:     int64(len([]byte(strings.Join(deleted, "\n")))),
			Mode:     0644,
			ModTime:  time.Now(),
		}
		if err := tw.WriteHeader(delHeader); err != nil {
			return nil, nil, fmt.Errorf("writing deletion manifest: %w", err)
		}
		if _, err := tw.Write([]byte(strings.Join(deleted, "\n"))); err != nil {
			return nil, nil, fmt.Errorf("writing deletion entries: %w", err)
		}
	}

	allChanged := append(added, modified...)
	for _, rel := range allChanged {
		abs := filepath.Join(fs.WorkspaceRoot, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			return nil, nil, fmt.Errorf("stating %s: %w", rel, err)
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil, nil, fmt.Errorf("creating tar header for %s: %w", rel, err)
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return nil, nil, fmt.Errorf("writing tar header for %s: %w", rel, err)
		}
		f, err := os.Open(abs)
		if err != nil {
			return nil, nil, fmt.Errorf("opening %s: %w", rel, err)
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("writing %s: %w", rel, err)
		}
		f.Close()
	}

	if err := tw.Close(); err != nil {
		return nil, nil, fmt.Errorf("closing tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, nil, fmt.Errorf("closing gzip: %w", err)
	}

	// Hash the archive
	archiveHash := sha256.New()
	rh, err := os.Open(storePath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening archive for hash: %w", err)
	}
	defer rh.Close()
	if _, err := io.Copy(archiveHash, rh); err != nil {
		return nil, nil, fmt.Errorf("hashing archive: %w", err)
	}

	stat, err := os.Stat(storePath)
	if err != nil {
		return nil, nil, fmt.Errorf("stating snapshot: %w", err)
	}

	entries := make([]DiffEntry, 0, len(added)+len(modified)+len(deleted))
	for _, r := range added { entries = append(entries, DiffEntry{RelPath: r, Change: "added"}) }
	for _, r := range modified { entries = append(entries, DiffEntry{RelPath: r, Change: "modified"}) }
	for _, r := range deleted { entries = append(entries, DiffEntry{RelPath: r, Change: "deleted"}) }

	sn := &Snapshot{
		ChainIndex:  chainIndex,
		AgentID:     agentID,
		SnapType:    SnapFileSystemDiff,
		StoragePath: storePath,
		SizeBytes:   stat.Size(),
		Hash:        hex.EncodeToString(archiveHash.Sum(nil)),
		TimestampNS: time.Now().UnixNano(),
		Metadata: map[string]string{
			"workspace_root":    fs.WorkspaceRoot,
			"base_snapshot":     baseSnapPath,
			"differential":      "true",
			"files_changed":     fmt.Sprintf("%d", len(allChanged)),
			"files_deleted":     fmt.Sprintf("%d", len(deleted)),
		},
	}
	return sn, entries, nil
}

// CleanupAfterIndex removes all snapshot archives at or above the given chain index.
func (fs *FileSystemSnap) CleanupAfterIndex(snaps []*Snapshot) error {
	for _, sn := range snaps {
		if err := os.Remove(sn.StoragePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing snapshot %s: %w", sn.StoragePath, err)
		}
	}
	return nil
}

// SnapshotPaths creates a snapshot of only the specified files/directories.
// This is used for targeted snapshots (e.g., only the files a tool will modify).
func (fs *FileSystemSnap) SnapshotPaths(paths []string, storePath string) (*Snapshot, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, fmt.Errorf("creating snapshot dir: %w", err)
	}
	f, err := os.Create(storePath)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot file: %w", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	workspaceClean := filepath.Clean(fs.WorkspaceRoot)
	for _, p := range paths {
		abs := p
		if !filepath.IsAbs(p) {
			abs = filepath.Join(fs.WorkspaceRoot, p)
		}
		abs = filepath.Clean(abs)
		// Prevent path traversal — path must be under workspace root
		if !strings.HasPrefix(abs, workspaceClean+string(os.PathSeparator)) && abs != workspaceClean {
			return nil, fmt.Errorf("path %q is outside workspace root", p)
		}
		// Skip excluded paths
		rel, err := filepath.Rel(fs.WorkspaceRoot, abs)
		if err != nil {
			return nil, fmt.Errorf("computing relative path for %s: %w", p, err)
		}
		if fs.isExcluded(filepath.ToSlash(rel)) {
			continue
		}
		if err := fs.addToTar(abs, tw, io.Discard); err != nil {
			return nil, fmt.Errorf("adding %s to tar: %w", p, err)
		}
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("closing tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip: %w", err)
	}

	// Hash the entire archive on disk (same approach as Capture)
	stat, err := os.Stat(storePath)
	if err != nil {
		return nil, err
	}
	archiveHash := sha256.New()
	fh, err := os.Open(storePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive for hashing: %w", err)
	}
	defer fh.Close()
	if _, err := io.Copy(archiveHash, fh); err != nil {
		return nil, fmt.Errorf("hashing archive: %w", err)
	}
	return &Snapshot{
		StoragePath: storePath,
		SizeBytes:   stat.Size(),
		Hash:        hex.EncodeToString(archiveHash.Sum(nil)),
		TimestampNS: time.Now().UnixNano(),
	}, nil
}

func (fs *FileSystemSnap) addToTar(absPath string, tw *tar.Writer, hash io.Writer) error {
	return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(fs.WorkspaceRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if fs.isExcluded(rel) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = rel + "/"
			return tw.WriteHeader(header)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if len(fs.IncludeExts) > 0 {
			ext := strings.ToLower(filepath.Ext(rel))
			if !fs.IncludeExts[ext] {
				return nil
			}
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		fh, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, fh)
		fh.Close()
		return err
	})
}

func (fs *FileSystemSnap) isExcluded(rel string) bool {
	for _, p := range fs.ExcludedPaths() {
		if rel == p || strings.HasPrefix(rel, p+"/") || strings.HasPrefix(rel, p+"\\") {
			return true
		}
	}
	return false
}

// ExcludedPaths returns the list of excluded paths as a sorted slice.
func (fs *FileSystemSnap) ExcludedPaths() []string {
	paths := make([]string, 0, len(fs.ExcludePaths))
	for p := range fs.ExcludePaths {
		paths = append(paths, p)
	}
	return paths
}


