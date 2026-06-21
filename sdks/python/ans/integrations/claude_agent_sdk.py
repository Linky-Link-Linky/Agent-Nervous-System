"""
ANS integration for the Anthropic Claude Agent SDK (claude-agent-sdk).

The Claude Agent SDK exposes PreToolUse and PostToolUse hook events that fire
before and after every tool call in the agent loop. ANS hooks into these events
to produce cryptographically signed receipts for every tool call Claude makes.

Usage:
    from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient
    from ans.integrations.claude_agent_sdk import ans_hooks

    options = ClaudeAgentOptions(hooks=ans_hooks(agent_id="ans_xyz123"))
    async with ClaudeSDKClient(options=options) as client:
        await client.query("Write a hello world program")
        async for msg in client.receive_response():
            print(msg)

Install:
    pip install claude-agent-sdk          # Anthropic's Claude Agent SDK
    pip install "ans-sdk"                 # ANS Python SDK

SPDX-License-Identifier: Apache-2.0
"""
from __future__ import annotations

import asyncio
import sys
import time
from typing import Any, Dict, Optional

from ans.client import ANSClient, ANSError, hash_payload

try:
    from claude_agent_sdk import HookMatcher
    _claude_agent_sdk_available = True
except ImportError:
    HookMatcher = None
    _claude_agent_sdk_available = False


def ans_hooks(
    agent_id: str,
    client: Optional[ANSClient] = None,
    silent: bool = True,
    parent_agent_id: str = "",
) -> dict:
    """
    Returns a hooks dict compatible with ClaudeAgentOptions(hooks=...).

    The dict registers two hook events:
      PreToolUse  — sends a pre-action ANS receipt before Claude executes a tool
      PostToolUse — sends a post-action ANS receipt after Claude executes a tool

    Pre/post receipts are linked via pre_receipt_id, stored in a dict keyed by
    tool_use_id (the second positional argument every hook callback receives).

    Args:
        agent_id: The ANS agent ID to attribute receipts to.
        client:   Optional pre-created Client. Defaults to Client() (auto-detect socket).
        silent:   If True (default), ANSError is logged but never raised.
        parent_agent_id: Parent agent ID (if sub-agent).

    Returns:
        A dict with "PreToolUse" and "PostToolUse" keys, each mapping to a list
        of HookMatcher objects, as expected by ClaudeAgentOptions.
    """
    _client = client or ANSClient()
    _pending: Dict[str, str] = {}
    _start: Dict[str, int] = {}
    _payloads: Dict[str, Dict] = {}

    async def pre_hook(
        input_data: Dict[str, Any],
        tool_use_id: Optional[str],
        context: Any,
    ) -> Dict[str, Any]:
        tool_name = input_data.get("tool_name", "unknown")
        tool_input = input_data.get("tool_input", {}) or {}
        key = tool_use_id or tool_name

        payload: Dict[str, Any] = {
            "tool": tool_name,
            **{k: str(v)[:200] for k, v in tool_input.items()},
        }
        ph = hash_payload(payload)
        _payloads[key] = payload
        summary = f"claude:{tool_name}"[:80]
        _start[key] = time.monotonic_ns()

        try:
            loop = asyncio.get_running_loop()
            resp = await loop.run_in_executor(None, lambda: _client.sign_append(
                agent_id=agent_id,
                phase="pre",
                action_type="custom",
                payload_hash=ph,
                payload_summary=summary,
                policy_decision="allow",
                auth_context="Claude Agent SDK PreToolUse hook",
                parent_agent_id=parent_agent_id,
            ))
            _pending[key] = resp.get("receipt_id", "")
        except ANSError as exc:
            if not silent:
                raise
            print(f"ans: pre-receipt error (non-fatal): {exc}", file=sys.stderr)
        return {}

    async def post_hook(
        input_data: Dict[str, Any],
        tool_use_id: Optional[str],
        context: Any,
    ) -> Dict[str, Any]:
        tool_name = input_data.get("tool_name", "unknown")
        key = tool_use_id or tool_name

        is_error = bool(input_data.get("is_error", False))
        outcome = "failure" if is_error else "success"
        tool_response = input_data.get("tool_response", None)
        o_sum = str(tool_response)[:120] if tool_response is not None else ""

        start_ns = _start.pop(key, None)
        duration_ms = int((time.monotonic_ns() - start_ns) / 1_000_000) if start_ns else 0
        pre_id = _pending.pop(key, "")

        # Use the same payload as pre-hook for consistent hash
        payload = _payloads.pop(key, {"tool": tool_name})
        ph = hash_payload(payload)

        try:
            loop = asyncio.get_running_loop()
            await loop.run_in_executor(None, lambda: _client.sign_append(
                agent_id=agent_id,
                phase="post",
                action_type="custom",
                payload_hash=ph,
                payload_summary=f"claude:{tool_name}"[:80],
                outcome=outcome,
                outcome_summary=o_sum,
                duration_ms=duration_ms,
                pre_receipt_id=pre_id,
                parent_agent_id=parent_agent_id,
            ))
        except ANSError as exc:
            if not silent:
                raise
            print(f"ans: post-receipt error (non-fatal): {exc}", file=sys.stderr)
        return {}

    return {
        "PreToolUse": [HookMatcher(matcher=None, hooks=[pre_hook])],
        "PostToolUse": [HookMatcher(matcher=None, hooks=[post_hook])],
    }
