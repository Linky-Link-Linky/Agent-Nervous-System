"""
ANS integration for the Anthropic Python SDK (pip install anthropic).

This module provides a wrapper around the `anthropic` SDK's tool-use loop.
It wraps `client.messages.create()` to intercept tool calls in the response,
producing pre+post ANS receipts for every tool_use block Claude returns.

This targets the Messages API tool-use pattern (not the Agent SDK).
Use claude_agent_sdk.py for the higher-level Agent SDK hooks instead.

Usage:
    import anthropic
    from ans.integrations.anthropic_sdk import ANSAnthropicClient

    base = anthropic.Anthropic(api_key="...")
    ans_client = ANSAnthropicClient(base, agent_id="ans_xyz123")

    # Use ans_client.messages.create() exactly like anthropic.Anthropic().messages.create()
    # Every tool_use block in the response will produce a pre+post ANS receipt.
    response = ans_client.messages.create(
        model="claude-opus-4-6",
        max_tokens=1024,
        tools=[...],
        messages=[{"role": "user", "content": "..."}],
    )

SPDX-License-Identifier: MIT
"""
from __future__ import annotations

import sys
import time
from typing import Any, Optional

from ans.client import ANSClient, ANSError, hash_payload

try:
    import anthropic as _anthropic
    _anthropic_available = True
except ImportError:
    _anthropic = None
    _anthropic_available = False


class _ANSMessages:
    """Wraps anthropic.Anthropic().messages to intercept tool_use blocks."""

    def __init__(self, messages: Any, agent_id: str, client: ANSClient, parent_agent_id: str, silent: bool) -> None:
        self._messages = messages
        self._agent_id = agent_id
        self._client = client
        self._parent_agent_id = parent_agent_id
        self._silent = silent

    def create(self, **kwargs: Any) -> Any:
        start = time.time()
        response = self._messages.create(**kwargs)
        duration_ms = int((time.time() - start) * 1000)
        for block in getattr(response, "content", []):
            if getattr(block, "type", None) == "tool_use":
                tool_name = getattr(block, "name", "unknown")
                tool_input = getattr(block, "input", {})
                tool_id = getattr(block, "id", tool_name)
                payload = {"tool": tool_name,
                           **{k: str(v)[:200] for k, v in (tool_input or {}).items()}}
                ph = hash_payload(payload)
                summary = f"anthropic:{tool_name}"[:80]

                pre_id = ""
                try:
                    resp = self._client.sign_append(
                        agent_id=self._agent_id,
                        phase="pre",
                        action_type="custom",
                        payload_hash=ph,
                        payload_summary=summary,
                        policy_decision="allow",
                        auth_context=f"anthropic.messages.create tool_use block id={tool_id}",
                        parent_agent_id=self._parent_agent_id,
                    )
                    pre_id = resp.get("receipt_id", "")
                except ANSError as exc:
                    if not self._silent:
                        raise
                    print(f"ans: pre-receipt error: {exc}", file=sys.stderr)

                try:
                    self._client.sign_append(
                        agent_id=self._agent_id,
                        phase="post",
                        action_type="custom",
                        payload_hash=ph,
                        payload_summary=summary,
                        outcome="success",
                        outcome_summary="tool_use block returned by model, execution pending",
                        duration_ms=duration_ms,
                        pre_receipt_id=pre_id,
                        parent_agent_id=self._parent_agent_id,
                    )
                except ANSError as exc:
                    if not self._silent:
                        raise
                    print(f"ans: post-receipt error: {exc}", file=sys.stderr)
        return response


class ANSAnthropicClient:
    """
    Drop-in wrapper around anthropic.Anthropic() that intercepts tool_use blocks
    and produces ANS receipts for each one.

    Delegates all other attributes to the underlying client.
    """

    def __init__(
        self,
        anthropic_client: Any,
        agent_id: str,
        ans_client: Optional[ANSClient] = None,
        silent: bool = True,
        parent_agent_id: str = "",
    ) -> None:
        if not _anthropic_available:
            raise ImportError("The Anthropic SDK is not installed. Install with: pip install anthropic")
        self._inner = anthropic_client
        self._agent_id = agent_id
        self._ans_client = ans_client or ANSClient()
        self._silent = silent
        self._parent_agent_id = parent_agent_id
        self.messages = _ANSMessages(
            anthropic_client.messages,
            agent_id=agent_id,
            client=self._ans_client,
            parent_agent_id=parent_agent_id,
            silent=silent,
        )

    def __getattr__(self, name: str) -> Any:
        return getattr(self._inner, name)
