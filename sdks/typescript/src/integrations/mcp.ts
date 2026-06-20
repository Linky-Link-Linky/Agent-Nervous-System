/**
 * ANS middleware for MCP (Model Context Protocol) tool handlers.
 *
 * Wraps a tool execution function with pre/post ANS receipts.
 *
 * Usage:
 *   import { ANSMiddleware } from "ans-sdk/integrations/mcp";
 *
 *   const middleware = ANSMiddleware({ agentId: "ans_xyz123" });
 *   const handler = async (toolName: string, args: Record<string, unknown>) => {
 *     return middleware(async () => {
 *       // your tool logic here
 *       return { result: "ok" };
 *     }, toolName, args);
 *   };
 *
 * SPDX-License-Identifier: MIT
 */
import { Client, ANSError, ActionType } from "../client";

// ANSMiddleware is the public API name (factory function, not a class — do not use `new`)
export const ANSMiddleware = ansMiddleware;

export function ansMiddleware(opts: {
  agentId: string;
  client?: Client;
  silent?: boolean;
}): (
  toolFn: () => Promise<unknown>,
  toolName: string,
  args: Record<string, unknown>
) => Promise<unknown> {
  const client = opts.client || new Client();
  const silent = opts.silent !== false;

  return async (
    toolFn: () => Promise<unknown>,
    toolName: string,
    args: Record<string, unknown>
  ): Promise<unknown> => {
    const summary = `mcp:${toolName}`.slice(0, 80);
    let preId = "";
    try {
      const resp = await client.signAppend({
        agentId: opts.agentId,
        phase: "pre",
        actionType: "custom" as ActionType,
        payload: { tool: toolName, ...args },
        payloadSummary: summary,
        policyDecision: "allow",
        authContext: `MCP tool: ${toolName}`,
      });
      preId = resp.receipt_id;
    } catch (e) {
      if (!silent) throw e;
      console.error(`ans: pre-receipt error: ${e instanceof ANSError ? e.message : e}`);
    }

    const start = performance.now();
    let outcome = "success";
    let resultSummary = "";
    try {
      const result = await toolFn();
      resultSummary = result != null ? String(result).slice(0, 120) : "";
      return result;
    } catch (e) {
      outcome = "failure";
      resultSummary = e instanceof Error ? e.message.slice(0, 120) : String(e).slice(0, 120);
      throw e;
    } finally {
      const duration = Math.round(performance.now() - start);
      try {
        await client.signAppend({
          agentId: opts.agentId,
          phase: "post",
          actionType: "custom" as ActionType,
          payload: { tool: toolName },
          payloadSummary: summary,
          outcome,
          outcomeSummary: resultSummary,
          durationMs: duration,
          preReceiptId: preId,
        });
      } catch (e) {
        if (!silent) throw e;
        console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
      }
    }
  };
}
