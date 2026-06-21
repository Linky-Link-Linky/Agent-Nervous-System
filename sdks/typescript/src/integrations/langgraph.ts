/**
 * LangGraph integration for ANS.
 *
 * Provides ANSGraph — a drop-in wrapper for StateGraph that automatically
 * traces every node execution with ANS receipts.
 *
 * Usage:
 *   import { ANSGraph } from "ans-sdk/integrations/langgraph";
 *   import { Client } from "ans-sdk";
 *
 *   const graph = new ANSGraph({ agentId: "ans_xyz123" });
 *   graph.addNode("my_node", (state) => ({ ...state, processed: true }));
 *   graph.addEdge("__start__", "my_node");
 *   const app = graph.compile();
 *   const result = await app.invoke({ input: "hello" });
 *
 * SPDX-License-Identifier: Apache-2.0
 */
import { Client, ANSError, ActionType } from "../client";
import { wrap } from "../wrap";

let StateGraph: any;
try {
  StateGraph = require("@langchain/langgraph").StateGraph;
} catch {
  throw new Error(
    "@langchain/langgraph is not installed. Install it with: npm install @langchain/langgraph"
  );
}

export function ansNode(
  func: (...args: unknown[]) => unknown,
  opts: { agentId: string; client?: Client; silent?: boolean; actionType?: ActionType }
): (...args: unknown[]) => unknown {
  return wrap(func, {
    agentId: opts.agentId,
    actionType: opts.actionType || ("agent.delegate" as ActionType),
    client: opts.client,
    silent: opts.silent,
  });
}

export class ANSGraph {
  private client: Client;
  private agentId: string;
  private silent: boolean;
  private graph: any;

  constructor(opts: { agentId: string; client?: Client; silent?: boolean }) {
    this.client = opts.client || new Client();
    this.agentId = opts.agentId;
    this.silent = opts.silent !== false;
    this.graph = new StateGraph({ channels: {} });
  }

  addNode(name: string, func: (...args: unknown[]) => unknown): void {
    const wrapped = wrap(func, {
      agentId: this.agentId,
      actionType: "agent.delegate" as ActionType,
      client: this.client,
      silent: this.silent,
    });
    this.graph.addNode(name, wrapped);
  }

  addEdge(start: string, end: string): void {
    this.graph.addEdge(start, end);
  }

  addConditionalEdges(source: string, condition: (...args: unknown[]) => string, mapping: Record<string, string>): void {
    this.graph.addConditionalEdges(source, condition, mapping);
  }

  setEntryPoint(node: string): void {
    this.graph.setEntryPoint(node);
  }

  setFinishPoint(node: string): void {
    this.graph.setFinishPoint(node);
  }

  compile(): any {
    return this.graph.compile();
  }
}
