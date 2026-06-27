"""Reusable widgets for the ANS TUI."""

from datetime import datetime
from typing import List, Dict, Any

from textual.app import ComposeResult
from textual.containers import Container, Vertical
from textual.widgets import Static, DataTable


class ReceiptTable(DataTable):
    """Table widget for displaying receipts."""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.cursor_type = "row"
        self.zebra_stripes = True

    def populate(self, receipts: List[Dict[str, Any]]) -> None:
        """Populate the table with receipt data."""
        self.clear(columns=True)

        # Add columns
        self.add_column("Index", width=8)
        self.add_column("Phase", width=6)
        self.add_column("Agent ID", width=16)
        self.add_column("Action", width=20)
        self.add_column("Status", width=12)
        self.add_column("Time", width=20)

        # Add rows
        for receipt in receipts:
            index = receipt.get("chain_index", "?")
            phase = receipt.get("phase", "?")
            agent_id = receipt.get("agent_id", "?")[:14] + "..."
            action = receipt.get("action_type", "?")
            
            # Status depends on phase
            if phase == "pre":
                status = receipt.get("policy_decision", "?")
            else:
                status = receipt.get("outcome", "?")
            
            # Format timestamp
            ts_ns = receipt.get("timestamp_ns", 0)
            if ts_ns > 0:
                dt = datetime.fromtimestamp(ts_ns / 1e9)
                time_str = dt.strftime("%Y-%m-%d %H:%M:%S")
            else:
                time_str = "?"

            self.add_row(index, phase, agent_id, action, status, time_str)


class AgentTable(DataTable):
    """Table widget for displaying agents."""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.cursor_type = "row"
        self.zebra_stripes = True

    def populate(self, agents: List[Dict[str, Any]]) -> None:
        """Populate the table with agent data."""
        self.clear(columns=True)

        # Add columns
        self.add_column("Agent ID", width=20)
        self.add_column("Name", width=25)
        self.add_column("Version", width=12)
        self.add_column("Owner", width=20)
        self.add_column("Receipts", width=10)

        # Add rows
        for agent in agents:
            agent_id = agent.get("agent_id", "?")
            name = agent.get("name", "?")
            version = agent.get("version", "?")
            owner = agent.get("owner", "-")
            receipt_count = agent.get("receipt_count", 0)

            self.add_row(agent_id, name, version, owner, str(receipt_count))


class StatusPanel(Container):
    """Panel widget for displaying status information."""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.border_title = "Status"

    def compose(self) -> ComposeResult:
        """Compose the status panel."""
        yield Static("Loading...", id="status_content")

    def update_status(self, status: Dict[str, Any]) -> None:
        """Update the status display."""
        uptime = status.get("uptime", "?")
        chain_length = status.get("chain_length", 0)
        total_receipts = status.get("total_receipts", 0)
        total_agents = status.get("total_agents", 0)
        db_size = status.get("db_size_bytes", 0)
        db_size_mb = db_size / (1024 * 1024)

        content = f"""
[bold]Daemon Status[/bold]

Uptime: {uptime}
Chain Length: {chain_length}
Total Receipts: {total_receipts}
Total Agents: {total_agents}
Database Size: {db_size_mb:.2f} MB
"""
        self.query_one("#status_content", Static).update(content.strip())


class VerifyLog(Vertical):
    """Widget for displaying verification results."""

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.border_title = "Verification Log"

    def compose(self) -> ComposeResult:
        """Compose the verification log."""
        yield Static("No verification performed yet.", id="verify_log_content")

    def add_entry(self, message: str, level: str = "info") -> None:
        """Add a verification log entry."""
        content_widget = self.query_one("#verify_log_content", Static)
        current = content_widget.content
        
        # Color based on level
        if level == "success":
            colored_msg = f"[green]✓[/green] {message}"
        elif level == "error":
            colored_msg = f"[red]✗[/red] {message}"
        elif level == "warning":
            colored_msg = f"[yellow]⚠[/yellow] {message}"
        else:
            colored_msg = f"[blue]ℹ[/blue] {message}"
        
        # Append to existing content
        if current == "No verification performed yet.":
            new_content = colored_msg
        else:
            new_content = f"{current}\n{colored_msg}"
        
        content_widget.update(new_content)

    def clear(self) -> None:
        """Clear the verification log."""
        self.query_one("#verify_log_content", Static).update("No verification performed yet.")
