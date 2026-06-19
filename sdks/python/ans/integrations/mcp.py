"""Model Context Protocol (MCP) server integration for ANS."""

import asyncio
import functools
import inspect
import sys
import time
from typing import Callable

try:
    from mcp.server import Server
    _mcp_available = True
except ImportError:
    Server = None
    _mcp_available = False

from ..client import ANSClient, ANSError, hash_payload


def ans_mcp_middleware(client: ANSClient, agent_id: str, parent_agent_id: str = "", silent: bool = True):
    """
    Create middleware for MCP server tool handlers.

    Wraps tool handlers with ANS pre/post receipt generation.

    Args:
        client: ANS client
        agent_id: Agent ID
        parent_agent_id: Parent agent ID (if sub-agent)
        silent: If True (default), ANSError is logged but never raised.

    Returns:
        Decorator function to wrap tool handlers

    Example:
        from mcp.server import Server
        from ans.integrations.mcp import ans_mcp_middleware

        ans_client = ANSClient()
        ans_wrap = ans_mcp_middleware(ans_client, agent_id="ans_xyz")

        server = Server("my-server")

        @server.tool()
        @ans_wrap
        def read_file(path: str) -> str:
            with open(path) as f:
                return f.read()
    """

    def _pre(args, kwargs, p_hash, summary):
        try:
            resp = client.sign_append(
                agent_id=agent_id, phase="pre", action_type="agent.delegate",
                payload_hash=p_hash, payload_summary=summary,
                policy_decision="allow", parent_agent_id=parent_agent_id,
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not silent: raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post(p_hash, summary, outcome, osum, dur, pre_id):
        if not pre_id: return
        try:
            client.sign_append(
                agent_id=agent_id, phase="post", action_type="agent.delegate",
                payload_hash=p_hash, payload_summary=summary,
                outcome=outcome, outcome_summary=osum,
                duration_ms=dur, pre_receipt_id=pre_id,
                parent_agent_id=parent_agent_id,
            )
        except ANSError as e:
            if not silent: raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            payload = {"tool": func.__name__, "args": args, "kwargs": kwargs}
            ph = hash_payload(payload)
            summary = f"MCP tool: {func.__name__}"
            pre_id = _pre(args, kwargs, ph, summary)

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

        @functools.wraps(func)
        async def async_wrapper(*args, **kwargs):
            loop = asyncio.get_running_loop()
            payload = {"tool": func.__name__, "args": args, "kwargs": kwargs}
            ph = hash_payload(payload)
            summary = f"MCP tool: {func.__name__}"
            pre_id = await loop.run_in_executor(None, lambda: _pre(args, kwargs, ph, summary))

            start = time.time()
            outcome, o_sum, result, error = "success", "", None, None
            try:
                result = await func(*args, **kwargs)
                o_sum = str(result)[:120] if result else "Completed"
            except Exception as e:
                outcome, o_sum, error = "failure", f"Error: {type(e).__name__}: {str(e)[:100]}", e

            await loop.run_in_executor(None, lambda: _post(ph, summary, outcome, o_sum, int((time.time() - start) * 1000), pre_id))
            if error: raise error
            return result

        if inspect.iscoroutinefunction(func):
            return async_wrapper
        return wrapper

    return decorator
