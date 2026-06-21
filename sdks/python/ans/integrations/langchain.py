"""LangChain integration for ANS."""

import sys
import time
from typing import Any, Optional

try:
    from langchain.callbacks.base import BaseCallbackHandler
    from langchain.schema import AgentAction, AgentFinish, LLMResult
    _langchain_available = True
except ImportError:
    BaseCallbackHandler = object
    AgentAction = None
    AgentFinish = None
    LLMResult = None
    _langchain_available = False

from ..client import ANSClient, ANSError, hash_payload


class ANSCallbackHandler(BaseCallbackHandler):
    """
    LangChain callback handler that logs agent actions to ANS.

    Example:
        from ans.integrations.langchain import ANSCallbackHandler
        from langchain.chains import LLMChain

        handler = ANSCallbackHandler(client, agent_id="ans_xyz")
        chain = LLMChain(llm=llm, prompt=prompt)
        chain.invoke({"input": "..."}, callbacks=[handler])
    """

    def __init__(self, client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
        if not _langchain_available:
            raise ImportError("LangChain is not installed. Install with: pip install ans-sdk[langchain]")
        super().__init__()
        self.client = client
        self.agent_id = agent_id
        self.parent_agent_id = parent_agent_id
        self.silent = silent
        self._pending: list[tuple[str, str, float]] = []  # (payload_hash, pre_id, start_time)

    def _pre(self, ph: str, summary: str) -> str:
        try:
            resp = self.client.sign_append(
                agent_id=self.agent_id, phase="pre", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                policy_decision="allow", parent_agent_id=self.parent_agent_id,
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not self.silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post(self, ph: str, summary: str, outcome: str, o_sum: str, pre_id: str, duration_ms: int = 0) -> None:
        if not pre_id: return
        try:
            self.client.sign_append(
                agent_id=self.agent_id, phase="post", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                outcome=outcome, outcome_summary=o_sum,
                duration_ms=duration_ms, pre_receipt_id=pre_id,
                parent_agent_id=self.parent_agent_id,
            )
        except ANSError as e:
            if not self.silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def on_agent_action(self, action: AgentAction, **kwargs: Any) -> Any:
        payload = {
            "tool": action.tool,
            "tool_input": action.tool_input,
            "log": action.log,
        }
        ph = hash_payload(payload)
        summary = f"Tool: {action.tool}"
        now = time.time()
        pre_id = self._pre(ph, summary)
        if pre_id:
            self._pending.append((ph, pre_id, now))

    def _pop_pending(self) -> tuple:
        if not self._pending:
            return ("", "", 0)
        ph, pre_id, start = self._pending.pop(0)
        return (ph, pre_id, start)

    def on_tool_end(self, output: str, observation_prefix: Optional[str] = None, **kwargs: Any) -> Any:
        ph, pre_id, start = self._pop_pending()
        duration_ms = int((time.time() - start) * 1000) if start else 0
        o_sum = output[:120] if output else "Completed"
        self._post(ph, "Tool completed", "success", o_sum, pre_id, duration_ms)

    def on_tool_error(self, error: Exception, **kwargs: Any) -> Any:
        ph, pre_id, start = self._pop_pending()
        duration_ms = int((time.time() - start) * 1000) if start else 0
        o_sum = f"Error: {type(error).__name__}: {str(error)[:100]}"
        self._post(ph, "Tool failed", "failure", o_sum, pre_id, duration_ms)

    def on_agent_finish(self, finish: AgentFinish, **kwargs: Any) -> Any:
        payload = {"output": finish.return_values, "log": finish.log}
        ph = hash_payload(payload)
        now = time.time()
        pre_id = self._pre(ph, "Agent finished")
        # Close any dangling pre-receipts with a neutral post receipt
        for dangling_ph, dangling_pre_id, dangling_start in self._pending:
            dur = int((time.time() - dangling_start) * 1000) if dangling_start else 0
            self._post(dangling_ph, "Tool (interrupted)", "failure", "Interrupted: agent finished", dangling_pre_id, dur)
        self._pending.clear()
        dur = int((time.time() - now) * 1000)
        self._post(ph, "Agent finished", "success", "Agent completed", pre_id, dur)
