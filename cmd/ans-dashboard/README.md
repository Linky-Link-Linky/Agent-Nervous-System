# ANS Dashboard

Full-screen terminal dashboard for the Agent Nervous System daemon. Built with Go, `tview`, and `tcell`.

## Quickstart

From the `ans/` directory:

```
go run ./cmd/ans-dashboard/
```

Or build a static binary:

```
go build -o ans-dashboard.exe ./cmd/ans-dashboard/
./ans-dashboard.exe
```

## Layout

```
┌────────────────────────────────────────────────────────────────────┐
│  ■ ■ ■   ■   ■ ■ ■   ■ ■ ■ ■ ■ ■ ■       AGENT NERVOUS SYSTEM   │
├──────────────────────┬─────────────────────────────────────────────┤
│  RESOURCE OVERVIEW   │  [ GPU's: 24 ]  [ Units: 130 ]             │
│                      │  [ A100s: 15 ]  [ H100s: 7  ]              │
├──────────────────────┴─────────────────────────────────────────────┤
│  BREAKDOWN  [ 1 day ▾ ]  [ Type: all ▾ ]                          │
│  ● audit-trail   ● snapshot-engine   ● mcp-proxy                  │
│  ● policy-engine   ● identity-broker                               │
│  [stacked bar chart — 24 columns of hourly resource usage]         │
├─────────────────────────────────┬──────────────────────────────────┤
│  AUDIT TRAIL (live feed)        │  POLICY ENGINE                   │
│  [scrolling event log]          │  [active rules + violations]     │
├─────────────────────────────────┴──────────────────────────────────┤
│  UPTIME: 4h 22m  |  ACTIVE AGENTS: 3  |  LAST SNAPSHOT: 14:20:01  │
└────────────────────────────────────────────────────────────────────┘
```

## Keybindings

| Key | Action                |
|-----|-----------------------|
| Q / Esc / Ctrl+C | Quit dashboard       |
| S   | Trigger manual snapshot |

## Provider Interface

The dashboard polls a `DashboardProvider` interface every 2 seconds:

```go
type DashboardProvider interface {
    Stats() ComponentStats
    RecentEvents() []AuditEvent
    ChartData() []ChartDataPoint
    ActiveRules() []RuleEntry
}
```

Replace the mock provider in `dashboard.NewApp()` with a real implementation wired to the ANS daemon's 5 components: Audit Trail, Snapshot Engine, MCP Proxy, Policy Engine, and Identity Broker.

## Color Palette

- Background: `#0A0A0F` (near-black)
- Primary: `#A855F7` (vivid purple)
- Secondary: `#7C3AED` (deep violet)
- Foreground: `#E2E8F0` (cool off-white)
- Dim text: `#94A3B8` (muted slate)
- Alert: `#F472B6` (pink)
- Success: `#86EFAC` (muted green)
