"""LangGraph integration for ANS."""

from typing import Any, Callable

try:
    from langgraph.graph import StateGraph
    _langgraph_available = True
except ImportError:
    StateGraph = None
    _langgraph_available = False

from ..client import ANSClient
from ..middleware import ANSMiddleware


def ans_node(
    func: Callable,
    client: ANSClient,
    agent_id: str,
    parent_agent_id: str = "",
    action_type: str = "agent.delegate",
) -> Callable:
    """
    Wrap a LangGraph node function with ANS tracing.
    
    Args:
        func: Node function to wrap
        client: ANS client
        agent_id: Agent ID
        parent_agent_id: Parent agent ID (if sub-agent)
        action_type: Action type for receipts
    
    Example:
        from ans.integrations.langgraph import ans_node
        
        def my_node(state):
            # ... node logic
            return state
        
        traced_node = ans_node(my_node, client, agent_id="ans_xyz")
        
        graph = StateGraph(...)
        graph.add_node("my_node", traced_node)
    """
    middleware = ANSMiddleware(client, agent_id, parent_agent_id)

    return middleware.wrap(
        func,
        action_type=action_type,
        payload_summary_fn=lambda state: f"Node: {func.__name__}",
    )


class ANSGraph:
    """
    Wrapper for StateGraph that automatically traces all node executions.
    
    Example:
        from ans.integrations.langgraph import ANSGraph
        
        graph = ANSGraph(client, agent_id="ans_xyz")
        graph.add_node("node1", node1_func)
        graph.add_node("node2", node2_func)
        graph.add_edge("node1", "node2")
        
        app = graph.compile()
        app.invoke({"input": "..."})
    """

    def __init__(
        self,
        client: ANSClient,
        agent_id: str,
        parent_agent_id: str = "",
        state_schema: Any = None,
    ):
        if not _langgraph_available:
            raise ImportError("LangGraph is not installed. Install with: pip install ans-sdk[langgraph]")
        self.client = client
        self.agent_id = agent_id
        self.parent_agent_id = parent_agent_id
        self.graph = StateGraph(state_schema) if state_schema else StateGraph(dict)
        self.middleware = ANSMiddleware(client, agent_id, parent_agent_id)

    def add_node(self, name: str, func: Callable) -> None:
        """Add a node with automatic ANS tracing."""
        wrapped = self.middleware.wrap(
            func,
            action_type="agent.delegate",
            payload_summary_fn=lambda state: f"Node: {name}",
        )
        self.graph.add_node(name, wrapped)

    def add_edge(self, start: str, end: str) -> None:
        """Add an edge between nodes."""
        self.graph.add_edge(start, end)

    def add_conditional_edges(self, source: str, condition: Callable, mapping: dict) -> None:
        """Add conditional edges."""
        self.graph.add_conditional_edges(source, condition, mapping)

    def set_entry_point(self, node: str) -> None:
        """Set the entry point of the graph."""
        self.graph.set_entry_point(node)

    def set_finish_point(self, node: str) -> None:
        """Set the finish point of the graph."""
        self.graph.set_finish_point(node)

    def compile(self) -> Any:
        """Compile the graph."""
        return self.graph.compile()
