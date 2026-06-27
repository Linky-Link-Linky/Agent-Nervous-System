"""Screen definitions for the ANS TUI."""

from textual.app import ComposeResult
from textual.containers import Container, Horizontal
from textual.screen import Screen
from textual.widgets import Button, Input, Label

from ..client import ANSError
from .widgets import ReceiptTable, AgentTable, StatusPanel, VerifyLog


class ChainScreen(Screen):
    """Screen for viewing the receipt chain."""

    BINDINGS = [
        ("r", "refresh", "Refresh"),
    ]

    @property
    def client(self):
        return self.app.ans_client  # type: ignore[attr-defined]

    def compose(self) -> ComposeResult:
        """Compose the chain screen."""
        with Container():
            yield Label("Receipt Chain", id="chain_title")
            with Horizontal():
                yield Button("Refresh", id="refresh_btn", variant="primary")
                yield Button("Verify Chain", id="verify_chain_btn", variant="success")
            yield ReceiptTable(id="receipt_table")
            yield StatusPanel(id="status_panel")

    def on_mount(self) -> None:
        """Load receipts on mount."""
        self.refresh()

    def refresh(self) -> None:
        """Refresh the receipt table."""
        try:
            # Query recent receipts
            receipts = self.client.query(limit=100)
            table = self.query_one("#receipt_table", ReceiptTable)
            table.populate(receipts)

            # Update status
            status = self.client.status()
            panel = self.query_one("#status_panel", StatusPanel)
            panel.update_status(status)

        except ANSError as e:
            self.notify(f"Error: {e}", severity="error")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button presses."""
        if event.button.id == "refresh_btn":
            self.refresh()
        elif event.button.id == "verify_chain_btn":
            self.app.push_screen("verify")

    def action_refresh(self) -> None:
        """Refresh action."""
        self.refresh()


class AgentsScreen(Screen):
    """Screen for viewing registered agents."""

    BINDINGS = [
        ("r", "refresh", "Refresh"),
    ]

    @property
    def client(self):
        return self.app.ans_client  # type: ignore[attr-defined]

    def compose(self) -> ComposeResult:
        """Compose the agents screen."""
        with Container():
            yield Label("Registered Agents", id="agents_title")
            with Horizontal():
                yield Button("Refresh", id="refresh_btn", variant="primary")
            yield AgentTable(id="agent_table")
            yield StatusPanel(id="status_panel")

    def on_mount(self) -> None:
        """Load agents on mount."""
        self.refresh()

    def refresh(self) -> None:
        """Refresh the agents table."""
        try:
            # In a real implementation, we'd have a "list agents" daemon command
            # For now, we'll query unique agent IDs from receipts
            receipts = self.client.query(limit=1000)
            
            # Group by agent_id
            agents_map = {}
            for r in receipts:
                agent_id = r.get("agent_id", "?")
                if agent_id not in agents_map:
                    agents_map[agent_id] = {
                        "agent_id": agent_id,
                        "name": r.get("agent_name", "Unknown"),
                        "version": "?",
                        "owner": "?",
                        "receipt_count": 0,
                    }
                agents_map[agent_id]["receipt_count"] += 1

            agents = list(agents_map.values())
            table = self.query_one("#agent_table", AgentTable)
            table.populate(agents)

            # Update status
            status = self.client.status()
            panel = self.query_one("#status_panel", StatusPanel)
            panel.update_status(status)

        except ANSError as e:
            self.notify(f"Error: {e}", severity="error")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button presses."""
        if event.button.id == "refresh_btn":
            self.refresh()

    def action_refresh(self) -> None:
        """Refresh action."""
        self.refresh()


class VerifyScreen(Screen):
    """Screen for verifying receipts."""

    BINDINGS = [
        ("c", "verify_chain", "Verify Chain"),
        ("r", "clear_log", "Clear"),
    ]

    @property
    def client(self):
        return self.app.ans_client  # type: ignore[attr-defined]

    def compose(self) -> ComposeResult:
        """Compose the verify screen."""
        with Container():
            yield Label("Receipt Verification", id="verify_title")
            with Horizontal():
                yield Label("Receipt ID:")
                yield Input(placeholder="Enter receipt ID", id="receipt_id_input")
                yield Button("Verify", id="verify_btn", variant="success")
            with Horizontal():
                yield Button("Verify Entire Chain", id="verify_chain_btn", variant="primary")
                yield Button("Clear Log", id="clear_btn")
            yield VerifyLog(id="verify_log")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button presses."""
        if event.button.id == "verify_btn":
            receipt_id = self.query_one("#receipt_id_input", Input).value
            if receipt_id:
                self.verify_receipt(receipt_id)
        elif event.button.id == "verify_chain_btn":
            self.verify_chain()
        elif event.button.id == "clear_btn":
            self.clear_log()

    def verify_receipt(self, receipt_id: str) -> None:
        """Verify a single receipt."""
        log = self.query_one("#verify_log", VerifyLog)
        log.add_entry(f"Verifying receipt: {receipt_id}", "info")

        try:
            result = self.client.verify(receipt_id)
            if result.get("valid"):
                log.add_entry(f"Receipt {receipt_id} is VALID", "success")
                log.add_entry(f"  Agent: {result.get('agent_id')}", "info")
                log.add_entry(f"  Action: {result.get('action_type')}", "info")
                log.add_entry(f"  Phase: {result.get('phase')}", "info")
            else:
                error_msg = result.get("error", "Unknown error")
                log.add_entry(f"Receipt {receipt_id} is INVALID: {error_msg}", "error")
        except ANSError as e:
            log.add_entry(f"Error verifying receipt: {e}", "error")

    def verify_chain(self) -> None:
        """Verify the entire chain."""
        log = self.query_one("#verify_log", VerifyLog)
        log.add_entry("Starting full chain verification...", "info")

        try:
            # Query all receipts
            receipts = self.client.query(limit=10000)
            log.add_entry(f"Found {len(receipts)} receipts to verify", "info")

            valid_count = 0
            invalid_count = 0

            for i, receipt in enumerate(receipts):
                receipt_id = receipt.get("receipt_id")
                if not receipt_id:
                    continue

                result = self.client.verify(receipt_id)
                if result.get("valid"):
                    valid_count += 1
                else:
                    invalid_count += 1
                    log.add_entry(
                        f"Receipt {receipt_id} (#{i+1}) FAILED: {result.get('error')}",
                        "error"
                    )

            if invalid_count == 0:
                log.add_entry(
                    f"✓ Chain verification PASSED: {valid_count} receipts verified",
                    "success"
                )
            else:
                log.add_entry(
                    f"✗ Chain verification FAILED: {invalid_count}/{len(receipts)} invalid",
                    "error"
                )

        except ANSError as e:
            log.add_entry(f"Error during chain verification: {e}", "error")

    def clear_log(self) -> None:
        """Clear the verification log."""
        log = self.query_one("#verify_log", VerifyLog)
        log.clear()

    def action_verify_chain(self) -> None:
        """Verify chain action."""
        self.verify_chain()

    def action_clear_log(self) -> None:
        """Clear log action."""
        self.clear_log()
