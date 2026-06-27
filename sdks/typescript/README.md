# ANS TypeScript SDK

Cryptographic audit receipts for every AI agent tool call

## Install

```bash
npm install ans-sdk
```

## Quick Start

```typescript
import { Client, wrap } from "ans-sdk";

const client = new Client();
const traced = wrap(myTool, { agentId: "ans_abc123", actionType: "custom", client });
const result = await traced("arg1", "arg2");
```

## Client API

```typescript
import { Client } from "ans-sdk";

const client = new Client();

// Ping the daemon
await client.ping();

// Register an agent
await client.register("my-agent", { version: "1.0", owner: "me" });

// Sign and append a receipt
const resp = await client.signAppend({
  agentId: "ans_abc123",
  phase: "pre",
  actionType: "file.write",
  payload: { path: "/tmp/test.txt" },
  payloadSummary: "write file",
  policyDecision: "allow",
});

// Verify a receipt
const result = await client.verify("abc123...");
console.log(result.valid, result.receipt);

// Daemon status
const status = await client.status();

// Query receipts
const receipts = await client.query({ limit: 10, agentId: "ans_abc123" });
```

## Function Wrapping

The `wrap()` function automatically creates pre/post receipts around any function:

```typescript
import { wrap } from "ans-sdk";

const tracedFn = wrap(myFunction, {
  agentId: "ans_abc123",
  actionType: "custom",
  silent: true,  // don't throw on ANS errors (default)
});
```

Works with both sync and async functions.

## Integrations

```typescript
// OpenAI Agents SDK
import { ansAgentPlugin } from "ans-sdk/integrations/openai-agents";

// Anthropic Claude
import { ANSAnthropicClient } from "ans-sdk/integrations/anthropic";

// Google Generative AI
import { ANSGenAIClient } from "ans-sdk/integrations/google-genai";

// MCP middleware
import { ansMiddleware } from "ans-sdk/integrations/mcp";
```
