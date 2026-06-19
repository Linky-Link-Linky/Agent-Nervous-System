"""
Universal model wrapper — True Protocol Neutrality for ANS.

Wrap ANY model client with ANS tracing using a single function.

Auto-detects the provider type and returns a wrapped proxy with the same
interface. Pre/post cryptographic receipts are generated for every API call.

Supported providers (auto-detected):
  - OpenAI / OpenAI-compatible (llama.cpp, vLLM, Together, Groq, DeepSeek, Mistral, etc.)
  - Anthropic (Messages API)
  - Google Generative AI (Gemini)
  - Ollama (local open-weight models)
  - PydanticAI (20+ providers through unified interface)

Usage:
    from openai import OpenAI
    from ans import wrap_model, ANSClient

    raw = OpenAI(api_key="...")
    client = ANSClient()
    wrapped = wrap_model(raw, client, agent_id="ans_xyz")

    # Use wrapped exactly as you would use raw — same API, same types
    resp = wrapped.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello"}],
    )

With Ollama (local):
    from ollama import Client
    raw = Client()
    wrapped = wrap_model(raw, client, agent_id="ans_xyz")
    resp = wrapped.chat(model="llama3", messages=[...])

With Anthropic:
    import anthropic
    raw = anthropic.Anthropic(api_key="...")
    wrapped = wrap_model(raw, client, agent_id="ans_xyz")
    resp = wrapped.messages.create(model="claude-opus-4", ...)

With Google Gemini:
    import google.generativeai as genai
    genai.configure(api_key="...")
    raw = genai.GenerativeModel("gemini-2.0-flash")
    wrapped = wrap_model(raw, client, agent_id="ans_xyz")
    resp = wrapped.generate_content("Hello")

SPDX-License-Identifier: MIT
"""
from __future__ import annotations

from typing import Any, Optional

from .client import ANSClient


def _detect_provider(model: Any) -> str:
    """Auto-detect the model provider type."""
    modname = type(model).__module__

    if "anthropic" in modname:
        return "anthropic"
    if "ollama" in modname:
        return "ollama"
    if "google.generativeai" in modname or "genai" in modname:
        return "google_genai"
    if "openai" in modname:
        return "openai"
    if "pydantic_ai" in modname:
        return "pydantic_ai"
    if hasattr(model, "chat") and hasattr(model.chat, "completions"):
        return "openai"
    if hasattr(model, "messages") and hasattr(model.messages, "create"):
        return "anthropic"
    if hasattr(model, "generate_content") or hasattr(model, "generate_content_async"):
        return "google_genai"

    return "unknown"


def wrap_model(
    model: Any,
    ans_client: ANSClient,
    agent_id: str,
    parent_agent_id: str = "",
    silent: bool = True,
) -> Any:
    """
    Wrap any model client with ANS tracing.

    Args:
        model: The model client object to wrap.
        ans_client: An ANS client connected to the daemon.
        agent_id: ANS agent ID for receipt attribution.
        parent_agent_id: Parent agent ID (if sub-agent).
        silent: If True (default), receipt errors are logged but never raised.

    Returns:
        A wrapped proxy with the same interface as the input model.

    Raises:
        ValueError: If the provider type cannot be determined.
    """
    provider = _detect_provider(model)

    if provider == "openai":
        from .integrations.openai_compat import ANSOpenAICompatClient
        return ANSOpenAICompatClient(
            model, ans_client, agent_id=agent_id,
            parent_agent_id=parent_agent_id, silent=silent,
        )

    if provider == "anthropic":
        from .integrations.anthropic_sdk import ANSAnthropicClient
        return ANSAnthropicClient(
            model, agent_id=agent_id, ans_client=ans_client,
            silent=silent, parent_agent_id=parent_agent_id,
        )

    if provider == "ollama":
        from .integrations.ollama import ANSOllamaClient, ANSAsyncOllamaClient
        modname = type(model).__module__
        if "AsyncClient" in type(model).__name__:
            return ANSAsyncOllamaClient(
                model, ans_client, agent_id=agent_id,
                parent_agent_id=parent_agent_id,
            )
        return ANSOllamaClient(
            model, ans_client, agent_id=agent_id,
            parent_agent_id=parent_agent_id,
        )

    if provider == "google_genai":
        from .integrations.google_genai import ANSGenAIClient
        return ANSGenAIClient(
            model, ans_client, agent_id=agent_id,
            parent_agent_id=parent_agent_id, silent=silent,
        )

    raise ValueError(
        f"Cannot determine provider for model type: {type(model).__name__} "
        f"(module: {type(model).__module__}). "
        f"Supported: OpenAI-compatible, Anthropic, Ollama, Google GenAI. "
        f"Try using a provider-specific integration directly."
    )
