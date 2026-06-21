/**
 * ANS TypeScript SDK — function wrapper.
 * Wraps any sync or async function with pre/post receipts.
 * SPDX-License-Identifier: Apache-2.0
 */
import { Client, ANSError, ActionType, SignAppendResp } from "./client";

// Detect async functions by checking the constructor name.
// This avoids calling the function just to detect its type.
function isAsyncFunction(fn: (...args: unknown[]) => unknown): boolean {
  return Object.getPrototypeOf(fn).constructor.name === "AsyncFunction";
}

/**
 * Wraps any function (sync or async) with ANS pre/post receipts.
 * Detects async at wrap time using Object.getPrototypeOf.
 *
 * @param fn The function to wrap
 * @param opts Options including agentId, actionType, optional client, silent flag
 * @returns A wrapped function with the same signature
 */
export function wrap<T extends (...args: unknown[]) => unknown>(
  fn: T,
  opts: {
    agentId: string;
    actionType?: ActionType;
    client?: Client;
    silent?: boolean;
  }
): T {
  const client = opts.client || new Client();
  const silent = opts.silent !== false;
  const isAsync = isAsyncFunction(fn);

  if (isAsync) {
    return (async (...args: unknown[]) => {
      const actionType: ActionType = opts.actionType || "custom" as ActionType;
      let argsStr: string;
      try { argsStr = JSON.stringify(args).slice(0, 200); } catch { argsStr = "[unserializable]"; }
      const payload = { function: fn.name || "anonymous", args: argsStr };
      const summary = `${fn.name || "anonymous"}(${args.map(a => String(a).slice(0, 30)).join(", ")})`.slice(0, 80);

      let preId = "";
      try {
        const resp = await client.signAppend({
          agentId: opts.agentId,
          phase: "pre",
          actionType,
          payload,
          payloadSummary: summary,
          policyDecision: "allow",
          authContext: `wrap() on ${fn.name || "anonymous"}`,
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
            actionType,
            payload: { function: fn.name || "anonymous" },
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
    }) as T;
  } else {
    return ((...args: unknown[]) => {
      const actionType: ActionType = opts.actionType || "custom" as ActionType;
      let argsStr: string;
      try { argsStr = JSON.stringify(args).slice(0, 200); } catch { argsStr = "[unserializable]"; }
      const payload = { function: fn.name || "anonymous", args: argsStr };
      const summary = `${fn.name || "anonymous"}(${args.map(a => String(a).slice(0, 30)).join(", ")})`.slice(0, 80);

      // Fire-and-forget pre receipt (cannot await in sync context)
      const prePromise: Promise<SignAppendResp | undefined> = client.signAppend({
        agentId: opts.agentId,
        phase: "pre",
        actionType,
        payload,
        payloadSummary: summary,
        policyDecision: "allow",
        authContext: `wrap() on ${fn.name || "anonymous"}`,
      }).catch(e => {
        if (!silent) throw e;
        console.error(`ans: pre-receipt error: ${e instanceof ANSError ? e.message : e}`);
        return undefined;
      });

      const start = performance.now();
      let outcome = "success";
      let resultSummary = "";
      try {
        const result = fn(...args);
        resultSummary = result != null ? String(result).slice(0, 120) : "";
        return result;
      } catch (e) {
        outcome = "failure";
        resultSummary = e instanceof Error ? e.message.slice(0, 120) : String(e).slice(0, 120);
        throw e;
      } finally {
        const duration = Math.round(performance.now() - start);
        // Chain: wait for pre-receipt, then send post-receipt with link
        prePromise.then(preResp => {
          client.signAppend({
            agentId: opts.agentId,
            phase: "post",
            actionType,
            payload: { function: fn.name || "anonymous" },
            payloadSummary: summary,
            outcome,
            outcomeSummary: resultSummary,
            durationMs: duration,
            preReceiptId: preResp?.receipt_id || "",
          }).catch(e => {
            if (!silent) throw e;
            console.error(`ans: post-receipt error: ${e instanceof ANSError ? e.message : e}`);
          });
        }).catch(e => {
          if (!silent) throw e;
        });
      }
    }) as T;
  }
}
