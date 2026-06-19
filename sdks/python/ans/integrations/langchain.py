"""LangChain integration for ANS."""

import sys
import time
from typing import Any, Dict, Optional

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
        self._action_receipts: Dict[str, str] = {}
        self._action_starts: Dict[str, float] = {}

    def _pre(self, ph: str, summary: str) -> str:
        self._action_starts[ph] = time.time()
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

    def _post(self, ph: str, summary: str, outcome: str, o_sum: str, pre_id: str) -> None:
        if not pre_id: return
        start = self._action_starts.pop(ph, 0)
        duration_ms = int((time.time() - start) * 1000) if start else 0
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
        pre_id = self._pre(ph, summary)
        if pre_id:
            self._action_receipts[ph] = pre_id

    def _pop_receipt(self) -> tuple:
        if not self._action_receipts:
            return ("", "")
        ph = list(self._action_receipts.keys())[-1]
        pre_id = self._action_receipts.pop(ph)
        return (ph, pre_id)

    def on_tool_end(self, output: str, observation_prefix: Optional[str] = None, **kwargs: Any) -> Any:
        ph, pre_id = self._pop_receipt()
        o_sum = output[:120] if output else "Completed"
        self._post(ph, "Tool completed", "success", o_sum, pre_id)

    def on_tool_error(self, error: Exception, **kwargs: Any) -> Any:
        ph, pre_id = self._pop_receipt()
        o_sum = f"Error: {type(error).__name__}: {str(error)[:100]}"
        self._post(ph, "Tool failed", "failure", o_sum, pre_id)

    def on_agent_finish(self, finish: AgentFinish, **kwargs: Any) -> Any:
        payload = {"output": finish.return_values, "log": finish.log}
        ph = hash_payload(payload)
        pre_id = self._pre(ph, "Agent finished")
        # Close any dangling pre-receipts with a neutral post receipt
        for dangling_ph, dangling_pre_id in list(self._action_receipts.items()):
            self._post(dangling_ph, "Tool (interrupted)", "failure", "Interrupted: agent finished", dangling_pre_id)
            del self._action_receipts[dangling_ph]
        self._post(ph, "Agent finished", "success", "Agent completed", pre_id)
