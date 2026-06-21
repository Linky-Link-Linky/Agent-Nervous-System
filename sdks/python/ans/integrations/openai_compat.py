"""
OpenAI-compatible API integration for ANS.

Works with ANY provider that exposes an OpenAI-compatible /chat/completions endpoint:

  - llama.cpp (local open-weight)   - http://localhost:8080/v1
  - vLLM (local open-weight)         - http://localhost:8000/v1
  - Ollama (OpenAI-compat mode)      - http://localhost:11434/v1
  - Together AI                      - https://api.together.xyz/v1
  - Groq                             - https://api.groq.com/openai/v1
  - DeepSeek                         - https://api.deepseek.com/v1
  - Mistral API                      - https://api.mistral.ai/v1
  - Fireworks AI                     - https://api.fireworks.ai/inference/v1
  - Perplexity                       - https://api.perplexity.ai
  - xAI (Grok)                       - https://api.x.ai/v1
  - Any OpenAI-compatible endpoint

Usage:
    from openai import OpenAI
    from ans.integrations.openai_compat import ANSOpenAICompatClient

    base = OpenAI(base_url="http://localhost:8080/v1", api_key="not-needed")
    ans_client = ANSClient()
    client = ANSOpenAICompatClient(base, ans_client, agent_id="ans_xyz")

    response = client.chat.completions.create(
        model="llama-3-8b",
        messages=[{"role": "user", "content": "Hello"}],
    )

SPDX-License-Identifier: MIT
"""
from __future__ import annotations

import asyncio
import sys
import time
from typing import Any

from ans.client import ANSClient, ANSError, hash_payload

try:
    from openai import OpenAI, AsyncOpenAI
    _openai_available = True
except ImportError:
    OpenAI = None
    AsyncOpenAI = None
    _openai_available = False


class _ANSCompletions:
    """Wraps chat.completions to intercept .create() calls."""

    def __init__(
        self,
        inner: Any,
        agent_id: str,
        ans_client: ANSClient,
        parent_agent_id: str,
        silent: bool,
    ) -> None:
        self._inner = inner
        self._agent_id = agent_id
        self._ans_client = ans_client
        self._parent_agent_id = parent_agent_id
        self._silent = silent

    def _pre(self, ph: str, summary: str, model: str) -> str:
        try:
            resp = self._ans_client.sign_append(
                agent_id=self._agent_id, phase="pre", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                policy_decision="allow", parent_agent_id=self._parent_agent_id,
                auth_context=f"openai-compat model={model}",
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not self._silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post(self, ph: str, summary: str, outcome: str, o_sum: str, dur: int, pre_id: str) -> None:
        if not pre_id: return
        try:
            self._ans_client.sign_append(
                agent_id=self._agent_id, phase="post", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                outcome=outcome, outcome_summary=o_sum,
                duration_ms=dur, pre_receipt_id=pre_id,
                parent_agent_id=self._parent_agent_id,
            )
        except ANSError as e:
            if not self._silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def create(self, **kwargs: Any) -> Any:
        model = kwargs.get("model", "unknown")
        messages = kwargs.get("messages", [])
        payload = {"model": model, "messages": self._truncate(messages)}
        ph = hash_payload(payload)
        summary = f"{model}"[:80]
        pre_id = self._pre(ph, summary, model)

        start = time.time()
        outcome, out_sum, result, error = "success", "", None, None
        try:
            result = self._inner.create(**kwargs)
            choice = result.choices[0] if result.choices else None
            if choice and hasattr(choice, "message") and choice.message.content:
                out_sum = choice.message.content[:120]
            else:
                out_sum = "Completed"
        except Exception as e:
            outcome, out_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        self._post(ph, summary, outcome, out_sum, int((time.time() - start) * 1000), pre_id)
        if error: raise error
        return result

    async def create_async(self, **kwargs: Any) -> Any:
        loop = asyncio.get_running_loop()
        model = kwargs.get("model", "unknown")
        messages = kwargs.get("messages", [])
        payload = {"model": model, "messages": self._truncate(messages)}
        ph = hash_payload(payload)
        summary = f"{model}"[:80]
        pre_id = await loop.run_in_executor(None, lambda: self._pre(ph, summary, model))

        start = time.time()
        outcome, out_sum, result, error = "success", "", None, None
        try:
            result = await self._inner.create(**kwargs)
            choice = result.choices[0] if result.choices else None
            if choice and hasattr(choice, "message") and choice.message.content:
                out_sum = choice.message.content[:120]
            else:
                out_sum = "Completed"
        except Exception as e:
            outcome, out_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        await loop.run_in_executor(None, lambda: self._post(ph, summary, outcome, out_sum, int((time.time() - start) * 1000), pre_id))
        if error: raise error
        return result

    @staticmethod
    def _truncate(messages: list) -> list:
        if not messages:
            return []
        truncated = []
        for m in messages[-4:]:
            c = m.get("content", "")
            truncated.append({"role": m.get("role", "user"), "content": c[:500] if isinstance(c, str) else str(c)[:500]})
        return truncated


class _AsyncANSCompletions:
    """Wraps AsyncOpenAI().chat.completions to intercept .create() calls."""

    def __init__(self, inner: Any, agent_id: str, ans_client: ANSClient, parent_agent_id: str, silent: bool):
        self._inner = inner
        self._agent_id = agent_id
        self._ans_client = ans_client
        self._parent_agent_id = parent_agent_id
        self._silent = silent

    async def create(self, **kwargs: Any) -> Any:
        loop = asyncio.get_running_loop()
        model = kwargs.get("model", "unknown")
        messages = kwargs.get("messages", [])
        payload = {"model": model, "messages": _ANSCompletions._truncate(messages)}
        ph = hash_payload(payload)
        summary = f"{model}"[:80]

        try:
            pre_id = await loop.run_in_executor(None, lambda: self._sign_pre(ph, summary, model))
        except ANSError as e:
            if not self._silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            pre_id = ""

        start = time.time()
        outcome, out_sum, result, error = "success", "", None, None
        try:
            result = await self._inner.create(**kwargs)
            choice = result.choices[0] if result.choices else None
            if choice and hasattr(choice, "message") and choice.message.content:
                out_sum = choice.message.content[:120]
            else:
                out_sum = "Completed"
        except Exception as e:
            outcome, out_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

        if pre_id:
            try:
                await loop.run_in_executor(None, lambda: self._sign_post(ph, summary, outcome, out_sum, int((time.time() - start) * 1000), pre_id))
            except ANSError as e:
                if not self._silent: raise
                print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)
        if error: raise error
        return result

    def _sign_pre(self, ph: str, summary: str, model: str) -> str:
        try:
            resp = self._ans_client.sign_append(
                agent_id=self._agent_id, phase="pre", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                policy_decision="allow", parent_agent_id=self._parent_agent_id,
                auth_context=f"openai-compat model={model}",
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not self._silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _sign_post(self, ph: str, summary: str, outcome: str, o_sum: str, dur: int, pre_id: str) -> None:
        if not pre_id: return
        try:
            self._ans_client.sign_append(
                agent_id=self._agent_id, phase="post", action_type="agent.delegate",
                payload_hash=ph, payload_summary=summary,
                outcome=outcome, outcome_summary=o_sum,
                duration_ms=dur, pre_receipt_id=pre_id,
                parent_agent_id=self._parent_agent_id,
            )
        except ANSError as e:
            if not self._silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)


