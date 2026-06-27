"""Python decorator for automatic tracing of functions."""

import functools
import sys
import time
from typing import Callable, Optional

from . import client as _client_mod
from .client import ANSError, get_client, hash_payload


def trace(
    action_type: str,
    agent_id: Optional[str] = None,
    policy_decision: str = "allow",
    auth_context: str = "",
    payload_summary_fn: Optional[Callable[..., str]] = None,
    parent_agent_id: str = "",
    silent: bool = True,
    snapshot: bool = False,
):
    """
    Decorator that wraps a function with ANS pre/post receipt generation.

    Args:
        action_type: Action type (e.g., "file.write", "http.get")
        policy_decision: Policy decision for pre-action ("allow", "deny", "allow_with_conditions")
        auth_context: Authorizing context for pre-action
        payload_summary_fn: Optional function that takes the same args as the decorated function
                           and returns a human-readable summary string
        parent_agent_id: Parent agent ID (if sub-agent)
        silent: If True (default), ANSError is logged but never raised.

    Example:
        import os

        @ans.trace(action_type="file.write")
        def write_file(path: str, content: str):
            safe_path = os.path.realpath(path)
            with open(safe_path, 'w') as f:
                f.write(content)
    """

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            aid = agent_id or _client_mod._default_agent_id
            if aid is None:
                raise ANSError("ANS not configured. Call ans.configure(agent_id='...') first.")

            client = get_client()

            if payload_summary_fn is not None:
                summary = payload_summary_fn(*args, **kwargs)
            else:
                summary = f"{func.__name__}()"

            payload = {"args": args, "kwargs": kwargs}
            ph = hash_payload(payload)

            if snapshot:
                try:
                    client.take_snapshot(agent_id=aid, snap_type="filesystem")
                except ANSError as e:
                    if not silent:
                        raise
                    print(f"ans: snapshot error (non-fatal): {e}", file=sys.stderr)

            pre_receipt_id = ""
            try:
                pre_resp = client.sign_append(
                    agent_id=aid, phase="pre",
                    action_type=action_type, payload_hash=ph,
                    payload_summary=summary, policy_decision=policy_decision,
                    auth_context=auth_context, parent_agent_id=parent_agent_id,
                )
                pre_receipt_id = pre_resp["receipt_id"]
            except ANSError as e:
                if not silent:
                    raise
                print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)

            start_time = time.time()
            outcome = "success"
            outcome_summary = ""
            result = None
            error = None

            try:
                result = func(*args, **kwargs)
                outcome_summary = "Completed successfully"
            except Exception as e:
                outcome = "failure"
                outcome_summary = f"Error: {type(e).__name__}: {str(e)[:100]}"
                error = e

            duration_ms = int((time.time() - start_time) * 1000)

            if pre_receipt_id:
                try:
                    client.sign_append(
                        agent_id=aid, phase="post",
                        action_type=action_type, payload_hash=ph,
                        payload_summary=summary, outcome=outcome,
                        outcome_summary=outcome_summary, duration_ms=duration_ms,
                        pre_receipt_id=pre_receipt_id, parent_agent_id=parent_agent_id,
                    )
                except ANSError as e:
                    if not silent:
                        raise
                    print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

            if error is not None:
                raise error

            return result

        return wrapper

    return decorator
