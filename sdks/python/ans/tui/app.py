"""ANS Textual TUI — interactive terminal UI for the Agent Nervous System.
SPDX-License-Identifier: Apache-2.0"""
from __future__ import annotations

import os

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Container
from textual.screen import Screen
from textual.widgets import Footer, Header, Static

from ..client import ANSClient, ANSError
from .screens import ChainScreen, AgentsScreen, VerifyScreen


class _DaemonOfflineScreen(Screen):
    """Shown when the daemon socket cannot be reached."""

    def compose(self) -> ComposeResult:
        yield Static(
            "[bold red]Daemon Offline[/bold red]\n\n"
            "The ANS daemon socket could not be reached.\n"
            "Start the daemon with:\n\n"
            "    ans start\n\n"
            "Then press [bold]r[/bold] to retry, or [bold]q[/bold] to quit."
        )

    BINDINGS = [
        Binding("r", "retry", "Retry"),
        Binding("q", "quit", "Quit"),
    ]

    def action_retry(self) -> None:
        app = self.app
        if isinstance(app, ANSApp):
            self.dismiss()
            app.switch_screen("chain")


class ANSApp(App):
    """Interactive TUI for viewing ANS receipt chains.

    Keyboard shortcuts:
      c — Chain view    a — Agents view    v — Verify view
      r — Refresh       q — Quit           Ctrl+T — Toggle theme
    """

    CSS = """
    Screen { background: $surface; }

    #status-bar {
        height: 3;
        dock: top;
        background: $panel;
        color: $text;
        padding: 0 1;
    }

    #receipt-datatable {
        height: 1fr;
    }

    #action-bar {
        height: 1;
        dock: bottom;
        background: $primary;
        color: $text;
    }

    StatusPanel {
        height: 3;
    }

    _DaemonOfflineScreen {
        align: center middle;
    }
    """

    TITLE = "ANS — Agent Nervous System"
    SUB_TITLE = "Cryptographic Audit Trail Viewer"

    BINDINGS = [
        Binding("c", "show_chain", "Chain"),
        Binding("a", "show_agents", "Agents"),
        Binding("v", "show_verify", "Verify"),
        Binding("r", "refresh", "Refresh"),
        Binding("q", "quit", "Quit"),
        Binding("ctrl+t", "toggle_dark", "Theme"),
    ]

    SCREENS = {
        "chain": ChainScreen,
        "agents": AgentsScreen,
        "verify": VerifyScreen,
        "offline": _DaemonOfflineScreen,
    }

    _socket_path: str | None = os.environ.get("ANS_SOCKET_PATH")

    def __init__(self, socket_path: str | None = None, **kwargs) -> None:
        super().__init__(**kwargs)
        if socket_path is not None:
            self._socket_path = socket_path
        self.ans_client = ANSClient(socket_path=self._socket_path)

    @property
    def socket_path(self) -> str:
        """Resolved socket path (platform-aware default)."""
        return self.ans_client.socket_path

    def compose(self) -> ComposeResult:
        yield Header()
        yield Container(id="main")
        yield Footer()

    def on_mount(self) -> None:
        try:
            self.ans_client.status()
        except ANSError:
            self.push_screen("offline")
            return
        self.push_screen("chain")

    def action_show_chain(self) -> None:
        self.switch_screen("chain")

    def action_show_agents(self) -> None:
        self.switch_screen("agents")

    def action_show_verify(self) -> None:
        self.switch_screen("verify")

    def action_refresh(self) -> None:
        screen = self.screen
        if hasattr(screen, "action_refresh"):
            screen.action_refresh()
        else:
            screen.refresh()


def main() -> None:
    ANSApp().run()
