"""OpenAI Agents SDK integration for ANS."""

import functools
import sys
import time
from typing import Callable

try:
    import openai  # noqa: F401
    _openai_agents_available = True
except ImportError:
    _openai_agents_available = False

from ..client import ANSClient, ANSError, hash_payload


def ans_tool_plugin(
    client: ANSClient,
    agent_id: str,
    parent_agent_id: str = "",
    action_type: str = "custom",
    silent: bool = True,
):
    """
    Decorator factory for OpenAI agent tools that adds ANS tracing.

    Args:
        client: ANS client
        agent_id: Agent ID
        parent_agent_id: Parent agent ID (if sub-agent)
        action_type: Action type for receipts
        silent: If True (default), ANSError is logged but never raised.

    Example:
        from ans.integrations.openai_agents import ans_tool_plugin
        from openai import OpenAI

        ans_client = ANSClient()
        plugin = ans_tool_plugin(ans_client, agent_id="ans_xyz")

        @plugin
        def write_file(path: str, content: str):
            with open(path, 'w') as f:
                f.write(content)
            return f"Wrote {len(content)} bytes"

        client = OpenAI()
        response = client.chat.completions.create(
            model="gpt-4",
            messages=[...],
            tools=[write_file]
        )
    """

    def _pre(ph, summary):
        try:
            resp = client.sign_append(
                agent_id=agent_id, phase="pre", action_type=action_type,
                payload_hash=ph, payload_summary=summary,
                policy_decision="allow", parent_agent_id=parent_agent_id,
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post(ph, summary, outcome, o_sum, dur, pre_id):
        if not pre_id: return
        try:
            client.sign_append(
                agent_id=agent_id, phase="post", action_type=action_type,
                payload_hash=ph, payload_summary=summary,
                outcome=outcome, outcome_summary=o_sum,
                duration_ms=dur, pre_receipt_id=pre_id,
                parent_agent_id=parent_agent_id,
            )
        except ANSError as e:
            if not silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            payload = {"args": args, "kwargs": kwargs}
            ph = hash_payload(payload)
            summary = f"Tool: {func.__name__}"
            pre_id = _pre(ph, summary)

            start = time.time()
            outcome, o_sum, result, error = "success", "", None, None
            try:
                result = func(*args, **kwargs)
                o_sum = str(result)[:120] if result else "Completed"
            except Exception as e:
                outcome, o_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

            _post(ph, summary, outcome, o_sum, int((time.time() - start) * 1000), pre_id)
            if error: raise error
            return result

        return wrapper

    return decorator
