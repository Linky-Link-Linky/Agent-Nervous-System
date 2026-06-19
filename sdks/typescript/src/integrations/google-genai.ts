/**
 * ANS integration for the Google Gen AI TS SDK (@google/genai).
 *
 * Wraps client.models.generateContent() to intercept function_call parts and produce ANS receipts.
 *
 * Usage:
 *   import { GoogleGenAI } from "@google/genai";
 *   import { ANSGenAIClient } from "ans-sdk/integrations/google-genai";
 *
 *   const base = new GoogleGenAI({ apiKey: "..." });
 *   const ansClient = new ANSGenAIClient(base, { agentId: "ans_xyz123" });
 *   const response = await ansClient.models.generateContent({ ... });
 *
 * SPDX-License-Identifier: MIT
 */
import { Client, ANSError, ActionType } from "../client";

let GoogleGenAI: any;
try {
  GoogleGenAI = require("@google/genai");
} catch {
  throw new Error(
    "@google/genai is not installed. Install it with: npm install @google/genai"
  );
}

class _ANSModels {
  private models: any;
  private agentId: string;
  private client: Client;
  private silent: boolean;

  constructor(models: any, agentId: string, client: Client, silent: boolean) {
    this.models = models;
    this.agentId = agentId;
    this.client = client;
    this.silent = silent;
  }

  async generateContent(params: any): Promise<any> {
    const response = await this.models.generateContent(params);
    const candidates = response.candidates || [];
    for (const candidate of candidates) {
      const content = candidate.content;
      if (!content) continue;
      const parts = content.parts || [];
      for (const part of parts) {
        const fc = part.functionCall;
        if (!fc) continue;
        const fnName = fc.name || "unknown";
        const fnArgs: Record<string, unknown> = {};
        const args = fc.args || {};
        if (typeof args === "object") {
          for (const [k, v] of Object.entries(args)) {
            fnArgs[k] = String(v).slice(0, 200);
          }
        }
        const summary = `gemini:${fnName}`.slice(0, 80);
        let preId = "";
        try {
          const resp = await this.client.signAppend({
            agentId: this.agentId,
            phase: "pre",
            actionType: "custom" as ActionType,
            payload: { tool: fnName, ...fnArgs },
            payloadSummary: summary,
            policyDecision: "allow",
            authContext: `google-genai function_call: ${fnName}`,
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
            payload: { tool: fnName },
            payloadSummary: summary,
            outcome: "success",
            outcomeSummary: "function_call returned by model, execution pending",
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

export class ANSGenAIClient {
  private inner: any;
  public models: _ANSModels;

  constructor(
    genaiClient: any,
    opts: { agentId: string; ansClient?: Client; silent?: boolean }
  ) {
    this.inner = genaiClient;
    const client = opts.ansClient || new Client();
    const silent = opts.silent !== false;
    this.models = new _ANSModels(genaiClient.models, opts.agentId, client, silent);
  }

  get [Symbol.toStringTag](): string {
    return "ANSGenAIClient";
  }
}
