/**
 * ANS integration for the Anthropic TypeScript SDK (@anthropic-ai/sdk).
 *
 * Wraps client.messages.create() to intercept tool_use blocks and produce ANS receipts.
 *
 * Usage:
 *   import Anthropic from "@anthropic-ai/sdk";
 *   import { ANSAnthropicClient } from "ans-sdk/integrations/anthropic";
 *
 *   const base = new Anthropic({ apiKey: "..." });
 *   const ansClient = new ANSAnthropicClient(base, { agentId: "ans_xyz123" });
 *   const response = await ansClient.messages.create({ ... });
 *
 * SPDX-License-Identifier: MIT
 */
import { Client, ANSError, ActionType } from "../client";

let Anthropic: any;
try {
  Anthropic = require("@anthropic-ai/sdk");
} catch {
  throw new Error(
    "@anthropic-ai/sdk is not installed. Install it with: npm install @anthropic-ai/sdk"
  );
}

class _ANSMessages {
  private messages: any;
  private agentId: string;
  private client: Client;
  private silent: boolean;

  constructor(messages: any, agentId: string, client: Client, silent: boolean) {
    this.messages = messages;
    this.agentId = agentId;
    this.client = client;
    this.silent = silent;
  }

  async create(params: any): Promise<any> {
    const response = await this.messages.create(params);
    const content = response.content || [];
    for (const block of content) {
      if (block.type === "tool_use") {
        const toolName = block.name || "unknown";
        const toolInput = block.input || {};
        const toolId = block.id || toolName;
        const payload: Record<string, unknown> = { tool: toolName };
        for (const [k, v] of Object.entries(toolInput)) {
          payload[k] = String(v).slice(0, 200);
        }
        const summary = `anthropic:${toolName}`.slice(0, 80);
        let preId = "";
        try {
          const resp = await this.client.signAppend({
            agentId: this.agentId,
            phase: "pre",
            actionType: "custom" as ActionType,
            payload,
            payloadSummary: summary,
            policyDecision: "allow",
            authContext: `anthropic.messages.create tool_use id=${toolId}`,
          });
          preId = resp.receipt_id;
        } catch (e) {
          if (!this.silent) throw e;
          console.error(`ans: pre-receipt error: ${e instanceof ANSError ? e.message : e}`);
        }
        try {
          await this.client.signAppend({
            agentId: this.agentId,
            phase: "post",
            actionType: "custom" as ActionType,
            payload,
            payloadSummary: summary,
            outcome: "success",
            outcomeSummary: "tool_use block returned by model, execution pending",
            preReceiptId: preId,
          });
        } catch (e) {
          if (!this.silent) throw e;
          console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
        }
      }
    }
    return response;
  }
}

export class ANSAnthropicClient {
  private inner: any;
  public messages: _ANSMessages;

  constructor(
    anthropicClient: any,
    opts: { agentId: string; ansClient?: Client; silent?: boolean }
  ) {
    this.inner = anthropicClient;
    const client = opts.ansClient || new Client();
    const silent = opts.silent !== false;
    this.messages = new _ANSMessages(anthropicClient.messages, opts.agentId, client, silent);
  }

  get [Symbol.toStringTag](): string {
    return "ANSAnthropicClient";
  }
}
