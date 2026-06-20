"""MCP middleware — re-exported from ans.middleware for backward compatibility.

Usage:
    from ans.mcp import ANSMiddleware
"""
from .middleware import ANSMiddleware  # noqa: F401

__all__ = ["ANSMiddleware"]
