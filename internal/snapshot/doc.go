// Package snapshot implements filesystem snapshot and diff for ANS time-travel.
// Snapshots are stored as JSON directory dumps in ~/.ans/snapshots/. The
// FileSystemSnap driver walks workspace paths and records file metadata and
// content hashes. Diff detects added, removed, and modified files by name
// and hash comparison.
package snapshot
