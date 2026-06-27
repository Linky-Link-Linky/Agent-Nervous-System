"""Google Generative AI (Gemini) integration for ANS."""

import asyncio
import sys
import time
from typing import Any

try:
    import google.generativeai as genai
    _genai_available = True
except ImportError:
    genai = None
    _genai_available = False

from ..client import ANSClient, ANSError, hash_payload


def _sign_pre(client: ANSClient, agent_id: str, parent_agent_id: str, ph: str, summary: str, silent: bool) -> str:
    try:
        resp = client.sign_append(
            agent_id=agent_id, phase="pre", action_type="agent.delegate",
            payload_hash=ph, payload_summary=summary,
            policy_decision="allow", parent_agent_id=parent_agent_id,
        )
        return resp["receipt_id"]
    except ANSError as e:
        if not silent: raise
        print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
        return ""


def _sign_post(client: ANSClient, agent_id: str, parent_agent_id: str, ph: str, summary: str, outcome: str, o_sum: str, dur: int, pre_id: str, silent: bool) -> None:
    if not pre_id: return
    try:
        client.sign_append(
            agent_id=agent_id, phase="post", action_type="agent.delegate",
            payload_hash=ph, payload_summary=summary,
            outcome=outcome, outcome_summary=o_sum,
            duration_ms=dur, pre_receipt_id=pre_id,
            parent_agent_id=parent_agent_id,
        )
    except ANSError as e:
        if not silent: raise
        print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)


class ANSGenAIClient:
    """
    Wrapper for Google Generative AI (Gemini) client with ANS tracing.

    Example:
        import google.generativeai as genai
        from ans.integrations.google_genai import ANSGenAIClient

        genai.configure(api_key="...")
        base_model = genai.GenerativeModel("gemini-pro")

        ans_client = ANSClient()
        model = ANSGenAIClient(base_model, ans_client, agent_id="ans_xyz")

        response = model.generate_content("Hello")
    """

    def __init__(self, base_model: Any, ans_client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
        if not _genai_available:
            raise ImportError("Google Generative AI is not installed. Install with: pip install ans-sdk[google-genai]")
        self.base_model = base_model
        self.ans_client = ans_client
        self.agent_id = agent_id
        self.parent_agent_id = parent_agent_id
        self.silent = silent

    def _invoke(self, *args, **kwargs) -> tuple:
        prompt = args[0] if args else kwargs.get("contents", "")
        model_name = getattr(self.base_model, "model_name", "gemini")
        payload = {"model": model_name, "prompt": prompt}
        ph = hash_payload(payload)
        summary = f"Gemini: {model_name}"
        return ph, summary

    def _outcome_summary(self, result: Any) -> str:
        if hasattr(result, "text"):
            return result.text[:120]
        return "Completed"

    def generate_content(self, *args, **kwargs) -> Any:
        ph, summary = self._invoke(*args, **kwargs)
        pre_id = _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent)

        start = time.time()
        outcome, o_sum, result, error = "success", "", None, None
        try:
            result = self.base_model.generate_content(*args, **kwargs)
            o_sum = self._outcome_summary(result)
        except Exception as e:
            outcome, o_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, o_sum,
                   int((time.time() - start) * 1000), pre_id, self.silent)
        if error: raise error
        return result

    async def generate_content_async(self, *args, **kwargs) -> Any:
        loop = asyncio.get_running_loop()
        ph, summary = self._invoke(*args, **kwargs)
        pre_id = await loop.run_in_executor(None, lambda: _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent))

        start = time.time()
        outcome, o_sum, result, error = "success", "", None, None
        try:
            result = await self.base_model.generate_content_async(*args, **kwargs)
            o_sum = self._outcome_summary(result)
        except Exception as e:
            outcome, o_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        await loop.run_in_executor(None, lambda: _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, o_sum,
                                   int((time.time() - start) * 1000), pre_id, self.silent))
        if error: raise error
        return result
