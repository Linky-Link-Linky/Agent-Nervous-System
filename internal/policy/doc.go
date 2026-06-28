// Package policy implements the YAML-based allow/deny policy engine for ANS.
// Policies are stored in a SQLite-backed store, parsed from YAML rules, and
// evaluated against flat JSON-like action representations. Supports
// nociception (honor-based warnings) and per-rule allow/deny effects.
package policy
