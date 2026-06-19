// Package snapshot — SQLite-backed metadata store for agent state snapshots.
// SPDX-License-Identifier: MIT
package snapshot

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const snapshotSchema = `
CREATE TABLE IF NOT EXISTS snapshots (
	snapshot_id   TEXT    NOT NULL PRIMARY KEY,
	chain_index   INTEGER NOT NULL,
	agent_id      TEXT    NOT NULL,
	receipt_id    TEXT    NOT NULL DEFAULT '',
	snap_type     TEXT    NOT NULL,
	storage_path  TEXT    NOT NULL,
	size_bytes    INTEGER NOT NULL DEFAULT 0,
	hash          TEXT    NOT NULL,
	timestamp_ns  INTEGER NOT NULL,
	metadata      TEXT    NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_snap_agent ON snapshots(agent_id);
CREATE INDEX IF NOT EXISTS idx_snap_chain  ON snapshots(chain_index);
CREATE INDEX IF NOT EXISTS idx_snap_type   ON snapshots(snap_type);
`

func (s *Store) save(sn *Snapshot) error {
	metaJSON, err := json.Marshal(sn.Metadata)
	if err != nil {
		return fmt.Errorf("marshalling metadata: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO snapshots
		(snapshot_id, chain_index, agent_id, receipt_id, snap_type, storage_path, size_bytes, hash, timestamp_ns, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		sn.ID, sn.ChainIndex, sn.AgentID, sn.ReceiptID, string(sn.SnapType),
		sn.StoragePath, sn.SizeBytes, sn.Hash, sn.TimestampNS, string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("inserting snapshot: %w", err)
	}
	return nil
}

func (s *Store) getByChainIndex(snapType SnapType, chainIndex uint64) (*Snapshot, error) {
	var metaJSON string
	sn := &Snapshot{}
	err := s.db.QueryRow(
		`SELECT snapshot_id, chain_index, agent_id, receipt_id, snap_type, storage_path, size_bytes, hash, timestamp_ns, metadata
		FROM snapshots WHERE chain_index=? AND snap_type=?`, chainIndex, string(snapType),
	).Scan(&sn.ID, &sn.ChainIndex, &sn.AgentID, &sn.ReceiptID, (*string)(&sn.SnapType),
		&sn.StoragePath, &sn.SizeBytes, &sn.Hash, &sn.TimestampNS, &metaJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no %s snapshot at chain index %d", snapType, chainIndex)
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(metaJSON), &sn.Metadata); err != nil {
		sn.Metadata = make(map[string]string)
	}
	return sn, nil
}

func (s *Store) list(agentID string, snapType SnapType, limit, offset int) ([]*Snapshot, error) {
	q := `SELECT snapshot_id, chain_index, agent_id, receipt_id, snap_type, storage_path, size_bytes, hash, timestamp_ns, metadata
		FROM snapshots WHERE agent_id=? AND snap_type=? ORDER BY chain_index DESC`
	args := []interface{}{agentID, string(snapType)}
	if limit > 0 {
		q += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []*Snapshot
	for rows.Next() {
		var metaJSON string
		sn := &Snapshot{}
		if err := rows.Scan(&sn.ID, &sn.ChainIndex, &sn.AgentID, &sn.ReceiptID, (*string)(&sn.SnapType),
			&sn.StoragePath, &sn.SizeBytes, &sn.Hash, &sn.TimestampNS, &metaJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metaJSON), &sn.Metadata); err != nil {
			sn.Metadata = make(map[string]string)
		}
		snaps = append(snaps, sn)
	}
	return snaps, rows.Err()
}
