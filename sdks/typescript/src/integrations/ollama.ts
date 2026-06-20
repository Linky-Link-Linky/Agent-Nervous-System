/**
 * ANS integration for Ollama — open-weight models running locally.
 *
 * Wraps Ollama's chat() and generate() to produce ANS receipts.
 *
 * Usage:
 *   import ollama from "ollama";
 *   import { ANSOllamaClient } from "ans-sdk/integrations/ollama";
 *
 *   const client = new ANSOllamaClient(ollama, { agentId: "ans_xyz123" });
 *   const response = await client.chat({ model: "llama3", messages: [...] });
 *
 * SPDX-License-Identifier: MIT
 */
import { Client, ANSError, ActionType } from "../client";

let ollama: any;
try {
  ollama = require("ollama");
} catch {
  throw new Error(
    "ollama is not installed. Install it with: npm install ollama"
  );
}

async function signPre(client: Client, agentId: string, payload: Record<string, unknown>, summary: string, silent: boolean): Promise<string | null> {
  try {
    const resp = await client.signAppend({
      agentId,
      phase: "pre",
      actionType: "agent.delegate" as ActionType,
      payload,
      payloadSummary: summary,
      policyDecision: "allow",
      authContext: "Ollama call",
    });
    return resp.receipt_id;
  } catch (e) {
    if (!silent) throw e;
    console.error(`ans: pre-receipt error: ${e instanceof ANSError ? e.message : e}`);
    return null;
  }
}

async function signPost(client: Client, agentId: string, payload: Record<string, unknown>, summary: string, outcome: string, outcomeSummary: string, durationMs: number, preReceiptId: string | null, silent: boolean): Promise<void> {
  if (!preReceiptId) return;
  try {
    await client.signAppend({
      agentId,
      phase: "post",
      actionType: "agent.delegate" as ActionType,
      payload,
      payloadSummary: summary,
      outcome,
      outcomeSummary,
      durationMs,
      preReceiptId,
    });
  } catch (e) {
    if (!silent) throw e;
    console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
  }
}

function chatPayload(params: any): Record<string, unknown> {
  return { model: params.model || "unknown", messages: params.messages || [] };
}

function generatePayload(params: any): Record<string, unknown> {
  return { model: params.model || "unknown", prompt: params.prompt || "" };
}

function chatOutcome(result: any): string {
  if (result && result.message && result.message.content) {
    return String(result.message.content).slice(0, 120);
  }
  return "Completed";
}

function generateOutcome(result: any): string {
  if (result && result.response) {
    return String(result.response).slice(0, 120);
  }
  return "Completed";
}

export class ANSOllamaClient {
  private baseClient: any;
  private agentId: string;
  private client: Client;
  private silent: boolean;

  constructor(
    ollamaClient: any,
    opts: { agentId: string; client?: Client; silent?: boolean }
  ) {
    this.baseClient = ollamaClient;
    this.agentId = opts.agentId;
    this.client = opts.client || new Client();
    this.silent = opts.silent !== false;
  }

  async chat(params: any): Promise<any> {
    const payload = chatPayload(params);
    const summary = `Ollama: ${params.model || "unknown"}`.slice(0, 80);
    const preId = await signPre(this.client, this.agentId, payload, summary, this.silent);

    const start = Date.now();
    let outcome = "success";
    let outcomeSummary = "";
    let result: any;
    let error: any;
    try {
      result = await this.baseClient.chat(params);
      outcomeSummary = chatOutcome(result);
    } catch (e) {
      outcome = "failure";
      outcomeSummary = `Error: ${(e as Error).name}: ${String((e as Error).message).slice(0, 100)}`;
      error = e;
    }

    await signPost(this.client, this.agentId, payload, summary, outcome, outcomeSummary, Date.now() - start, preId, this.silent);
    if (error) throw error;
    return result;
  }

  async generate(params: any): Promise<any> {
    const payload = generatePayload(params);
    const summary = `Ollama: ${params.model || "unknown"}`.slice(0, 80);
    const preId = await signPre(this.client, this.agentId, payload, summary, this.silent);

    const start = Date.now();
    let outcome = "success";
    let outcomeSummary = "";
    let result: any;
    let error: any;
    try {
      result = await this.baseClient.generate(params);
      outcomeSummary = generateOutcome(result);
    } catch (e) {
      outcome = "failure";
      outcomeSummary = `Error: ${(e as Error).name}: ${String((e as Error).message).slice(0, 100)}`;
      error = e;
    }

    await signPost(this.client, this.agentId, payload, summary, outcome, outcomeSummary, Date.now() - start, preId, this.silent);
    if (error) throw error;
    return result;
  }

  get [Symbol.toStringTag](): string {
    return "ANSOllamaClient";
  }
}