class ANSOpenAICompatClient:
    """
    Wraps any OpenAI-compatible client (sync or async) with ANS tracing.

    Supports: llama.cpp, vLLM, Together AI, Groq, DeepSeek, Mistral API,
    Fireworks, Perplexity, xAI, and any OpenAI-compatible endpoint.

    Example:
        from openai import OpenAI
        from ans.integrations.openai_compat import ANSOpenAICompatClient

        base = OpenAI(base_url="http://localhost:8080/v1", api_key="not-needed")
        client = ANSOpenAICompatClient(base, ANSClient(), agent_id="ans_xyz")

        resp = client.chat.completions.create(
            model="llama-3-8b",
            messages=[{"role": "user", "content": "Hello"}],
        )
        print(resp.choices[0].message.content)
    """

    def __init__(
        self,
        client: Any,
        ans_client: ANSClient,
        agent_id: str,
        parent_agent_id: str = "",
        silent: bool = True,
    ) -> None:
        if not _openai_available:
            raise ImportError("The OpenAI SDK is not installed. Install with: pip install openai")
        self._inner = client
        self._ans_client = ans_client
        self._agent_id = agent_id
        self._parent_agent_id = parent_agent_id
        self._silent = silent

        is_async = isinstance(client, AsyncOpenAI) if AsyncOpenAI is not None else False
        inner_completions = client.chat.completions

        if is_async:
            self.chat = _AsyncChatNamespace(inner_completions, agent_id, ans_client, parent_agent_id, silent)
        else:
            self.chat = _ChatNamespace(inner_completions, agent_id, ans_client, parent_agent_id, silent)

    def __getattr__(self, name: str) -> Any:
        return getattr(self._inner, name)


class _ChatNamespace:
    def __init__(self, completions: Any, agent_id: str, ans_client: ANSClient, parent_agent_id: str, silent: bool):
        self.completions = _ANSCompletions(completions, agent_id, ans_client, parent_agent_id, silent)

    def __getattr__(self, name: str) -> Any:
        return getattr(self.completions, name)


class _AsyncChatNamespace:
    def __init__(self, completions: Any, agent_id: str, ans_client: ANSClient, parent_agent_id: str, silent: bool):
        self.completions = _AsyncANSCompletions(completions, agent_id, ans_client, parent_agent_id, silent)

    def __getattr__(self, name: str) -> Any:
        return getattr(self.completions, name)
