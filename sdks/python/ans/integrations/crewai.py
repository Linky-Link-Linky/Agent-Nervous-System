"""CrewAI integration for ANS."""

import sys
import time
from typing import Any, Optional

try:
    from crewai.tools import BaseTool
    _crewai_available = True
except ImportError:
    BaseTool = object
    _crewai_available = False

try:
    from pydantic import BaseModel, Field
except ImportError:
    BaseModel = object
    Field = lambda default=None, **kw: default

from ..client import ANSClient, ANSError, hash_payload


class ANSTool(BaseTool):
    """
    Base class for CrewAI tools with ANS tracing.

    Subclass this instead of BaseTool to automatically get ANS receipt generation.

    Example:
        import os
        from ans.integrations.crewai import ANSTool
        from pydantic import BaseModel

        class WriteFileArgs(BaseModel):
            path: str
            content: str

        class WriteFileTool(ANSTool):
            name: str = "write_file"
            description: str = "Write content to a file"
            args_schema: Type[BaseModel] = WriteFileArgs

            def _run(self, path: str, content: str) -> str:
                safe_path = os.path.realpath(path)
                with open(safe_path, 'w') as f:
                    f.write(content)
                return f"Wrote {len(content)} bytes to {safe_path}"
    """

    ans_client: Optional[ANSClient] = Field(default=None, exclude=True)
    ans_agent_id: Optional[str] = Field(default=None, exclude=True)
    ans_parent_agent_id: str = Field(default="", exclude=True)
    ans_action_type: str = Field(default="custom", exclude=True)
    ans_silent: bool = Field(default=True, exclude=True)

    def __init__(self, ans_client: ANSClient, ans_agent_id: str, ans_parent_agent_id: str = "", ans_action_type: str = "custom", ans_silent: bool = True, **kwargs):
        if not _crewai_available:
            raise ImportError("CrewAI is not installed. Install with: pip install ans-sdk[crewai]")
        super().__init__(**kwargs)
        self.ans_client = ans_client
        self.ans_agent_id = ans_agent_id
        self.ans_parent_agent_id = ans_parent_agent_id
        self.ans_action_type = ans_action_type
        self.ans_silent = ans_silent

    def _pre(self, ph: str, summary: str) -> str:
        try:
            resp = self.ans_client.sign_append(
                agent_id=self.ans_agent_id, phase="pre", action_type=self.ans_action_type,
                payload_hash=ph, payload_summary=summary,
                policy_decision="allow", parent_agent_id=self.ans_parent_agent_id,
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not self.ans_silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post(self, ph: str, summary: str, outcome: str, o_sum: str, dur: int, pre_id: str) -> None:
        if not pre_id: return
        try:
            self.ans_client.sign_append(
                agent_id=self.ans_agent_id, phase="post", action_type=self.ans_action_type,
                payload_hash=ph, payload_summary=summary,
                outcome=outcome, outcome_summary=o_sum,
                duration_ms=dur, pre_receipt_id=pre_id,
                parent_agent_id=self.ans_parent_agent_id,
            )
        except ANSError as e:
            if not self.ans_silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def run(self, *args, **kwargs) -> Any:
        if self.ans_client is None or self.ans_agent_id is None:
            return self._run(*args, **kwargs)

        payload = {"args": args, "kwargs": kwargs}
        ph = hash_payload(payload)
        summary = f"Tool: {self.name}"
        pre_id = self._pre(ph, summary)

        start = time.time()
        outcome, o_sum, result, error = "success", "", None, None
        try:
            result = self._run(*args, **kwargs)
            o_sum = str(result)[:120] if result else "Completed"
        except Exception as e:
            outcome, o_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        self._post(ph, summary, outcome, o_sum, int((time.time() - start) * 1000), pre_id)
        if error: raise error
        return result
