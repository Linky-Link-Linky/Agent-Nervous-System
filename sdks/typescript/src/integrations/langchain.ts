/**
 * ANS integration for LangChain.js.
 * Provides ANSCallbackHandler that wraps tool calls with ANS receipts.
 *
 * Usage:
 *   import { ANSCallbackHandler } from "ans-sdk/integrations/langchain";
 *   import { RunnableSequence } from "@langchain/core/runnables";
 *
 *   const chain = RunnableSequence.from([...]);
 *   const result = await chain.invoke(input, {
 *     callbacks: [new ANSCallbackHandler({ agentId: "ans_xyz123" })]
 *   });
 *
 * SPDX-License-Identifier: Apache-2.0
 */
import { Client, ANSError, ActionType } from "../client";

let BaseCallbackHandler: any;
try {
  BaseCallbackHandler = require("@langchain/core/callbacks/base").BaseCallbackHandler;
} catch {
  throw new Error(
    "@langchain/core is not installed. Install it with: npm install @langchain/core"
  );
}

export class ANSCallbackHandler extends BaseCallbackHandler {
  name = "ANSCallbackHandler";
  private agentId: string;
  private client: Client;
  private silent: boolean;
  private preReceiptIds = new Map<string, string>();

  constructor(opts: { agentId: string; client?: Client; silent?: boolean }) {
    super();
    this.agentId = opts.agentId;
    this.client = opts.client || new Client();
    this.silent = opts.silent !== false;
  }

  async handleToolStart(tool: { name: string }, input: string, runId: string): Promise<void> {
    const summary = `tool:${tool.name} ${String(input).slice(0, 40)}`.slice(0, 80);
    try {
      const resp = await this.client.signAppend({
        agentId: this.agentId,
        phase: "pre",
        actionType: "custom" as ActionType,
        payload: { tool: tool.name, input: String(input).slice(0, 200) },
        payloadSummary: summary,
        policyDecision: "allow",
        authContext: `LangChain tool: ${tool.name}`,
      });
      this.preReceiptIds.set(runId, resp.receipt_id);
    } catch (e) {
      if (!this.silent) throw e;
      console.error(`ans: pre-receipt error: ${e instanceof ANSError ? e.message : e}`);
    }
  }

  async handleToolEnd(output: string, runId: string): Promise<void> {
    const preId = this.preReceiptIds.get(runId) || "";
    try {
      await this.client.signAppend({
        agentId: this.agentId,
        phase: "post",
        actionType: "custom" as ActionType,
        payload: {},
        payloadSummary: "",
        outcome: "success",
        outcomeSummary: String(output).slice(0, 120),
        preReceiptId: preId,
      });
    } catch (e) {
      if (!this.silent) throw e;
      console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
    } finally {
      this.preReceiptIds.delete(runId);
    }
  }

  async handleToolError(err: Error, runId: string): Promise<void> {
    const preId = this.preReceiptIds.get(runId) || "";
    try {
      await this.client.signAppend({
        agentId: this.agentId,
        phase: "post",
        actionType: "custom" as ActionType,
        payload: {},
        payloadSummary: "",
        outcome: "failure",
        outcomeSummary: err.message.slice(0, 120),
        preReceiptId: preId,
      });
    } catch (e) {
      if (!this.silent) throw e;
      console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
    } finally {
      this.preReceiptIds.delete(runId);
    }
  }
}
