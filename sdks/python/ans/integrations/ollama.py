"""Ollama integration for ANS — open-weight models running locally."""

import asyncio
import sys
import time
from typing import Any, Optional

try:
    from ollama import Client, AsyncClient
    _ollama_available = True
except ImportError:
    Client = None
    AsyncClient = None
    _ollama_available = False

from ..client import ANSClient, ANSError, hash_payload


def _sign_pre(client: ANSClient, agent_id: str, parent_agent_id: str, payload_hash: str, summary: str, silent: bool = True) -> str:
	try:
		resp = client.sign_append(
			agent_id=agent_id, phase="pre", action_type="agent.delegate",
			payload_hash=payload_hash, payload_summary=summary,
			policy_decision="allow", parent_agent_id=parent_agent_id,
		)
		return resp["receipt_id"]
	except ANSError as e:
		if not silent:
			raise
		print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
		return ""


def _sign_post(client: ANSClient, agent_id: str, parent_agent_id: str, payload_hash: str, summary: str, outcome: str, outcome_summary: str, duration_ms: int, pre_receipt_id: str, silent: bool = True) -> None:
	if not pre_receipt_id:
		return
	try:
		client.sign_append(
			agent_id=agent_id, phase="post", action_type="agent.delegate",
			payload_hash=payload_hash, payload_summary=summary,
			outcome=outcome, outcome_summary=outcome_summary,
			duration_ms=duration_ms, pre_receipt_id=pre_receipt_id,
			parent_agent_id=parent_agent_id,
		)
	except ANSError as e:
		if not silent:
			raise
		print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)


class _BaseOllamaWrapper:
    """Shared pre/post logic for sync and async Ollama wrappers."""

    def __init__(self, ans_client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
        self.ans_client = ans_client
        self.agent_id = agent_id
        self.parent_agent_id = parent_agent_id
        self.silent = silent

    def _chat_payload(self, kwargs: dict):
        return {"model": kwargs.get("model", "unknown"), "messages": kwargs.get("messages", [])}

    def _generate_payload(self, kwargs: dict):
        return {"model": kwargs.get("model", "unknown"), "prompt": kwargs.get("prompt", "")}

    def _chat_outcome(self, result: Any) -> str:
        if isinstance(result, dict) and "message" in result:
            content = result["message"].get("content", "")
            return content[:120] if content else "Completed"
        return "Completed"

    def _generate_outcome(self, result: Any) -> str:
        if isinstance(result, dict) and "response" in result:
            return result["response"][:120]
        return "Completed"


class ANSOllamaClient(_BaseOllamaWrapper):
    """
    Wrapper for Ollama Client that logs all chat/generate calls to ANS.

    Works with any model Ollama supports (Llama 3, Mistral, Gemma, DeepSeek,
    Phi, Qwen, etc.) — open-weight models running locally.

    Example:
        from ollama import Client
        from ans.integrations.ollama import ANSOllamaClient

        base = Client()
        ans = ANSClient()
        client = ANSOllamaClient(base, ans, agent_id="ans_xyz")

        response = client.chat(model="llama3", messages=[{"role": "user", "content": "H"}])
    """

    def __init__(self, base_client: Client, ans_client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
        if not _ollama_available:
            raise ImportError("Ollama is not installed. Install with: pip install ans-sdk[ollama]")
        super().__init__(ans_client, agent_id, parent_agent_id, silent)
        self.base_client = base_client

    def chat(self, *args, **kwargs) -> Any:
        payload = self._chat_payload(kwargs)
        ph = hash_payload(payload)
        summary = f"Ollama: {kwargs.get('model', 'unknown')}"
        pre_id = _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent)

        start = time.time()
        outcome, outcome_summary, result, error = "success", "", None, None
        try:
            result = self.base_client.chat(*args, **kwargs)
            outcome_summary = self._chat_outcome(result)
        except Exception as e:
            outcome, outcome_summary, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, outcome_summary,
                   int((time.time() - start) * 1000), pre_id, self.silent)
        if error: raise error
        return result

    def generate(self, *args, **kwargs) -> Any:
        payload = self._generate_payload(kwargs)
        ph = hash_payload(payload)
        summary = f"Ollama: {kwargs.get('model', 'unknown')}"
        pre_id = _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent)

        start = time.time()
        outcome, outcome_summary, result, error = "success", "", None, None
        try:
            result = self.base_client.generate(*args, **kwargs)
            outcome_summary = self._generate_outcome(result)
        except Exception as e:
            outcome, outcome_summary, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, outcome_summary,
                   int((time.time() - start) * 1000), pre_id, self.silent)
        if error: raise error
        return result


class ANSAsyncOllamaClient(_BaseOllamaWrapper):
    """
    Async wrapper for Ollama AsyncClient with ANS tracing.

    Example:
        from ollama import AsyncClient
        from ans.integrations.ollama import ANSAsyncOllamaClient

        base = AsyncClient()
        ans = ANSClient()
        client = ANSAsyncOllamaClient(base, ans, agent_id="ans_xyz")

        response = await client.chat(model="llama3", messages=[...])
    """

    def __init__(self, base_client: AsyncClient, ans_client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
        if not _ollama_available:
            raise ImportError("Ollama is not installed. Install with: pip install ans-sdk[ollama]")
        super().__init__(ans_client, agent_id, parent_agent_id, silent)
        self.base_client = base_client

    async def chat(self, *args, **kwargs) -> Any:
        loop = asyncio.get_running_loop()
        payload = self._chat_payload(kwargs)
        ph = hash_payload(payload)
        summary = f"Ollama: {kwargs.get('model', 'unknown')}"
        pre_id = await loop.run_in_executor(None, lambda: _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent))

        start = time.time()
        outcome, outcome_summary, result, error = "success", "", None, None
        try:
            result = await self.base_client.chat(*args, **kwargs)
            outcome_summary = self._chat_outcome(result)
        except Exception as e:
            outcome, outcome_summary, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        await loop.run_in_executor(None, lambda: _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, outcome_summary,
                                   int((time.time() - start) * 1000), pre_id, self.silent))
        if error: raise error
        return result

    async def generate(self, *args, **kwargs) -> Any:
        loop = asyncio.get_running_loop()
        payload = self._generate_payload(kwargs)
        ph = hash_payload(payload)
        summary = f"Ollama: {kwargs.get('model', 'unknown')}"
        pre_id = await loop.run_in_executor(None, lambda: _sign_pre(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, self.silent))

        start = time.time()
        outcome, outcome_summary, result, error = "success", "", None, None
        try:
            result = await self.base_client.generate(*args, **kwargs)
            outcome_summary = self._generate_outcome(result)
        except Exception as e:
            outcome, outcome_summary, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        await loop.run_in_executor(None, lambda: _sign_post(self.ans_client, self.agent_id, self.parent_agent_id, ph, summary, outcome, outcome_summary,
                                   int((time.time() - start) * 1000), pre_id, self.silent))
        if error: raise error
        return result
