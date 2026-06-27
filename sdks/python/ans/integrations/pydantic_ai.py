"""PydanticAI integration for ANS (supports 20+ providers)."""

import asyncio
import sys
import time
from typing import Any, Optional

try:
    from pydantic_ai import Agent
    _pydantic_ai_available = True
except ImportError:
    Agent = None
    _pydantic_ai_available = False

from ..client import ANSClient, ANSError, hash_payload


def ans_hooks(client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
    """
    Create ANS capability hooks for PydanticAI agents.

    PydanticAI supports 20+ providers (OpenAI, Anthropic, Google, Groq, Mistral,
    Cohere, Gemini, DeepSeek, Ollama, etc.) through a unified interface.
    These hooks work with ALL of them — True Protocol Neutrality.

    Args:
        client: ANS client
        agent_id: Agent ID
        parent_agent_id: Parent agent ID (if sub-agent)
        silent: If True (default), ANSError is logged but never raised.

    Returns:
        Capability object to add to Agent

    Example:
        from pydantic_ai import Agent
        from ans.integrations.pydantic_ai import ans_hooks

        ans_client = ANSClient()
        hooks = ans_hooks(ans_client, agent_id="ans_xyz")

        agent = Agent("openai:gpt-4", capabilities=[hooks])
        result = agent.run_sync("Hello")
    """

    class ANSCapability:
        def __init__(self):
            self._pre_receipts = {}
            self._start_times = {}
            self._counter = 0

        async def on_tool_start(self, tool_name: str, tool_input: Any) -> None:
            payload = {"tool": tool_name, "input": tool_input}
            ph = hash_payload(payload)
            self._counter += 1
            key = f"{ph}:{self._counter}"
            summary = f"Tool: {tool_name}"

            self._start_times[key] = time.time()
            loop = asyncio.get_running_loop()
            try:
                resp = await loop.run_in_executor(None, lambda: client.sign_append(
                    agent_id=agent_id, phase="pre", action_type="agent.delegate",
                    payload_hash=ph, payload_summary=summary,
                    policy_decision="allow", parent_agent_id=parent_agent_id,
                ))
                self._pre_receipts[key] = resp["receipt_id"]
            except ANSError as e:
                if not silent:
                    raise
                print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)

        async def on_tool_end(
            self, tool_name: str, tool_input: Any, tool_output: Any, error: Optional[Exception]
        ) -> None:
            payload = {"tool": tool_name, "input": tool_input}
            ph = hash_payload(payload)
            self._counter += 1
            key = f"{ph}:{self._counter}"
            summary = f"Tool: {tool_name}"
            pre_id = self._pre_receipts.pop(key, "")
            start = self._start_times.pop(key, 0)

            if not pre_id:
                # Key mismatch — try to match by payload hash prefix (last resort)
                for k in list(self._pre_receipts.keys()):
                    if k.startswith(ph):
                        pre_id = self._pre_receipts.pop(k, "")
                        start = self._start_times.pop(k, 0)
                        break
                if not pre_id:
                    return

            duration_ms = int((time.time() - start) * 1000) if start else 0

            if error is not None:
                outcome, o_sum = "failure", f"Error: {type(error).__name__}: {str(error)[:100]}"
            else:
                outcome, o_sum = "success", (str(tool_output)[:120] if tool_output else "Completed")

            loop = asyncio.get_running_loop()
            try:
                await loop.run_in_executor(None, lambda: client.sign_append(
                    agent_id=agent_id, phase="post", action_type="agent.delegate",
                    payload_hash=ph, payload_summary=summary,
                    outcome=outcome, outcome_summary=o_sum,
                    duration_ms=duration_ms, pre_receipt_id=pre_id,
                    parent_agent_id=parent_agent_id,
                ))
            except ANSError as e:
                if not silent:
                    raise
                print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    return ANSCapability()
