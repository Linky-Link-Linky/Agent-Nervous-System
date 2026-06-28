<div align="center">

# ANS — Agent Nervous System

### `git log` for your AI Agents

**Cryptographic audit trails · State rollback · Policy-as-Code · Zero-trust identity · MCP security**

[![Go 1.23+](https://img.shields.io/badge/go-1.23%2B-00ADD8?logo=go)](https://go.dev/)
[![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Python SDK](https://img.shields.io/badge/python-3.10%2B-3776AB?logo=python)](sdks/python/)
[![TypeScript SDK](https://img.shields.io/badge/typescript-5.0%2B-3178C6?logo=typescript)](sdks/typescript/)
[![Go Reference](https://img.shields.io/badge/go-reference-00ADD8?logo=go)](https://pkg.go.dev/github.com/Linky-Link-Linky/Agent-Nervous-System)
[![Built for](https://img.shields.io/badge/built%20for-AI%20Agents-FF6F00)](#)

**Works fully offline. No SaaS, no API keys, no monthly bill.**

</div>

---

## Why ANS?

Agents call tools, read databases, write files, execute code, delegate to sub-agents — and you have **no idea what actually happened**.

| Problem | ANS Solution |
|---------|-------------|
| **Mutable logs** | **Ed25519-signed receipts** — tampering breaks the signature |
| **No chain of custody** | **SHA-256 hash-linked chain** — receipts point to each other |
| **Lost on crash** | **Pre-action receipts** — intent recorded *before* execution |
| **Can't reproduce state** | **Workspace snapshots** — full tar.gz before every action |
| **Interleaved timelines** | **MergeChains** — causal topological ordering |
| **No access control** | **Policy-as-Code** — deny actions before they execute |
| **Static credentials** | **Ephemeral Identity Broker** — 60s scoped tokens |
| **MCP insecure by default** | **MCP Security Proxy** — injection detection, PII redaction, rate limiting |

---

## At a Glance

ANS is a **Go library** that provides the core primitives for agent auditing. Run the daemon programmatically from your Go application, or use one of the 13 SDK integrations (Python, TypeScript, LangChain, Anthropic, OpenAI, etc.) to instrument agents across any language or framework.

| Capability | What It Gives You |
|-----------|------------------|
| **Receipt Chain** | Pre/post Ed25519-signed receipts, hash-linked, SQLite-backed. Tamper-evident audit trail. |
| **Time-Travel** | Full workspace snapshots before every action. Restore any point in time. |
| **Multi-Agent Merge** | Causal topological sort across concurrent sub-agents. No more interleaved log nightmares. |
| **Identity & Keys** | Ed25519 keypairs, AES-256-GCM encrypted, HKDF-SHA256 derived. Keys never leave the daemon. |
| **Merkle Pruning** | Compact old receipts into Merkle anchors. Infinite scale without losing cryptographic integrity. |
| **Export** | JSONL, CSV, TXT, PDF, Parquet. Compliance-ready reports. |
| **Real-Time Streaming** | NDJSON stdout or CloudEvents 1.0 webhooks. Wire into any pipeline. |
| **13 SDK Integrations** | Python, TypeScript, Anthropic, OpenAI, OpenAI Compatible, Gemini, Ollama, LangChain, LangGraph, CrewAI, PydanticAI, MCP, Claude Agent SDK |
| **Policy-as-Code** | JSON-declarative, 10 operators, all/any/none compounds. PII detection. NociceptionError (0x1F). |
| **Compensation** | Register reverse commands. Undo actions with dry-run. No shell injection. |
| **Ephemeral Identity** | Zero-trust broker. 60s TTL, single-use, auto-revoke. Vault, AWS, GCP, Azure, OAuth2. |
| **MCP Security Proxy** | TCP proxy with rate limiting, token budgets, PII redaction, injection detection, policy enforcement, tool approval, audit. |

---

## Packages

| Package | Import | Description |
|---------|--------|-------------|
| **daemon** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon` | Background daemon process — handles receipt signing, policy evaluation, snapshot management, and MCP proxy lifecycle |
| **chain** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain` | Merkle receipt chain with SQLite store — append, verify, prune, export |
| **snapshot** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot` | Filesystem snapshots (tar.gz + SHA-256) with differential capture and time-travel restore |
| **policy** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/policy` | YAML-based allow/deny policy engine with 10 operators and compound conditions |
| **identity** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity` | Ed25519 key management — generation, encryption, signing, rotation |
| **identitybroker** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identitybroker` | Zero-trust ephemeral credential provisioning (Vault, AWS STS, GCP, OAuth2) |
| **mcp** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/mcp` | MCP security proxy — injection detection, PII redaction, rate limiting, policy enforcement |
| **receipt** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt` | Receipt data structures and serialization |
| **client** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client` | HTTP client to daemon API over Unix socket / named pipe |
| **broker** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/broker` | Cloud identity provider abstraction layer (AWS, GCP, Vault) |
| **clock** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/clock` | Wall clock abstraction for deterministic testing |
| **poller** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/poller` | Daemon state polling for real-time updates |
| **model** | `github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model` | Shared data models |

---

## Quick Start — Embed the Daemon

```go
package main

import (
    "log/slog"
    "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
    "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
)

func main() {
    // Open or create the chain store
    store, err := chain.NewStore("~/.ans/chain.db")
    if err != nil {
        slog.Error("chain store", "error", err)
        return
    }
    defer store.Close()

    // Start the daemon
    d, err := daemon.New(store, &daemon.Config{
        WebhookURL: "",
        NDJSON:     false,
    })
    if err != nil {
        slog.Error("daemon init", "error", err)
        return
    }

    // Run (blocks until SIGTERM)
    slog.Info("starting ANS daemon")
    if err := d.Run(); err != nil {
        slog.Error("daemon exited", "error", err)
    }
}
```

See the [SDK integrations](#8-sdk-integrations-13-frameworks) for instrumenting agents from Python, TypeScript, LangChain, Anthropic, OpenAI, and more.

---

## Features

### 1. The Receipt Chain

Every tool call produces **two linked, Ed25519-signed receipts** — one *before* the action (recording intent, policy check, and a workspace snapshot) and one *after* (recording outcome, duration, and the link back to the pre-receipt).

```
                          ┌──────────────────────────────┐
                          │        AGENT EXECUTES         │
                          │   ans_3vQb7uL6x9              │
                          └──────────┬───────────────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
    ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
    │    PRE-RECEIPT   │   │    TOOL CALLS    │   │   POST-RECEIPT   │
    │  ═══════════════ │   │                  │   │ ════════════════ │
    │  agent_id        │   │  file.write      │   │  agent_id        │
    │  action_type     │   │  http.post       │   │  outcome         │
    │  payload_hash ◄──┼───┤  shell.exec ◄────┼───┤  duration_ms     │
    │  policy_decision │   │  db.query        │   │  pre_receipt_id──┼─── link
    │  snapshot_id     │   │  agent.delegate  │   │  timestamp_ns    │   back
    │  timestamp_ns    │   │                  │   │  Ed25519 SIG ────┤
    │  Ed25519 SIG ────┤   └──────────────────┘   └──────────────────┘
    └──────────────────┘
              │                                              │
              └──────────────────────┬──────────────────────┘
                                     ▼
                    ┌───────────────────────────────────┐
                    │        APPEND-ONLY CHAIN          │
                    │   (SQLite, mutex-guarded insert)  │
                    │                                   │
                    │  Receipt 1 ◄── hash ── Receipt 2  │
                    │     ◄── hash ── Receipt 3 ◄── ... │
                    │                                   │
                    │  Each stored with:                │
                    │  • Full JSON body                 │
                    │  • SHA-256 of the receipt         │
                    │  • Pointer to previous hash       │
                    └───────────────────────────────────┘
```

**Key security properties:**
- Every receipt is **Ed25519-signed** by the agent's private key before it leaves the daemon
- Can't forge, delete, or reorder receipts without breaking the hash chain
- Pre-receipts are signed **before** the tool runs — so even if the agent crashes mid-execution, you have proof of intent
- `chain.Verify()` walks every receipt: recomputes hashes, checks chain pointers, verifies signatures against registered public keys

---

### 2. Time-Travel — Full State Rewind

Every pre-action receipt captures a **full workspace snapshot** as a compressed, hashed, and cryptographically bound archive. You can restore the workspace to any point in time.

```
                       What happens:                    What you see:
                                                                 
  ┌──────────┐        ┌──────────────────────┐        ┌──────────────┐
  │ Agent    │        │  Capture snapshot    │        │  Snapshot    │
  │ runs     │───────►│  as compressed       │───────►│  metadata in │
  │ action   │        │  .tar.gz archive     │        │  chain DB    │
  └──────────┘        └──────────────────────┘        └──────────────┘
                              │                              │
                              │ SHA-256                      │ SnapshotID
                              ▼                              ▼
  ┌──────────┐        ┌──────────────────────┐        ┌──────────────┐
  │ You call │        │  Read archive from   │        │  Replay      │
  │ snapshot │───────►│  disk, verify hash,  │───────►│  Receipts to │
  │ restore  │        │  extract to workspace│        │  restore     │
  └──────────┘        └──────────────────────┘        └──────────────┘
```

**How it works:**
- Before every action, the daemon creates a gzipped tar archive of the workspace
- The archive is **SHA-256 hashed** (whole file, not just content — tamper-proof)
- The hash, path, size, and timestamp are stored in the chain DB alongside the receipt
- The `SnapshotID` is embedded in the receipt's `signableFields` — it's **Ed25519-signed** together with the action data
- On restore, the archive is extracted with **full path traversal protection** (no escape exploits)
- `.ans/` and `node_modules/` are automatically excluded from snapshots
- Snapshots survive pruning — you can compact the receipt chain and still restore to any checkpoint

---

### 3. Multi-Agent Merge

When multiple agents (or sub-agents) produce interleaved receipts, `MergeChains` reconstructs a single causal timeline:

1. **Sub-agent receipts** follow the delegation receipt that spawned them
2. **Post-receipts** follow their pre-action pair
3. Remaining receipts are ordered by timestamp

No more reconstructing "what happened" from interleaved log files. The chain captures causality.

---

### 4. Identity & Key Management

| Property | Implementation |
|----------|---------------|
| **Identity** | `ans_` + base58 of SHA-256(Ed25519 public key)[:10] (e.g. `ans_3vQb7uL6x9`) |
| **Signing** | Ed25519 — all receipts are signed by the agent's private key |
| **Key storage** | Encrypted on disk with **AES-256-GCM**, key derived from machine-local secret via HKDF-SHA256 |
| **Rotation** | `identity.Rotate()` generates a new keypair, records the rotation as a signed receipt |
| **Isolation** | SDKs never touch private keys — signing happens in the daemon over a local socket |

---

### 5. Pruning — Infinite Scale

The chain grows forever. When it gets large, compact old receipts into a **Merkle anchor**:

```
store.Prune(ctx, 10000)
// Pruned 10000 receipts (index 1–10000)
// Merkle root: 7c9d8a3f2b1e4f6a8d0c2b4e6f8a0d1c2b4e6f8a
```

After pruning, `chain.Verify()` still works — it accepts the Merkle anchor as proof. Receipt data is removed, but the cryptographic integrity is preserved. **Snapshots at pruned indices are NOT deleted** — you can still restore state at any pruned index.

---

### 6. Export & Compliance

```go
// Human-readable audit report
store.Export(ctx, "audit.pdf", export.FormatPDF)
store.Export(ctx, "audit.txt", export.FormatTXT)

// Analytics-ready
store.Export(ctx, "audit.jsonl", export.FormatJSONL)
store.Export(ctx, "audit.csv", export.FormatCSV)
store.Export(ctx, "audit.parquet", export.FormatParquet)
```

---

### 7. Real-Time Streaming

```go
// Stream every new receipt via channel
receiptCh := daemon.Subscribe(ctx)
for r := range receiptCh {
    log.Printf("new receipt: %s", r.ID)
}

// NDJSON callback
daemon.OnAppend(func(r *receipt.Receipt) {
    json.NewEncoder(os.Stdout).Encode(r)
})

// CloudEvents webhook
daemon.Config.WebhookURL = "https://hooks.example.com/ans"
```

---

### 8. SDK Integrations (13 Frameworks)

| Framework | Integration | Loc | SDK |
|-----------|------------|:---:|:---:|
| **Python native** | `@ans.trace(action_type="...")` | 1 | Python |
| **Anthropic Messages API** | `ANSAnthropicClient(base, agent_id=...)` | 1 | Python, TypeScript |
| **Claude Agent SDK** | `ClaudeAgentOptions(hooks=ans_hooks(...))` | 1 | Python |
| **OpenAI Agents SDK** | `ans_tool_plugin()` | 1 | Python, TypeScript |
| **OpenAI Compatible** | `ANSOpenAICompatClient(base, ...)` | 1 | Python |
| **Google Gemini** | `ANSGenAIClient(base, agent_id=...)` | 1 | Python, TypeScript |
| **Ollama** | `ANSOllamaClient(base, agent_id=...)` | 1 | Python, TypeScript |
| **LangChain** | `chain.invoke(..., callbacks=[ANSCallbackHandler(...)])` | 1 | Python, TypeScript |
| **LangGraph** | `ans_node(my_node, agent_id=...)` | 1 | Python, TypeScript |
| **CrewAI** | `ANSTool` (subclass) | 3 | Python |
| **PydanticAI** | `Agent("...", capabilities=[ans_hooks(...)])` | 1 | Python |
| **MCP** | Middleware | 1 | Python, TypeScript |
| **TypeScript/Node.js** | `wrap(tool, {agentId, client})` | 1 | TypeScript |

```bash
pip install sdks/python/     # Python SDK — from cloned repo
cd sdks/typescript && npm install  # TypeScript SDK
```

Every integration supports a `silent` parameter — daemon restarts mid-trace **never crash your agent**.

---

### 9. Ephemeral Identity Provisioning (Zero-Trust Broker)

**The Problem:** Agents execute commands under the developer's master API keys or admin terminal permissions. This violates every rule of enterprise access management and creates a **massive insider threat surface**.

**The Solution:** An autonomous, **zero-trust Identity Broker** that provisions ephemeral, scoped, single-use credentials with 60-second TTLs.

Instead of letting an agent read static environmental keys from a `.env` file, the ANS Identity Broker hooks directly into infrastructure access managers (like HashiCorp Vault or AWS IAM). When the agent requests a tool call to read from a production bucket, ANS provisions a **unique, ephemeral, single-use token with a lifetime of exactly 60 seconds**, scoped strictly to that specific file object.

#### Supported Providers

| Provider | Type | TTL | Revocation |
|----------|------|-----|------------|
| **HashiCorp Vault** | Dynamic secrets | 60s | Lease revoke API |
| **AWS IAM (STS)** | AssumeRole with inline policy | 60s | Auto-expire (non-renewable) |
| **GCP IAM** | Short-lived service account tokens | 60s | Token invalidation |
| **Azure AD** | Ephemeral app registrations | 60s | App deletion |
| **Generic OAuth2** | Client credentials flow | 60s | Token revoke endpoint |

#### Security Properties

| Property | Implementation |
|----------|---------------|
| **Max TTL** | 60 seconds (hard-coded server-side; client requests >60s are clamped) |
| **Scope enforcement** | Resource + action validated by provider before provisioning |
| **Single-use** | Default — each credential intended for one operation |
| **Auto-revocation** | Daemon schedules revocation at expiry time |
| **Audit trail** | Every provision/revoke flows through the daemon |
| **Secret redaction** | Credentials `[REDACTED]` in JSON marshaling and logs |
| **No static keys** | Agents receive time-limited tokens, never long-lived credentials |

---

### 10. Policy-as-Code — Biological Immune System

ANS policies act as a **biological immune system** for your agents — they evaluate every action at runtime and can deny, warn, or audit tool calls before they execute.

Policies are JSON-declarative with compound conditions (all/any/none) and 10 operators:

| Operator | Description |
|----------|-------------|
| `eq`, `neq` | String equality / inequality |
| `contains` | Substring match |
| `matches` | Regex pattern match (validated at registration — prevents ReDoS) |
| `gt`, `lt`, `gte`, `lte` | Numeric comparison |
| `in`, `not_in` | Set membership (list or comma-separated string) |

#### Denial Flow

When a policy denies an action, the daemon returns a **NociceptionError** (0x1F) — a "pain signal" that propagates to the agent as an error:

```
Action ──► Policy Executor ──► Enabled Policies (sorted by priority)
              │
              ├─ All allow? ──► Receipt signed and appended
              │
              └─ Any deny?  ──► NociceptionError returned
                                (action NOT recorded — no receipt created)
```

The first denial is **terminal** — no further policies are evaluated.

```go
executor := policy.NewExecutor(store)
result := executor.Evaluate(facts)
if !result.Allowed {
    return fmt.Errorf("NociceptionError (0x1F): %s", result.Reason)
}
```

#### Available Facts

| Fact | Source | Description |
|------|--------|-------------|
| `agent_id` | Receipt | Agent identifier |
| `action_type` | Receipt | Action type (file.read, http.post, etc.) |
| `phase` | Receipt | "pre" or "post" |
| `payload_summary` | Receipt | Short description of the action payload |
| `parent_agent_id` | Receipt | Parent agent when delegating |
| `context.contains_pii` | PII detector | Any PII detected in the action |
| `context.has_email` | PII Detector | Email address found |
| `context.has_ssn` | PII Detector | Social Security Number found |
| `context.has_credit_card` | PII Detector | Credit card number found |
| `context.has_api_key` | PII Detector | API key pattern found |
| `model.weight_type` | Policy context | "open" (default) — tunable per deployment |

ANS includes a built-in PII detector that scans action payloads for emails, SSNs, credit card numbers, phone numbers, IP addresses, and API keys (`sk-*`, `pk-*`).

---

### 11. Compensation — Undo for Agents

Every action can have a **compensating action** registered — a reversal command that undoes the action's effects. Compensations execute in **reverse order** (newest first). Each command is validated at registration against `safeCmdPattern` to prevent shell injection — executed via `strings.Fields` + direct `exec.Command` (no shell interpreter).

---

### 12. MCP Security Proxy

The MCP Security Proxy is a **transparent TCP-level proxy** that sits between MCP clients and servers, providing real-time traffic auditing, prompt injection detection, context window optimization, PII redaction, rate limiting, agent-scoped token budgets, per-method policy enforcement, and tool-use approval workflows — all backed by the same policy engine used for receipt access control.

```
┌─────────────┐          ┌──────────────────────────┐          ┌─────────────┐
│  MCP Client  │──JSON─►│     ANS MCP Proxy         │──JSON─►│  MCP Server  │
│  (Claude,    │        │                           │        │  (tools,     │
│   Cursor,    │◄───────│  Request Pipeline:        │◄───────│   resources) │
│   etc.)      │        │                           │        │              │
└─────────────┘          │  1. Rate limit check     │          └─────────────┘
                         │  2. Token budget check   │
                         │  3. Policy allow/deny    │
                         │  4. Tool approval        │
                         │  5. Injection detection  │
                         │  6. Forward to server    │
                         │                           │
                         │  Response Pipeline:       │
                         │  1. Receive from server   │
                         │  2. PII redaction         │
                         │  3. Context optimization  │
                         │  4. Audit log             │
                         │  5. Forward to client     │
                         └──────────────────────────┘
```

#### Safety Feature Pipeline

| Layer | Direction | What It Does | When It Blocks |
|-------|-----------|-------------|----------------|
| **Rate Limiter** | Client→Server | Token-bucket per client IP (default 60 req/min) | Returns JSON-RPC error, message dropped |
| **Token Budget** | Client→Server | Enforces per-agent estimated-token window (default 50K/min) | Returns JSON-RPC error, message dropped |
| **Policy Check** | Client→Server | Evaluates method against `policy.Executor` with fact `mcp.<method>` | Returns JSON-RPC error, message dropped |
| **Tool Approval** | Client→Server | Intercepts `tools/call`, evaluates tool name against policy with fact `mcp.tool_name` | Returns JSON-RPC error, message dropped |
| **Injection Detection** | Both | 6 regex patterns | Logged — messages always forwarded |
| **PII Redaction** | Server→Client | Replaces emails, SSNs, CCs, phones, IPs, API keys with `[REDACTED_*]` | Modifies response body, never blocks |
| **Context Optimizer** | Server→Client | Prunes repeated/base64/whitespace for >500 token responses | Modifies response body, never blocks |
| **Audit Log** | Both | Writes entry to `mcp_log` table | Never blocks |

```go
proxy := mcp.NewProxy(":8080", "http://localhost:9090")
proxy.Start()
// ... traffic flows through safety layers ...
proxy.Stop()
```

---

### 13. MCP Integration (SDK)

ANS works as **MCP middleware** (separate from the MCP Security Proxy above), wrapping any MCP tool with automatic pre/post tracing and optional state snapshots:

```python
# Python MCP middleware
from ans.mcp import ANSMiddleware
from mcp import Server

server = Server("my-server")
ans_mw = ANSMiddleware(server, agent_id="ans_3vQb7uL6x9")
ans_mw.wrap_all_tools()  # Every tool call is now traced
```

```typescript
// TypeScript MCP middleware
import { ANSMiddleware } from "ans-sdk/mcp";
import { Server } from "@modelcontextprotocol/sdk";

const server = new Server({ name: "my-server" });
const ans = new ANSMiddleware(server, { agentId: "ans_3vQb7uL6x9" });
ans.wrapAllTools();
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      YOUR APPLICATION                           │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Python SDK  │  │  TypeScript  │  │  MCP Client  │   ...    │
│  │  (zero deps) │  │  SDK         │  │              │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
└─────────┼─────────────────┼─────────────────┼───────────────────┘
          │                 │                 │
          └─────────────────┼─────────────────┘
                            │
            Unix socket 0600 / Windows named pipe
                            │
┌───────────────────────────┼─────────────────────────────────────┐
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    GO DAEMON                             │   │
│  │                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │  Chain Store  │  │  Keystore    │  │  Snapshot    │  │   │
│  │  │  (SQLite)     │  │  (Ed25519,   │  │  Store       │  │   │
│  │  │               │  │   AES-256-   │  │  (.tar.gz)   │  │   │
│  │  │  Receipts     │  │   GCM)       │  │  Differential│  │   │
│  │  │  Compensations│  │              │  │  SHA-256     │  │   │
│  │  │  Merkle Roots │  │  Agent IDs   │  │  Path guard  │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  │                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │  Policy      │  │  Identity    │  │  MCP Proxy   │  │   │
│  │  │  Executor    │  │  Broker      │  │  (injection  │  │   │
│  │  │  (Immune     │  │  (ephemeral  │  │   detection, │  │   │
│  │  │   System)    │  │   creds)     │  │   optimizer, │  │   │
│  │  │  Nociception │  │  Providers:  │  │   audit log) │  │   │
│  │  │  Error 0x1F  │  │  Dev, Env    │  │              │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  │                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │  Ed25519     │  │  Protocol    │  │  Webhook     │  │   │
│  │  │  Signer      │  │  Handler     │  │  Emitter     │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ~/.ans/                                                        │
│  ├── chain.db          SQLite database (receipts, snapshots,   │
│  │                     policies, MCP log, broker tokens)       │
│  ├── keys/             Encrypted Ed25519 key files             │
│  ├── snapshots/        Compressed tar.gz archives (full + diff)│
│  ├── machine.secret    32-byte HKDF seed for key encryption    │
│  └── daemon.pid        PID file for stop/status                │
└─────────────────────────────────────────────────────────────────┘
```

**Why this architecture?**
- **No network** — Unix socket/named pipe only (mode 0600 / ACL-secured). Zero attack surface.
- **No cloud** — Works fully offline. Your chain is yours.
- **No private key exposure** — SDKs send unsigned payloads; signing happens inside the daemon
- **Multiple agents, one chain** — All agents share the same append-only log. Cross-agent verification works out of the box.
- **Daemon survives agent restarts** — Stop and restart your agent 100 times; the chain persists.

---

## Security Model

| Property | Mechanism |
|----------|-----------|
| **Receipt authenticity** | **Ed25519** signature — verified against the agent's registered public key |
| **Chain integrity** | **SHA-256** hash linking — every receipt contains the hash of the previous one |
| **Pre-action intent** | Pre-receipts are signed **before** the tool executes — crash-safe audit trail |
| **Key storage** | **AES-256-GCM** encryption, key derived from machine-local secret via **HKDF-SHA256** |
| **Key isolation** | Private keys never leave the daemon — SDKs sign nothing locally |
| **Socket security** | Unix socket `mode 0600` / Windows named pipe ACL — local processes only |
| **Input validation** | Length limits on every protocol field; type validation on phase, outcome, policy, duration |
| **Injection prevention** | Agent IDs restricted to `^ans_[a-zA-Z0-9_-]{3,}$`; no raw SQL — all parameterized queries |
| **Snapshot authenticity** | Full-archive **SHA-256** hash stored in chain DB; `SnapshotID` is **Ed25519-signed** |
| **Snapshot path safety** | Traversal guard on capture AND restore — paths outside workspace root are rejected |
| **Snapshot exclusions** | `.ans/` and `node_modules/` automatically excluded; extensions can be whitelisted |
| **Differential snapshots** | Only changed files stored since base snapshot; deletions tracked via `.ans_deleted` manifest |
| **Hash correctness** | Archive hash computed on **the entire `.tar.gz` file** — not just file bodies — so headers and metadata are covered |
| **FD safety** | File handles closed explicitly in walk callbacks — no `defer`-in-loop FD leaks |
| **Policy injection** | Action denied → `NociceptionError` (0x1F) — action NOT recorded in chain; first denial is terminal |
| **ReDoS prevention** | All policy regex patterns validated and compiled at registration time; invalid regex rejected |
| **Shell injection prevention** | Compensation commands validated against `safeCmdPattern`; executed via `strings.Fields` + direct exec (no shell) |
| **Connection hardening** | TCP keep-alive enabled; read deadline always armed (never disabled); 30s idle timeout |
| **MCP injection detection** | 6 prompt injection patterns scanned on every message; logged to `mcp_log` table |
| **MCP PII redaction** | Server responses scanned for 6 PII types (email, SSN, CC, phone, IP, API key); replaced with `[REDACTED_*]` before forwarding to client |
| **MCP rate limiting** | Token-bucket per client IP (default 60 req/min); returns JSON-RPC error on overflow |
| **MCP token budgets** | Sliding-window per-agent estimated-token cap (default 50K/min); prevents context exhaustion |
| **MCP policy enforcement** | Every method evaluated against `policy.Executor` with fact `mcp.<method>`; deny policies block requests before forwarding |
| **MCP tool approval** | `tools/call` requests intercepted; tool name checked against policy with fact `mcp.tool_name`; denied calls return JSON-RPC error |
| **Identity broker TTL** | Ephemeral credentials hard-capped at 60s server-side; single-use by default |
| **Secret redaction** | Broker credentials `[REDACTED]` in JSON marshaling and all log output |
| **Export safety** | Output paths validated with `filepath.Clean`; directory traversal blocked |
| **Error handling** | All protocol errors logged; silent mode (`silent=True`) prevents agent crashes on daemon restarts |
| **Thread safety** | `sync.RWMutex` guarding the snapshot store; chain inserts are mutex-guarded |
| **Memory safety** | Frame size limited to 4 MB pre-allocation; no unbounded reads |

---

## Comparison

| Feature | ANS | Plain Logging | Commercial Audit Tools |
|---------|:---:|:---:|:---:|
| **Cryptographic signing** | ✓ Ed25519 | ✗ | Usually |
| **Hash-linked chain** | ✓ SHA-256 | ✗ | Sometimes |
| **Pre/post action receipts** | ✓ Both | ✗ Pre only | Rarely |
| **Workspace snapshots** | ✓ tar.gz + SHA-256 | ✗ | ✗ |
| **Time-travel restore** | ✓ Programmatic | ✗ | ✗ |
| **Multi-agent merge** | ✓ Causal ordering | ✗ | Rarely |
| **Merkle prune** | ✓ Infinite scale | N/A | Usually |
| **Offline-first** | ✓ Fully offline | ✓ | ✗ (SaaS) |
| **Open source** | ✓ Apache 2.0 | Usually | ✗ |
| **Framework integrations** | 13 | None | Vendor-specific |
| **MCP Security Proxy** | ✓ Full safety pipeline | ✗ | ✗ |
| **Policy-as-Code** | ✓ Declarative JSON + NociceptionError | ✗ | Vendor-only |
| **Ephemeral identity** | ✓ Broker with 60s TTL | ✗ | Usually |
| **Compensation (undo)** | ✓ Registered reverse actions | ✗ | ✗ |
| **Differential snapshots** | ✓ Changed-files only | ✗ | ✗ |
| **PII detection** | ✓ Built-in regex detector | ✗ | Usually |
| **Export formats** | 5 (JSONL, CSV, TXT, PDF, Parquet) | Depends | Usually 1-2 |
| **Real-time streaming** | ✓ NDJSON + CloudEvents webhooks | Sometimes | Usually |
| **Key rotation** | ✓ Ed25519 with rotation record | N/A | Usually |

---

## Why NOT a SaaS?

Audit trails should be as fundamental as `git`. You don't pay a vendor per commit.

- **Your chain. Your keys. Your library.** Open source, Apache 2.0 licensed.
- **Zero vendors** between you and the truth.
- **Works fully offline** — no internet connection required.
- **No API keys, no monthly bill, no data leaving your machine.**
- **Embed the daemon directly in your Go application** — no separate binary needed.

---

## The Bottom Line

> You are responsible for what your agents do.

ANS gives you the receipts — cryptographically signed, hash-linked, timestamped, snapshot-backed, verifiable, exportable, and time-travel-rewindable. Embed it in your Go application or use the SDK from any language.

```go
import "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"

d, _ := daemon.New(store, &daemon.Config{})
d.Run() // blocks — agents are now traced
```

---

## License

Apache 2.0 — ship it.

**Questions?** Open an issue at [github.com/Linky-Link-Linky/Agent-Nervous-System/issues](https://github.com/Linky-Link-Linky/Agent-Nervous-System/issues)
