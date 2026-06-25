<div align="center">

# ANS — Agent Nervous System

### `git log` for your AI Agents

**> Your AI agents are already running. Do you know what they actually did?**

**Cryptographic audit trails · State rollback · Policy-as-Code · Zero-trust identity · MCP security**

[![Go 1.22+](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Python SDK](https://img.shields.io/badge/python-3.10%2B-3776AB?logo=python)](sdks/python/)
[![TypeScript SDK](https://img.shields.io/badge/typescript-5.0%2B-3178C6?logo=typescript)](sdks/typescript/)
[![Go Reference](https://img.shields.io/badge/go-reference-00ADD8?logo=go)](https://pkg.go.dev/github.com/Linky-Link-Linky/Agent-Nervous-System)
[![Built for](https://img.shields.io/badge/built%20for-AI%20Agents-FF6F00)](#)

```bash
# One daemon. Zero config. Full cryptographic accountability.
ans start                      # Start the daemon
ans chain                      # See every action, ever
ans verify --chain             # Prove nothing was tampered with
ans time-travel 42             # Rewind workspace to chain index 42
ans compensate 42 --dry-run    # Preview undo actions
ans policy add my-policy.json  # Register a deny policy
ans mcp start --listen :8080   # Secure MCP traffic
```

**Works fully offline. One static binary. No SaaS, no API keys, no monthly bill.**

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

| Capability | What It Gives You |
|-----------|------------------|
| **Receipt Chain** | Pre/post Ed25519-signed receipts, hash-linked, SQLite-backed. Tamper-evident audit trail. |
| **Time-Travel** | Full workspace snapshots before every action. Restore any point in time with one command. |
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

## Quick Start — 30 Seconds

One command per platform. Choose yours:

### Linux / macOS (bash)

```bash
curl -fsSL https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex
```

### Or build from source (requires Go 1.22+)

```bash
git clone https://github.com/Linky-Link-Linky/Agent-Nervous-System.git && cd Agent-Nervous-System
make build && sudo make install
```

### First-Time Setup

```bash
ans init                      # create ~/.ans/ and default config
ans init --service            # optional: install as system service (systemd/launchd)
```

### Verify & Run

```bash
ans version                   # confirm installation
ans start                     # start the daemon (background)
ans register                     # register an agent (auto-generated name & version)
ans chain                     # see the receipt chain
ans verify --chain            # prove integrity: hashes + Ed25519 signatures ✓
```

### Persistent Settings

Settings persist across restarts via `~/.ans/config.json`. Set defaults with:

```bash
ans init --webhook https://hooks.example.com/ans --ndjson
```

After this, `ans start` will automatically use the webhook and NDJSON output without needing flags each time. CLI flags still override config values.

### (Optional) Use from Python

```bash
pip install ans/sdks/python/
```

```python
from ans import ANSClient
client = ANSClient()
with client.trace("hello_world"):
    print("Hello from my traced agent!")
```

Run it, then `ans chain` again — a new receipt appears.

### What next?

| I want to... | Run this |
|-------------|----------|
| See every action in detail | `ans chain --n 50` |
| Verify the chain hasn't been tampered with | `ans verify --chain` |
| Rewind workspace to a point in time | `ans time-travel 42` |
| Preview undo for an action | `ans compensate 42 --dry-run` |
| Register a deny policy | `ans policy add examples/policies/no-pii-open-weights.json` |
| Export a compliance PDF | `ans export --format pdf --output audit.pdf` |
| Compact old receipts (infinite scale) | `ans prune --up-to 10000` |
| Secure an MCP server | `ans mcp start --listen :8080 --target http://localhost:9090` |
| Get help | `ans help` |

### Troubleshooting

| Problem | Fix |
|---------|-----|
| `go: command not found` | Install Go from [go.dev/dl](https://go.dev/dl/) |
| `make: command not found` | Windows: [GnuWin32 Make](https://gnuwin32.sourceforge.net/packages/make.htm). Mac: `xcode-select --install`. Linux: `sudo apt install make` |
| `ans: command not found` | Run `sudo make install` (Linux/Mac) or the Windows copy command above |
| `ans: no config` | Run `ans init` first to create the data directory |
| Daemon won't start | Run `ans doctor` to check the socket, PID, and config. Then `ans stop` and `ans start` again |
| `ans chain` shows nothing | Register an agent first (`ans register`) |
| Permission denied | Prefix with `sudo` (Linux/Mac) or run terminal as Administrator (Windows) |
| Something else | Run `ans doctor` and include the output when [opening a GitHub issue](https://github.com/Linky-Link-Linky/Agent-Nervous-System/issues) |

---

## From Zero to Your First Traced Agent

A complete walkthrough — install ANS, integrate it into your agent code, and view the cryptographic audit trail.

### 1. Install ANS

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex

# Or build from source (Go 1.22+)
git clone https://github.com/Linky-Link-Linky/Agent-Nervous-System.git
cd Agent-Nervous-System
make build && sudo make install
```

### 2. Start the Daemon

```bash
ans init            # create ~/.ans/ config and data directory
ans start           # launch daemon in background
```

That's it. The daemon listens on a local Unix socket (or named pipe on Windows). No cloud, no accounts, no API keys.

### 3. Register an Agent Identity

Every traced action needs an agent identity — an Ed25519 keypair stored encrypted on disk:

```bash
ans register
```

Output:
```
● Agent registered
  Agent ID:  ans_3vQb7uL6x9
  Name:      my-bot
  Version:   1.0.0
```

Save the `Agent ID` — you'll use it when instrumenting your code.

### 4. Install the SDK

```bash
# Python
pip install sdks/python/       # from the cloned repo

# TypeScript
cd sdks/typescript && npm install
```

### 5. Instrument Your Agent

Choose the integration pattern that fits your codebase.

#### Pattern A: Decorator (simplest)

Wrap any function — ANS records a pre/post receipt pair automatically:

```python
from ans import ANSClient

client = ANSClient()

@client.trace("file.write", agent_id="ans_3vQb7uL6x9")
def save_config(data: dict) -> None:
    with open("/tmp/config.json", "w") as f:
        json.dump(data, f)

save_config({"key": "value"})  # automatically traced
```

#### Pattern B: Context Manager (inline control)

Trace a specific block without decorating:

```python
with client.trace("db.query", agent_id="ans_3vQb7uL6x9"):
    rows = db.execute("SELECT * FROM users")
```

#### Pattern C: LangChain / LangGraph

```python
from langchain_community.callbacks import ANSClient
from langchain.agents import AgentExecutor

client = ANSClient(agent_id="ans_3vQb7uL6x9")

agent = AgentExecutor(
    ...,
    callbacks=[client.as_langchain_callback()]
)
```

#### Pattern D: OpenAI / Anthropic

```python
from ans.integrations import ANSAnthropicClient

client = ANSAnthropicClient(
    base=anthropic.Anthropic(),
    agent_id="ans_3vQb7uL6x9",
)
response = client.messages.create(...)
```

### 6. View the Chain

Every traced action produced a signed, hash-linked receipt:

```bash
ans chain
```

```
  ╭─ a1b2c3d4  2026-06-18 14:30:22.000  file.write        ans_3vQb7uL6x9
  │  writing config file
  │  policy: allow
  ╰─ ✓ success  1200ms
     sig: a1b2c3d4e5f6a7b8…

  ╭─ b2c3d4e5  2026-06-18 14:31:05.000  db.query          ans_3vQb7uL6x9
  │  SELECT * FROM users
  │  policy: allow
  ╰─ ✓ success  300ms
     sig: b2c3d4e5f6a7b8c9…
```

### 7. Verify Integrity

Prove no one tampered with the chain:

```bash
ans verify --chain
✓ Chain integrity verified — 6 receipts checked (all hashes, all signatures)
```

### What's Next

| Integration | File |
|-------------|------|
| Python SDK + examples | `sdks/python/` |
| TypeScript SDK | `sdks/typescript/` |
| Policy examples | `examples/policies/` |
| Compensating action examples | `examples/compensations/` |
| Full integration demos | `examples/integrations/` |

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

```bash
$ ans chain

    ANS — Receipt Chain
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  ╭─ a1b2c3d4  2026-06-18 14:30:22.000  file.write        ans_3vQb7uL6x9
  │  writing config file
  │  policy: allow
  ╰─ ✓ success  1200ms
     sig: a1b2c3d4e5f6a7b8…

  ╭─ b2c3d4e5  2026-06-18 14:31:05.000  agent.delegate    ans_9yWc2kM4x1
  │  delegating to sub-agent
  │  policy: allow
  ╰─ ✗ failure  8700ms
     sig: b2c3d4e5f6a7b8c9…

  ╭─ c3d4e5f6  2026-06-18 14:32:15.000  http.post         ans_3vQb7uL6x9
  │  posting results to webhook
  │  policy: allow
  ╰─ ◐ partial  400ms
     sig: c3d4e5f6a7b8c9d0…

$ ans verify --chain
✓ Chain integrity verified — 6 receipts checked (all hashes, all signatures)
```

**Key security properties:**
- Every receipt is **Ed25519-signed** by the agent's private key before it leaves the daemon
- Can't forge, delete, or reorder receipts without breaking the hash chain
- Pre-receipts are signed **before** the tool runs — so even if the agent crashes mid-execution, you have proof of intent
- `ans verify --chain` walks every receipt: recomputes hashes, checks chain pointers, verifies signatures against registered public keys. **One broken link = FAIL**

---

### 2. Time-Travel — Full State Rewind

Every pre-action receipt captures a **full workspace snapshot** as a compressed, hashed, and cryptographically bound archive. You can restore the workspace to any point in time with a single command.

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
  │ You run  │        │  Read archive from   │        │  Replay      │
  │ ans time-│───────►│  disk, verify hash,  │───────►│  Receipts to │
  │ travel   │        │  extract to workspace│        │  restore     │
  └──────────┘        └──────────────────────┘        └──────────────┘
```

```bash
# List all snapshots for an agent
$ ans snapshots --agent ans_3vQb7uL6x9

SNAPSHOT ID           TYPE        INDEX      SIZE       TIMESTAMP
─────────────────────────────────────────────────────────────────
a1b2c3d4e5f6a7b8     filesystem  42         128.4 KB   14:30:22
b2c3d4e5f6a7b8c9     filesystem  41         127.1 KB   14:29:15
c3d4e5f6a7b8c9d0     filesystem  40         125.3 KB   14:28:10

# Rewind workspace to exactly how it was at chain index 42
$ ans time-travel 42
✓ State restored to chain index 42
```

**How it works:**
- Before every action, the daemon creates a gzipped tar archive of the workspace
- The archive is **SHA-256 hashed** (whole file, not just content — tamper-proof)
- The hash, path, size, and timestamp are stored in the chain DB alongside the receipt
- The `SnapshotID` is embedded in the receipt's `signableFields` — it's **Ed25519-signed** together with the action data
- On restore, the archive is extracted with **full path traversal protection** (no escape exploits)
- `.ans/` and `node_modules/` are automatically excluded from snapshots
- Snapshots survive pruning — you can compact the receipt chain and still restore to any checkpoint

**The result:** You can prove *exactly* what file state existed before a given action, and you can re-create that state perfectly on demand. This is your agent's undo button.

---

### 3. Multi-Agent Merge

When multiple agents (or sub-agents) produce interleaved receipts, `MergeChains` reconstructs a single causal timeline:

1. **Sub-agent receipts** follow the delegation receipt that spawned them
2. **Post-receipts** follow their pre-action pair
3. Remaining receipts are ordered by timestamp

```bash
$ ans chain --agent ans_parent  # Raw interleaved view
$ ans chain                     # Merged causal timeline
```

No more reconstructing "what happened" from interleaved log files. The chain captures causality.

---

### 4. Identity & Key Management

| Property | Implementation |
|----------|---------------|
| **Identity** | `ans_` + base58 of SHA-256(Ed25519 public key)[:10] (e.g. `ans_3vQb7uL6x9`) |
| **Signing** | Ed25519 — all receipts are signed by the agent's private key |
| **Key storage** | Encrypted on disk with **AES-256-GCM**, key derived from machine-local secret via HKDF-SHA256 |
| **Rotation** | `ans rotate <agent-id>` generates a new keypair, records the rotation as a signed receipt |
| **Isolation** | SDKs never touch private keys — signing happens in the daemon over a local socket |

```bash
# Register a new agent (generates keypair)
$ ans register --name my-agent --version 1.0.0 --owner acme-corp  # all flags optional
ans_3vQb7uL6x9

# Verify who signed a receipt
$ ans verify a1b2c3d4e5f6a7b8
Receipt a1b2c3d4e5f6a7b8
  Agent:     ans_3vQb7uL6x9 (my-agent v1.0.0)
  Action:    file.write
  Outcome:   success
  Valid:     ✓ (Ed25519 signature verified)
```

---

### 5. Pruning — Infinite Scale

The chain grows forever. When it gets large, compact old receipts into a **Merkle anchor**:

```bash
$ ans prune --up-to 10000
Pruned 10000 receipts (index 1–10000)
Merkle root: 7c9d8a3f2b1e4f6a8d0c2b4e6f8a0d1c2b4e6f8a
```

After pruning, `ans verify --chain` still works — it accepts the Merkle anchor as proof. Receipt data is removed, but the cryptographic integrity is preserved. **Snapshots at pruned indices are NOT deleted** — you can still `ans time-travel 500` to restore state at any pruned index.

---

### 6. Export & Compliance

```bash
# Human-readable audit report
ans export --format pdf       # Full PDF report (with table of contents)
ans export --format txt       # Plain text summary

# Analytics-ready
ans export --format jsonl     # JSONL — import into any data pipeline
ans export --format csv       # CSV — open in Excel, Google Sheets
ans export --format parquet   # Parquet — import into Spark, DuckDB, BigQuery
```

---

### 7. Real-Time Streaming

```bash
# Stream every new receipt as NDJSON to stdout (capture with > file)
ans start --ndjson > receipt-stream.jsonl

# POST CloudEvents-formatted payloads to any webhook
ans start --webhook https://hooks.example.com/ans
```

```json
// Example NDJSON line
{"type":"receipt","data":{"receipt_id":"a1b2...","agent_id":"ans_3vQb...", ...}}

// Example webhook payload (CloudEvents 1.0)
{
  "specversion": "1.0",
  "id": "a1b2c3d4e5f6a7b8",
  "source": "ans/daemon",
  "type": "ans.receipt.append",
  "datacontenttype": "application/json",
  "time": "2026-06-18T14:30:22Z",
  "data": { "receipt_id": "a1b2...", "agent_id": "ans_3vQb...", ... }
}
```

---

### 8. SDK Integrations (13 Frameworks)

| Framework | Integration | Loc | SDK |
|-----------|------------|:---:|:---:|
| **Python native** | `@ans.trace(action_type="...")` | 1 | Python |
| **Anthropic Messages API** | `ANSAnthropicClient(base, agent_id=...)` | 1 | Python, TypeScript |
| **Claude Agent SDK** | `ClaudeAgentOptions(hooks=ans_hooks(...))` | 1 | Python |
| **OpenAI Agents SDK** | `ans_tool_plugin()` | 1 | Python, TypeScript |
| **OpenAI Compatible** | `ANSOpenAICompatClient(base, ...)` — works with any provider exposing an OpenAI-compatible `/chat/completions` endpoint: llama.cpp, vLLM, Together, Groq, DeepSeek, Mistral, Fireworks, Perplexity, xAI | 1 | Python |
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

#### How It Works

Instead of letting an agent read static environmental keys from a `.env` file, the ANS Identity Broker hooks directly into infrastructure access managers (like HashiCorp Vault or AWS IAM). When the agent requests a tool call to read from a production bucket, ANS provisions a **unique, ephemeral, single-use token with a lifetime of exactly 60 seconds**, scoped strictly to that specific file object.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    AGENT REQUESTS TOOL CALL                         │
│  "Read s3://prod-bucket/config.json"                               │
└──────────────────────┬──────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│              ANS IDENTITY BROKER (Zero-Trust Gate)                  │
│                                                                     │
│  1. Verify agent identity (Ed25519 signature)                      │
│  2. Check policy: Is this agent allowed to read this resource?     │
│  3. Generate ephemeral credential request:                         │
│     • Resource: s3://prod-bucket/config.json                       │
│     • Permissions: ["read"]                                        │
│     • TTL: 60 seconds                                              │
│     • Agent ID: ans_3vQb7uL6x9                                     │
│     • Pre-receipt ID: abc123 (audit trail link)                    │
└──────────────────────┬──────────────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   VAULT      │ │  AWS IAM/STS │ │  GCP IAM     │
│              │ │              │ │              │
│  Provision   │ │  AssumeRole  │ │  Token       │
│  dynamic     │ │  with inline │ │  exchange    │
│  secret      │ │  policy:     │ │  with scope  │
│              │ │              │ │              │
│  Path:       │ │  Resource:   │ │  Resource:   │
│  aws/creds/  │ │  arn:aws:s3  │ │  storage.    │
│  deploy      │ │  ::prod-     │ │  objects.get │
│              │ │  bucket/     │ │              │
│  TTL: 60s    │ │  config.json │ │  TTL: 60s    │
│              │ │              │ │              │
│  Returns:    │ │  Action:     │ │  Returns:    │
│  • Token     │ │  s3:GetObject│ │  • OAuth2    │
│  • Lease ID  │ │              │ │    token     │
└──────┬───────┘ │  TTL: 60s    │ └──────┬───────┘
       │         │              │        │
       │         │  Returns:    │        │
       │         │  • AccessKey │        │
       │         │  • SecretKey │        │
       │         │  • Token     │        │
       │         └──────┬───────┘        │
       │                │                │
       └────────────────┼────────────────┘
                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  CREDENTIAL RETURNED TO AGENT                       │
│                                                                     │
│  {                                                                  │
│    "credential_id": "cred_a1b2c3d4",                               │
│    "type": "aws-sts",                                              │
│    "secret": "ASIAXYZ...",                                         │
│    "metadata": {                                                   │
│      "secret_key": "[REDACTED]",                                   │
│      "session_token": "[REDACTED]",                                │
│      "region": "us-east-1"                                         │
│    },                                                              │
│    "expires_at": "2026-06-18T14:31:22Z",  ← 60 seconds from now   │
│    "scope": {                                                      │
│      "resource": "s3://prod-bucket/config.json",                   │
│      "permissions": ["read"]                                       │
│    }                                                               │
│  }                                                                  │
└──────────────────────┬──────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│              AGENT EXECUTES TOOL WITH EPHEMERAL CRED                │
│                                                                     │
│  import boto3                                                       │
│  s3 = boto3.client(                                                │
│      's3',                                                          │
│      aws_access_key_id=cred.secret,                                │
│      aws_secret_access_key=cred.metadata["secret_key"],            │
│      aws_session_token=cred.metadata["session_token"]              │
│  )                                                                  │
│  data = s3.get_object(Bucket='prod-bucket', Key='config.json')     │
└──────────────────────┬──────────────────────────────────────────────┘
                       │
                       ▼ (60 seconds later)
┌─────────────────────────────────────────────────────────────────────┐
│           CREDENTIAL EXPIRES & AUTO-REVOKES                         │
│  • AWS STS token expires (non-renewable)                           │
│  • Vault lease revoked via API                                     │
│  • GCP token invalidated                                           │
│  • Credential marked as revoked in ANS cache                       │
└─────────────────────────────────────────────────────────────────────┘
```

#### Python SDK Usage

```python
from ans import ANSClient
from ans.broker import IdentityBroker, Scope, ephemeral_credential

# Initialize broker
client = ANSClient()
broker = IdentityBroker(client)

# Option 1: Manual provision/revoke
cred = broker.provision(
    provider="aws-iam",
    agent_id="ans_3vQb7uL6x9",
    action_type="file.read",
    scope=Scope(
        resource="s3://prod-bucket/config.json",
        permissions=["read"]
    ),
    ttl_seconds=60
)

# Use the credential
import boto3
s3 = boto3.client(
    's3',
    aws_access_key_id=cred.secret,
    aws_secret_access_key=cred.metadata["secret_key"],
    aws_session_token=cred.metadata["session_token"]
)
data = s3.get_object(Bucket='prod-bucket', Key='config.json')

# Revoke immediately (or wait 60s for auto-expiry)
broker.revoke(cred.credential_id)


# Option 2: Context manager (auto-revokes)
with ephemeral_credential(
    broker,
    provider="vault",
    agent_id="ans_3vQb7uL6x9",
    action_type="db.query",
    scope=Scope("vault://database/creds/readonly", ["read"])
) as cred:
    # Credential is valid for 60 seconds
    import psycopg2
    conn = psycopg2.connect(
        host="db.example.com",
        user=cred.metadata["username"],
        password=cred.secret
    )
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users LIMIT 10")
# Credential is automatically revoked here
```

#### Supported Providers

| Provider | Type | TTL | Revocation |
|----------|------|-----|------------|
| **HashiCorp Vault** | Dynamic secrets | 60s | Lease revoke API |
| **AWS IAM (STS)** | AssumeRole with inline policy | 60s | Auto-expire (non-renewable) |
| **GCP IAM** | Short-lived service account tokens | 60s | Token invalidation |
| **Azure AD** | Ephemeral app registrations | 60s | App deletion |
| **Generic OAuth2** | Client credentials flow | 60s | Token revoke endpoint |

#### CLI Commands

```bash
# Provision a credential (returns credential ID and secret)
ans token request \
  --resource "s3://prod-bucket/config.json" \
  --action read \
  --ttl 60

# List active credentials
ans token list

# Revoke a credential immediately
ans token revoke <credential-id>
```

#### Security Properties

| Property | Implementation |
|----------|---------------|
| **Max TTL** | 60 seconds (hard-coded server-side; client requests >60s are clamped) |
| **Scope enforcement** | Resource + action validated by provider before provisioning |
| **Single-use** | Default — each credential intended for one operation |
| **Auto-revocation** | Daemon schedules revocation at expiry time |
| **Audit trail** | Every provision/revoke flows through the daemon |
| **Secret redaction** | Credentials `[REDACTED]` in JSON marshaling and logs |
| **Provider isolation** | `DevProvider` returns synthetic keys; `EnvProvider` reads `AWS_*` vars |
| **No static keys** | Agents receive time-limited tokens, never long-lived credentials |

#### Why This Matters

**Without ephemeral provisioning:**
- Agent reads `AWS_ACCESS_KEY_ID` from `.env` → has **permanent** access to **everything** that key allows
- Agent crashes → key is still valid for days/months
- Attacker compromises agent → exfiltrates permanent credentials
- Compliance audit → "Why does this agent have admin keys?"

**With ephemeral provisioning:**
- Agent gets a **60-second token** scoped to **one S3 object** with **read-only** access
- Agent crashes → token expires in < 60s, no cleanup needed
- Attacker compromises agent → token is already expired or revokes in seconds
- Compliance audit → "Every action has a unique, time-limited, scoped credential with full audit trail"

This is **zero-trust** for AI agents.

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

#### Anatomy of a Policy

```json
{
  "id": "no-pii-on-open-models",
  "name": "Block PII on Open-Weight Models",
  "enabled": true,
  "priority": 100,
  "severity": "high",
  "conditions": {
    "all": [
      {
        "any": [
          { "fact": "context.has_email", "operator": "eq", "value": true },
          { "fact": "context.has_ssn", "operator": "eq", "value": true },
          { "fact": "context.has_credit_card", "operator": "eq", "value": true }
        ]
      },
      {
        "fact": "model.weight_type", "operator": "eq", "value": "open"
      }
    ]
  },
  "action": {
    "effect": "deny",
    "error_type": "NociceptionError",
    "error_message": "Cannot send PII to open-weight models"
  }
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

#### PII Detection

ANS includes a built-in PII detector that scans action payloads for:
- Emails, SSNs, credit card numbers, phone numbers, IP addresses, API keys (`sk-*`, `pk-*`)

Each PII type is available as an individual fact and is also aggregated into `context.contains_pii`.

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

```bash
# Register a policy from JSON file
ans policy add examples/policies/no-pii-open-weights.json

# List all policies
ans policy list

# Test a policy without running an action
ans policy eval --action-type "http.post" --payload-summary "email: user@example.com"
# ✗ DENIED — Cannot send PII to open-weight models

# Remove a policy
ans policy remove no-pii-on-open-models
```

---

### 11. Compensation — Undo for Agents

Every action can have a **compensating action** registered — a reversal command that undoes the action's effects. This is how you implement rollback at the application level.

```bash
# After a destructive file deletion
ans chain | grep "file.delete"

# Preview what compensation would execute
ans compensate 47 --dry-run
# [47] file.delete: restore from backup (cmd: restore-backup.sh)

# Execute the compensation
ans compensate 47
# ✓ Compensation complete: 1 run, 0 failed
```

Compensations execute in **reverse order** (newest first). Each command is validated at registration against `safeCmdPattern` to prevent shell injection — executed via `strings.Fields` + direct `exec.Command` (no shell interpreter).

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

Every message passes through the proxy's safety layers in order:

| Layer | Direction | What It Does | When It Blocks |
|-------|-----------|-------------|----------------|
| **Rate Limiter** | Client→Server | Token-bucket per client IP (default 60 req/min) | Returns JSON-RPC error, message dropped |
| **Token Budget** | Client→Server | Enforces per-agent estimated-token window (default 50K/min) | Returns JSON-RPC error, message dropped |
| **Policy Check** | Client→Server | Evaluates method against `policy.Executor` with fact `mcp.<method>` | Returns JSON-RPC error, message dropped |
| **Tool Approval** | Client→Server | Intercepts `tools/call`, evaluates tool name against policy with fact `mcp.tool_name` | Returns JSON-RPC error, message dropped |
| **Injection Detection** | Both | 6 regex patterns (unchanged) | Logged — messages always forwarded |
| **PII Redaction** | Server→Client | Replaces emails, SSNs, CCs, phones, IPs, API keys with `[REDACTED_*]` | Modifies response body, never blocks |
| **Context Optimizer** | Server→Client | Prunes repeated/base64/whitespace for >500 token responses (unchanged) | Modifies response body, never blocks |
| **Audit Log** | Both | Writes entry to `mcp_log` table (unchanged) | Never blocks |

#### PII Redaction

Server responses are automatically scanned and redacted for 6 PII types before reaching the client:

| PII Type | Example | Replaced With |
|----------|---------|---------------|
| Email | `user@example.com` | `[REDACTED_EMAIL]` |
| SSN | `123-45-6789` | `[REDACTED_SSN]` |
| Credit Card | `4111 1111 1111 1111` | `[REDACTED_CC]` |
| Phone | `+14155551234` | `[REDACTED_PHONE]` |
| IP Address | `192.168.1.1` | `[REDACTED_IP]` |
| API Key | `sk-proj-abc123...` | `[REDACTED_API_KEY]` |

Redaction is applied to the JSON result body before the optimizer runs, so PII is never written to the audit log or forwarded to the client.

#### Rate Limiting & Token Budgets

| Feature | Default | Purpose |
|---------|---------|---------|
| **Rate limit** | 60 requests/minute per client IP | Prevents runaway agents from flooding the MCP server |
| **Token budget** | 50,000 estimated tokens/minute per client IP | Caps total context consumption per agent window |

When exceeded, the proxy returns a JSON-RPC error (`-32000` or `-32001`) to the client and drops the message — the request never reaches the MCP server.

#### Method-Level Policy Enforcement

The proxy evaluates every client request method against the daemon's `policy.Executor` using a fact path of `mcp.<method>`. Register a policy to deny specific MCP methods:

```json
{
  "id": "deny-tools-call",
  "name": "Block all tool calls",
  "enabled": true,
  "priority": 100,
  "severity": "high",
  "conditions": {
    "fact": "mcp.method", "operator": "eq", "value": "mcp.tools.call"
  },
  "action": {
    "effect": "deny",
    "error_type": "NociceptionError",
    "error_message": "Tool calls are not allowed through the proxy"
  }
}
```

Available facts for MCP policy evaluation:

| Fact | Description |
|------|-------------|
| `mcp.method` | The MCP method being called (`mcp.tools.call`, `mcp.resources.read`, etc.) |
| `mcp.tool_name` | The specific tool name (only for `tools/call` requests) |
| `mcp.client_addr` | Client IP address |
| `action_type` | Always `mcp.<method>` for compatibility with existing policies |

#### Tool-Use Approval Workflow

When a `tools/call` request arrives, the proxy extracts the tool name from the request params and checks it against the policy engine. Denied tool calls return a JSON-RPC error (`-32003`) to the client — the tool request never reaches the MCP server.

```bash
# Policy to block a specific tool
$ ans policy add - <<'EOF'
{
  "id": "block-dangerous-tool",
  "name": "Block dangerous-tool",
  "conditions": {
    "all": [
      { "fact": "mcp.method", "operator": "eq", "value": "mcp.tools.call" },
      { "fact": "mcp.tool_name", "operator": "eq", "value": "dangerous-tool" }
    ]
  },
  "action": { "effect": "deny", "error_message": "Tool blocked by policy" }
}
EOF
```

#### Usage

```bash
# Start the proxy with all safety features enabled by default
ans mcp start --listen :8080 --target http://localhost:9090

# View proxy status with safety stats
ans mcp status
# MCP proxy: running
#   Uptime:      342s
#   Messages:    1523
#   Total Toks:  284501
#   Burn Rate:   832.4 toks/s
#   Injections:  3
#   Pruned:      47 msgs (128 KB)
#   Rate Limited: 2
#   Budget Exceeded: 0
#   Policy Denied: 1
#   Tools Denied: 0

# View recent audit log
ans mcp log -n 10

# View only injection detections
ans mcp log --inj

# Filter by method
ans mcp log --method resources/read

# Stop the proxy
ans mcp stop
```

Each proxied message is logged to the `mcp_log` table with direction, method, token estimate, injection status, pruning status, and content preview. Rate limits, token budgets, policy denials, and tool approval rejections are logged at `INFO` level.

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
│  │                    GO DAEMON (ans start)                 │   │
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

## CLI Reference

```
Setup & Maintenance
  ans init                    Create ~/.ans/ and default config
    --webhook <url>           Default webhook URL for ans start
    --ndjson                  Enable NDJSON output by default
    --service                 Install as a system service (systemd/launchd)
  ans doctor                  Show diagnostics (socket, PID, config, chain status)
  ans start                   Start the daemon (background, persistent)
    --ndjson                  Emit NDJSON receipt stream to stdout (overrides config)
    --webhook <url>           POST CloudEvents to URL per new receipt (overrides config)
  ans stop                    Stop the daemon
  ans status                  Uptime, chain length, agent count, DB size
  ans update                  Update ANS to the latest version
  ans uninstall               Remove ANS binary, data, and config

Chain & Receipts
  ans chain                   Print the receipt tree (newest-first)
    --n <int>                 Number of receipts (default 20)
    --agent <id>              Filter by agent ID
  ans verify <id>             Verify a single receipt (hash + signature)
    --chain                   Verify the entire chain integrity
  ans agents                  List registered agents and their Ed25519 public keys
  ans export                  Export the receipt chain
    --format jsonl|csv|txt|pdf|parquet   (default jsonl)
    --output <path>           Output file (default stdout)
  ans prune --up-to <index>   Compact old receipts into a Merkle anchor
  ans rotate <agent-id>       Generate a new Ed25519 keypair for an agent

Time-Travel & Snapshots
  ans time-travel <index>     Restore workspace to state at chain index (or 64-char receipt hash)
    --type filesystem         Snapshot type (default filesystem)
  ans snapshot take           Take a snapshot of agent workspace
    --agent <id>              Agent ID (required)
    --type filesystem         Snapshot type (default filesystem)
    --paths a,b               Comma-separated paths to snapshot (empty = full workspace)
  ans snapshot diff           Show file-level diff vs prior snapshot
    --agent <id>              Agent ID (required)
  ans snapshots               List snapshots for an agent
    --agent <id>              Agent ID (required)
    --type filesystem         Snapshot type (default filesystem)
    --n <int>                 Number to show (default 20)

Compensation (Undo Actions)
  ans compensate <index>      Execute compensating actions for a chain index
    --dry-run                 Preview what would be executed without running

Policy-as-Code (Immune System)
  ans policy add <file.json>  Register a policy from JSON file
  ans policy list             List all policies
    --enabled                 Show only enabled policies
  ans policy remove <id>      Remove a policy by ID
  ans policy eval             Evaluate an action against policies
    --action-type <type>      Action type (required)
    --payload-summary <text>  Payload summary text

Ephemeral Tokens (Identity Broker)
  ans token request           Provision ephemeral credential
    --resource <arn>          Resource ARN or path (required)
    --action read             Action (read, write, etc.)
    --ttl 60                  Token TTL in seconds (max 60)
  ans token list              List active tokens
  ans token revoke <id>       Revoke a token immediately

MCP Security Proxy
  ans mcp start               Start MCP security proxy
    --listen :8080            Listen address (default :8080)
    --target <url>            Target MCP server URL (required)
    --safety-disable          Disable all safety features (PII redaction, rate limiting, etc.)
    --rate-limit 60           Requests/minute per client IP (0 = unlimited)
    --token-budget 50000      Estimated tokens/minute per agent (0 = unlimited)
    --pii-redact true         Enable PII redaction on server responses
  ans mcp stop                Stop the MCP proxy
  ans mcp status              Show proxy status and stats
  ans mcp log                 Show recent MCP audit log
    --n <int>                 Number of entries (default 20)
    --method <m>              Filter by method
    --inj                     Show only injection detections

Other
  ans version                 Print version info
  ans help                    Show this help
```

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
| **MCP line safety** | `bufio.Reader` with 4MB buffer (not `bufio.Scanner` which caps at 1MB); `ReadBytes('\n')` handles large JSON-RPC messages |
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
| **Time-travel restore** | ✓ One command | ✗ | ✗ |
| **Multi-agent merge** | ✓ Causal ordering | ✗ | Rarely |
| **Merkle prune** | ✓ Infinite scale | N/A | Usually |
| **Offline-first** | ✓ Fully offline | ✓ | ✗ (SaaS) |
| **Zero dependencies** | ✓ One Go binary | ✓ | ✗ |
| **Open source** | ✓ Apache 2.0 | Usually | ✗ |
| **Framework integrations** | 13 | None | Vendor-specific |
| **MCP support** | ✓ Middleware | ✗ | ✗ |
| **MCP Security Proxy** | ✓ Full safety pipeline | ✗ | ✗ |
| **Policy-as-Code** | ✓ Declarative JSON + NociceptionError | ✗ | Vendor-only |
| **Ephemeral identity** | ✓ Broker with 60s TTL | ✗ | Usually |
| **Compensation (undo)** | ✓ Registered reverse actions | ✗ | ✗ |
| **Differential snapshots** | ✓ Changed-files only | ✗ | ✗ |
| **Rollback by hash** | ✓ Time-travel by receipt hash | ✗ | ✗ |
| **PII detection** | ✓ Built-in regex detector | ✗ | Usually |
| **Export formats** | 5 (JSONL, CSV, TXT, PDF, Parquet) | Depends | Usually 1-2 |
| **Real-time streaming** | ✓ NDJSON + CloudEvents webhooks | Sometimes | Usually |
| **Key rotation** | ✓ Ed25519 with rotation record | N/A | Usually |

---

## Why NOT a SaaS?

Audit trails should be as fundamental as `git`. You don't pay a vendor per commit.

- **Your chain. Your keys. Your binary.** Open source, Apache 2.0 licensed.
- **Zero vendors** between you and the truth.
- **Works fully offline** — no internet connection required.
- **No API keys, no monthly bill, no data leaving your machine.**
- **One static Go binary** — copy to an air-gapped machine and it works instantly.

---

## The Bottom Line

> You are responsible for what your agents do.

ANS gives you the receipts — cryptographically signed, hash-linked, timestamped, snapshot-backed, verifiable, exportable, and time-travel-rewindable.

```bash
ans start                    # daemon started
# ... agents run, tools get called ...
ans chain                    # see everything, ordered causally
ans verify --chain           # prove nothing was tampered with
ans time-travel <index>      # rewind to any point in time
ans compensate <index>       # undo an action
ans policy list              # check active policies
ans token list               # see active ephemeral credentials
ans mcp status               # monitor MCP traffic security
ans export --format pdf      # export a compliance-ready audit report
```

---

## License

Apache 2.0 — ship it.

**Questions?** Open an issue at [github.com/Linky-Link-Linky/Agent-Nervous-System/issues](https://github.com/Linky-Link-Linky/Agent-Nervous-System/issues)
