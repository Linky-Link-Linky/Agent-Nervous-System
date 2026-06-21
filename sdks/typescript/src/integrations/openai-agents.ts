/**
 * ANS integration for the OpenAI Agents JS/TS SDK.
 *
 * Provides ansAgentPlugin — a factory that wraps tool functions with ANS receipts.
 *
 * Usage:
 *   import { ansAgentPlugin } from "ans-sdk/integrations/openai-agents";
 *   import { tool } from "@openai/agents";
 *
 *   const plugin = ansAgentPlugin({ agentId: "ans_xyz123" });
 *   const wrappedFn = plugin(myTool, "my_tool_name");
 *
 * SPDX-License-Identifier: Apache-2.0
 */
import { Client, ANSError, ActionType } from "../client";

export function ansAgentPlugin(opts: {
  agentId: string;
  client?: Client;
  silent?: boolean;
}): (fn: Function, toolName: string) => Function {
  const client = opts.client || new Client();
  const silent = opts.silent !== false;

  return (fn: Function, toolName: string): Function => {
    return async (...args: unknown[]) => {
      const summary = `openai:${toolName}`.slice(0, 80);
      let preId = "";
      try {
        const resp = await client.signAppend({
          agentId: opts.agentId,
          phase: "pre",
          actionType: "custom" as ActionType,
          payload: { tool: toolName, args: JSON.stringify(args).slice(0, 200) },
          payloadSummary: summary,
          policyDecision: "allow",
          authContext: `OpenAI Agents tool: ${toolName}`,
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
        const result = await fn(...args);
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
  };
}
