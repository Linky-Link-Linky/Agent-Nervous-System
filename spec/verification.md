# ANS Symbolic Verification

The ANS chain guarantees seven properties under Go's memory model at runtime.

## 1. Append-only (no deletion, no mutation)

**Go guarantee:** `Chain.db` is opened with SQLite WAL mode. Receipts are only
ever inserted via `INSERT INTO receipts`. Once written, a row is never
overwritten or deleted (except by `Prune` which replaces a range with a Merkle
anchor). The mutex (`sync.Mutex`) serialises all appends, preventing data races.

See `Chain.mu` and `Chain.AppendNew` in `internal/chain/chain.go`.

**Model-checked:** TLA+ `spec/chain.tla` — `AppendOnly` enforced by action
`AppendReceipt` which only ever `Append(chain, newReceipt)`.

**Fuzzed:** `FuzzHashChain` verifies hash chain integrity for arbitrary lengths.

## 2. Strictly increasing chain_index (no gaps, no duplicates)

**Go guarantee:** `AppendNew` holds `c.mu`, reads `c.lastIdx`, increments it,
and assigns the new value as `chain_index`. Because the mutex serialises all
appends, each goroutine sees a unique, strictly increasing index starting at 1.

Concretely: after N receipts, `chain_index` values are exactly `{1, 2, ..., N}`.

**Verified by:** `TestConcurrentOrdering` — 10 goroutines × 25 receipts,
checks every `chain_index` matches its position and `Count()` equals `N`.

## 3. Hash chain integrity

**Go guarantee:** `PrevReceiptHash` of receipt `i` = `ComputeHash(receipt[i-1])`.
`ComputeHash` marshals the receipt as JSON and SHA-256 digests it. Since the
JSON marshaller is deterministic for the `Receipt` struct (no maps, no
`json:"omitempty"` on non-zero values), the hash is stable.

**Verified by:** `TestAppendNew` (sequential) and `TestConcurrentOrdering`
(concurrent). Receipt `i`'s `PrevReceiptHash` is checked against the hash
of receipt `i-1` for every index.

**Fuzzed:** `FuzzHashChain` verifies random-length chains (1..100 receipts)
with every append creating new random signing keys.

## 4. Genesis receipt anchors to zero-hash

**Go guarantee:** `NewBuilder` receives `GenesisHash` = 64 hex zeros as the
`prevHash` for index 1. This is enforced in `Chain.AppendNew` by
passing `c.lastHash` (initialized to `GenesisHash` in `Open`) to the builder
callback.

## 5. Schema validation at insert time

**Go guarantee:** `validateReceipt(r)` runs inside `AppendNew` after signing
and before the SQL INSERT. It checks every JSON Schema constraint: receipt_id
hex format, agent_id prefix, prev_receipt_hash length, action_type enum,
field max lengths, phase/policy/outcome enums, signature format, and
pre_receipt_id format. Invalid receipts are rejected before reaching the
database.

**Verified by:** Every `AppendNew` call — the daemon tests exercise this with
real receipt data; `FuzzReceiptRoundtrip` validates JSON stability.

## 6. Every receipt is signed by a registered agent

**Go guarantee:** `Chain.AppendNew` requires a `*receipt.Signer` (wrapping
an Ed25519 private key). After signing, `receipt.Verify(r, pub)` must pass.
When `verify --chain` is used, agent public keys are loaded from the keystore
and `VerifyChain` checks every receipt's signature against its agent's key.

## 7. HLC timestamps are strictly monotonic

**Go guarantee:** `clock.HLC.Now()` wraps a `sync.Mutex`. Each call compares
the wall clock (UnixNano) against the maximum physical time seen; if the wall
clock has not advanced, the previous maximum is incremented by 1. This ensures
every emitted timestamp is strictly greater than any previous one, even under
concurrent access. Timestamps remain valid nanoseconds-since-epoch values for
compatibility with all display code.

**Verified by:** `TestHLCIncreases` (1,000 sequential calls) and
`TestHLCConcurrent` (50 concurrent goroutines, all unique timestamps).

---

## Summary table

| Property | Go mechanism | Test | TLA+ |
|---|---|---|---|
| Append-only | SQLite WAL + mutex | `TestConcurrentOrdering` | `AppendOnly` |
| No gaps | serialised `lastIdx` | `TestConcurrentOrdering` | `NoGaps` |
| Hash chain | `ComputeHash` determinism | `TestConcurrentOrdering`, `FuzzHashChain` | `HashChainLinked` |
| Genesis hash | `NewBuilder(genesisHash)` | `TestAppendNew` | `HashChainLinked` |
| Schema validation | `validateReceipt` in `AppendNew` | `FuzzReceiptRoundtrip` | — |
| Agent signature | `Signer` + `Verify` | `TestSignAndVerify` | `AllAgentsRegistered` |
| HLC monotonicity | `sync.Mutex` + nanosecond increment | `TestHLCIncreases` | — |
