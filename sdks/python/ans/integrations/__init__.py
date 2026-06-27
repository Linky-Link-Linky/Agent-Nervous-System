"""
Framework integrations for ANS — True Protocol Neutrality.

Every integration provides the same pre/post receipt pattern, same error handling,
and same protocol semantics regardless of provider.

Import integration modules directly (e.g. ``from ans.integrations.ollama import ANSOllamaClient``)
so optional dependencies never raise at package-import time.
"""

__all__ = [
    "ANSAnthropicClient",
    "ANSAsyncOllamaClient",
    "ANSGenAIClient",
    "ANSGraph",
    "ANSOllamaClient",
    "ANSOpenAICompatClient",
    "ANSCallbackHandler",
    "ANSTool",
    "ans_mcp_middleware",
    "ans_node",
    "ans_tool_plugin",
    "claude_agent_hooks",
    "pydantic_ai_hooks",
]


def __getattr__(name: str):
    import importlib
    if name in __all__:
        module_map = {
            "ANSAnthropicClient": ".anthropic_sdk",
            "ANSAsyncOllamaClient": ".ollama",
            "ANSGenAIClient": ".google_genai",
            "ANSGraph": ".langgraph",
            "ANSOllamaClient": ".ollama",
            "ANSOpenAICompatClient": ".openai_compat",
            "ANSCallbackHandler": ".langchain",
            "ANSTool": ".crewai",
            "ans_mcp_middleware": ".mcp",
            "ans_node": ".langgraph",
            "ans_tool_plugin": ".openai_agents",
            "claude_agent_hooks": (".claude_agent_sdk", "ans_hooks"),
            "pydantic_ai_hooks": (".pydantic_ai", "ans_hooks"),
        }
        entry = module_map[name]
        if isinstance(entry, tuple):
            mod = importlib.import_module(entry[0], __package__)
            return getattr(mod, entry[1])
        mod = importlib.import_module(entry, __package__)
        return getattr(mod, name)
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
