/**
 * ANS TypeScript SDK — Unix socket / named pipe client (Node.js).
 * One connection per call. Zero external dependencies.
 * SPDX-License-Identifier: MIT
 */
import * as crypto from "crypto";
import * as net from "net";
import * as path from "path";

const MSG = { SIGN_APPEND:0x01, SIGN_APPEND_RESP:0x02, VERIFY:0x03, VERIFY_RESP:0x04,
  QUERY:0x05, QUERY_RESP:0x06, REGISTER:0x07, REGISTER_RESP:0x08,
  STATUS:0x09, STATUS_RESP:0x0a, PING:0x0b, PONG:0x0c, ERROR:0xff } as const;
const MAX_FRAME = 4 * 1024 * 1024;

export type ActionType = "file.read"|"file.write"|"file.delete"|"http.get"|"http.post"|
  "http.other"|"shell.exec"|"db.read"|"db.write"|"agent.delegate"|
  "memory.read"|"memory.write"|"custom";

export interface SignAppendResp { receipt_id:string; chain_index:number; chain_tip:string; signature:string; }
export interface VerifyResp { valid:boolean; receipt_id:string; agent_id:string; agent_name?:string;
  action_type:string; phase:string; policy_decision?:string; outcome?:string;
  timestamp_ns:number; chain_index:number; error?:string; }

export class ANSError extends Error { constructor(msg:string){ super(msg); this.name="ANSError"; } }

function socketPath(): string {
  if (process.platform==="win32") return "\\\\.\\pipe\\ans";
  const xdg = process.env["XDG_RUNTIME_DIR"];
  return xdg ? path.join(xdg,"ans.sock") : "/tmp/ans.sock";
}

function tryParse<T>(s: string): T {
  try { return JSON.parse(s) as T; } catch { throw new ANSError("Invalid JSON response from daemon"); }
}

function tryParseMessage(s: string): string {
  try { return (JSON.parse(s) as {message:string}).message ?? "unknown error"; } catch { return "unknown error"; }
}

function hashPayload(p: Record<string,unknown>): string {
  const sorted = Object.fromEntries(Object.entries(p).sort(([a],[b])=>a.localeCompare(b)));
  return crypto.createHash("sha256").update(JSON.stringify(sorted)).digest("hex");
}

function call<T>(sockPath: string, msgType: number, body: unknown, timeoutMs = 10000): Promise<T> {
  return new Promise((resolve, reject) => {
    const conn = net.createConnection({ path: sockPath });
    let buf = Buffer.alloc(0);
    let done = false;
    const fail = (e: Error) => { if (!done) { done=true; conn.destroy(); reject(e); } };
    conn.on("error", e => fail(new ANSError(`Cannot connect to ANS daemon at ${sockPath}. Run: ans start\n${e.message}`)));
    conn.on("close", () => fail(new ANSError("Connection closed before response")));
    conn.on("end", () => fail(new ANSError("Connection ended before response")));
    const timer = setTimeout(() => fail(new ANSError("Request timed out")), timeoutMs);
    conn.once("connect", () => {
      const encoded = Buffer.from(JSON.stringify(body),"utf8");
      const plen = 1 + encoded.length;
      if (plen > MAX_FRAME) return fail(new ANSError(`Request too large: ${plen}`));
      const frame = Buffer.allocUnsafe(4 + plen);
      frame.writeUInt32BE(plen, 0);
      frame.writeUInt8(msgType, 4);
      encoded.copy(frame, 5);
      conn.write(frame);
    });
    conn.on("data", (chunk: Buffer) => {
      buf = Buffer.concat([buf, chunk]);
      if (buf.length < 4) return;
      const size = buf.readUInt32BE(0);
      if (size === 0 || size > MAX_FRAME) return fail(new ANSError(`Invalid frame size: ${size}`));
      if (buf.length < 4 + size) return;
      clearTimeout(timer);
      const t = buf.readUInt8(4);
      const b = buf.subarray(5, 4 + size);
      conn.destroy();
      if (done) return; done = true;
      if (t === MSG.ERROR) reject(new ANSError(tryParseMessage(b.toString())));
      else resolve(tryParse<T>(b.toString()));
    });
  });
}

export class Client {
  private readonly sockPath: string;
  constructor(sockPath?: string) { this.sockPath = sockPath ?? socketPath(); }

  async ping(): Promise<boolean> {
    return new Promise(resolve => {
      const c = net.createConnection({ path: this.sockPath });
      c.on("error", () => resolve(false));
      c.once("connect", () => { const f=Buffer.allocUnsafe(5); f.writeUInt32BE(1,0); f.writeUInt8(MSG.PING,4); c.write(f); });
      c.on("data", () => { c.destroy(); resolve(true); });
      setTimeout(() => { c.destroy(); resolve(false); }, 2000);
    });
  }

  async register(p: {name:string; version?:string; owner?:string; metadata?:Record<string,string>}): Promise<string> {
    const r = await call<{agent_id:string}>(this.sockPath, MSG.REGISTER,
      {name:p.name, version:p.version??"0.1.0", owner:p.owner??"", metadata:p.metadata??{}});
    return r.agent_id;
  }

  async signAppend(p: {agentId:string; phase:"pre"|"post"; actionType:ActionType;
    payload:Record<string,unknown>; payloadSummary?:string; policyDecision?:string;
    authContext?:string; outcome?:string; outcomeSummary?:string; durationMs?:number;
    preReceiptId?:string; parentAgentId?:string}): Promise<SignAppendResp> {
    return call<SignAppendResp>(this.sockPath, MSG.SIGN_APPEND, {
      agent_id:p.agentId, phase:p.phase, action_type:p.actionType,
      payload_hash:hashPayload(p.payload),
      payload_summary:(p.payloadSummary??"").slice(0,80),
      policy_decision:p.policyDecision??"allow", auth_context:p.authContext??"",
      outcome:p.outcome??"", outcome_summary:(p.outcomeSummary??"").slice(0,120),
      duration_ms:p.durationMs??0, pre_receipt_id:p.preReceiptId??"",
      parent_agent_id:p.parentAgentId??"",
    });
  }

  async verify(receiptId: string): Promise<VerifyResp> {
    return call<VerifyResp>(this.sockPath, MSG.VERIFY, {receipt_id:receiptId});
  }

  async status(): Promise<Record<string,unknown>> {
    return call(this.sockPath, MSG.STATUS, {});
  }

  async query(p?: {agentId?:string; actionType?:string; phase?:string; limit?:number; offset?:number}): Promise<unknown[]> {
    return call<unknown[]>(this.sockPath, MSG.QUERY, {
      agent_id:p?.agentId??"", action_type:p?.actionType??"",
      phase:p?.phase??"", limit:p?.limit??20, offset:p?.offset??0
    });
  }
}
