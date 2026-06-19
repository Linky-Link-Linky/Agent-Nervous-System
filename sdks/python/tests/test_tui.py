"""
ANS Textual TUI tests.

Uses Textual's built-in async test runner (app.run_test()).

SPDX-License-Identifier: MIT
"""
from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from ans.client import ANSClient
from ans.tui.app import ANSApp, _DaemonOfflineScreen
from ans.tui.widgets import StatusPanel, ReceiptTable


@pytest.mark.asyncio
async def test_tui_offline():
    """When daemon cannot connect, _DaemonOfflineScreen is pushed."""
    app = ANSApp(socket_path=r"\\.\pipe\nonexistent")
    async with app.run_test() as pilot:
        await pilot.pause()
        assert isinstance(app.screen, _DaemonOfflineScreen)


@pytest.mark.asyncio
async def test_status_panel():
    """StatusPanel displays mocked daemon status."""
    mock_client = MagicMock(spec=ANSClient)
    mock_client.status.return_value = {
        "uptime": "1h23m",
        "chain_length": 100,
        "total_receipts": 50,
        "total_agents": 3,
        "db_size_bytes": 1024000,
        "started_at": "2025-01-01T00:00:00Z",
    }

    app = ANSApp(socket_path=r"\\.\pipe\nonexistent")
    async with app.run_test() as pilot:
        panel = StatusPanel(id="test-status")
        await app.mount(panel)
        await pilot.pause()
        # After populate/update, the Static widget should exist
        content = panel.query_one("#status_content")
        assert content is not None
        # Verify update_status works
        panel.update_status(mock_client.status())
        rendered = str(content.content)
        assert "Daemon Status" in rendered
        assert "1h23m" in rendered
        assert "100" in rendered


@pytest.mark.asyncio
async def test_receipt_table_populate():
    """ReceiptTable renders rows from receipt data."""
    mock_data = [
        {
            "chain_index": 1,
            "phase": "pre",
            "agent_id": "ans_testAgent1",
            "action_type": "file.write",
            "outcome": "",
            "timestamp_ns": 1700000000000000000,
        },
        {
            "chain_index": 2,
            "phase": "post",
            "agent_id": "ans_testAgent1",
            "action_type": "file.write",
            "outcome": "success",
            "duration_ms": 5,
            "timestamp_ns": 1700000000005000000,
        },
        {
            "chain_index": 3,
            "phase": "pre",
            "agent_id": "ans_testAgent2",
            "action_type": "agent.delegate",
            "outcome": "",
            "timestamp_ns": 1700000000010000000,
        },
    ]

    app = ANSApp(socket_path=r"\\.\pipe\nonexistent")
    async with app.run_test() as pilot:
        table = ReceiptTable(id="test-receipts")
        await app.mount(table)
        await pilot.pause()
        table.populate(mock_data)
        await pilot.pause()
        # DataTable should have 3 data rows (plus header)
        assert table.row_count == 3
