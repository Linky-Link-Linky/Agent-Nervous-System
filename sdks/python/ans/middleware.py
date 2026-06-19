"""Middleware utilities for wrapping agent functions with ANS tracing."""

import asyncio
import functools
import sys
import time
from typing import Callable, Optional

from .client import ANSClient, ANSError, hash_payload


class ANSMiddleware:
    """
    Middleware for wrapping agent tools/functions with pre/post receipt generation.
    Can be used as a base class or standalone wrapper.
    """

    def __init__(
        self,
        client: ANSClient,
        agent_id: str,
        parent_agent_id: str = "",
        silent: bool = True,
    ):
        self.client = client
        self.agent_id = agent_id
        self.parent_agent_id = parent_agent_id
        self.silent = silent

    def _pre_receipt(self, payload_hash: str, summary: str, action_type: str, policy_decision: str = "allow", auth_context: str = "") -> str:
        try:
            resp = self.client.sign_append(
                agent_id=self.agent_id,
                phase="pre",
                action_type=action_type,
                payload_hash=payload_hash,
                payload_summary=summary,
                policy_decision=policy_decision,
                auth_context=auth_context,
                parent_agent_id=self.parent_agent_id,
            )
            return resp["receipt_id"]
        except ANSError as e:
            if not self.silent:
                raise
            print(f"ans: pre-receipt error (non-fatal): {e}", file=sys.stderr)
            return ""

    def _post_receipt(self, payload_hash: str, summary: str, action_type: str, outcome: str, outcome_summary: str, duration_ms: int, pre_receipt_id: str) -> None:
        if not pre_receipt_id:
            return
        try:
            self.client.sign_append(
                agent_id=self.agent_id,
                phase="post",
                action_type=action_type,
                payload_hash=payload_hash,
                payload_summary=summary,
                outcome=outcome,
                outcome_summary=outcome_summary,
                duration_ms=duration_ms,
                pre_receipt_id=pre_receipt_id,
                parent_agent_id=self.parent_agent_id,
            )
        except ANSError as e:
            if not self.silent:
                raise
            print(f"ans: post-receipt error (non-fatal): {e}", file=sys.stderr)

    def wrap(
        self,
        func: Callable,
        action_type: str,
        policy_decision: str = "allow",
        auth_context: str = "",
        payload_summary_fn: Optional[Callable[..., str]] = None,
    ) -> Callable:
        """
        Wrap a function with ANS pre/post receipt generation.

        Args:
            func: Function to wrap
            action_type: Action type (e.g., "file.write")
            policy_decision: Policy decision for pre-action
            auth_context: Authorizing context for pre-action
            payload_summary_fn: Optional function to generate payload summary

        Returns:
            Wrapped function that generates receipts
        """

        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            if payload_summary_fn is not None:
                summary = payload_summary_fn(*args, **kwargs)
            else:
                summary = f"{func.__name__}()"

            payload = {"args": args, "kwargs": kwargs}
            payload_hash = hash_payload(payload)

            pre_receipt_id = self._pre_receipt(payload_hash, summary, action_type, policy_decision, auth_context)

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

            self._post_receipt(payload_hash, summary, action_type, outcome, outcome_summary, duration_ms, pre_receipt_id)

            if error is not None:
                raise error

            return result

        return wrapper

    async def wrap_async(
        self,
        func: Callable,
        action_type: str,
        policy_decision: str = "allow",
        auth_context: str = "",
        payload_summary_fn: Optional[Callable[..., str]] = None,
    ) -> Callable:
        """
        Wrap an async function with ANS pre/post receipt generation.

        Same as wrap() but for async functions.
        """
        loop = asyncio.get_running_loop()

        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            if payload_summary_fn is not None:
                summary = payload_summary_fn(*args, **kwargs)
            else:
                summary = f"{func.__name__}()"

            payload = {"args": args, "kwargs": kwargs}
            payload_hash = hash_payload(payload)

            pre_receipt_id = await loop.run_in_executor(
                None, lambda: self._pre_receipt(payload_hash, summary, action_type, policy_decision, auth_context)
            )

            start_time = time.time()
            outcome = "success"
            outcome_summary = ""
            result = None
            error = None

            try:
                result = await func(*args, **kwargs)
                outcome_summary = "Completed successfully"
            except Exception as e:
                outcome = "failure"
                outcome_summary = f"Error: {type(e).__name__}: {str(e)[:100]}"
                error = e

            duration_ms = int((time.time() - start_time) * 1000)

            await loop.run_in_executor(
                None, lambda: self._post_receipt(payload_hash, summary, action_type, outcome, outcome_summary, duration_ms, pre_receipt_id)
            )

            if error is not None:
                raise error

            return result

        return wrapper
