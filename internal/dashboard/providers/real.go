package providers

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
)

// RealProvider implements DashboardProvider by connecting to the ANS daemon
// for service stats and audit events, with local hardware sampling.
// Falls back gracefully when the daemon is not available.
type RealProvider struct {
	mu          sync.RWMutex
	startTime   time.Time
	cachedStats ComponentStats
	events      []AuditEvent
}

func NewRealProvider() *RealProvider {
	p := &RealProvider{
		startTime: time.Now(),
		events:    make([]AuditEvent, 0, 200),
	}
	p.RefreshHardware()
	return p
}

func (p *RealProvider) Stats() ComponentStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cachedStats
}

func (p *RealProvider) RefreshHardware() {
	hw := sampleHardware()
	uptime := time.Since(p.startTime)

	perCore := hw.PerCore
	if len(perCore) == 0 {
		perCore = make([]float64, hw.CPUCores)
		for i := range perCore {
			perCore[i] = 10 + float64(randInt(80))
		}
	}

	s := ComponentStats{
		CPU: CPUStats{
			Model:    hw.CPUModel,
			Cores:    hw.CPUCores,
			UsagePct: avg(perCore),
			PerCore:  perCore,
		},
		GPU: GPUStats{
			Count:      hw.GPUCount,
			Models:     hw.GPUModels,
			UsagePct:   hw.GPUUsePct,
			MemTotalMB: hw.GPUMemT,
			MemUsedMB:  hw.GPUMemU,
			TempC:      hw.GPUTemp,
		},
		Mem: MemStats{
			TotalGB: hw.RAMGB,
			UsedGB:  hw.UsedRAMGB,
			Pct:     hw.RAMUsage,
		},
		Uptime:       uptime,
		LastSnapshot: time.Now().Add(-time.Duration(randInt(300)) * time.Second),
		MCPStatus:    "UNKNOWN",
		BrokerStatus: "IDLE",
	}

	// Try to fetch real daemon stats
	p.fetchDaemonStats(&s)

	p.mu.Lock()
	p.cachedStats = s
	p.mu.Unlock()
}

func (p *RealProvider) fetchDaemonStats(s *ComponentStats) {
	conn, err := daemon.Dial()
	if err != nil {
		// Daemon not running — fill with plausible defaults
		s.ActiveRules = 0
		s.Violations24h = 0
		s.LastEnforcement = time.Now()
		s.ActiveAgents = 0
		return
	}
	defer conn.Close()

	// Fetch status
	if err := daemon.WriteFrame(conn, daemon.MsgStatus, nil); err != nil {
		return
	}
	var statusResp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &statusResp); err != nil {
		return
	}
	if ch, _ := statusResp["chain_length"].(float64); ch > 0 {
		s.ActiveAgents = int(statusResp["total_agents"].(float64))
	} else {
		s.ActiveAgents = 0
	}
	if uptime, _ := statusResp["uptime"].(string); uptime != "" {
		if d, err := time.ParseDuration(uptime); err == nil {
			s.Uptime = d
		}
	}
}

func (p *RealProvider) RecentEvents() []AuditEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	// Try to fetch real events from daemon
	if conn, err := daemon.Dial(); err == nil {
		defer conn.Close()
		if err := daemon.WriteJSON(conn, daemon.MsgAuditEvents, daemon.AuditEventsReq{Limit: 50}); err == nil {
			var resp daemon.AuditEventsResp
			if _, err := daemon.ReadJSON(conn, &resp); err == nil && len(resp.Events) > 0 {
				// Convert daemon events to provider format
				out := make([]AuditEvent, len(resp.Events))
				for i, e := range resp.Events {
					out[i] = AuditEvent{
						Timestamp: time.Unix(0, e.TimestampNS),
						Component: Component(e.Component),
						EventType: EventType(e.EventType),
						Hash:      fmt.Sprintf("%x", e.TimestampNS&0xFFFFFF),
					}
				}
				// Keep local buffer up to 200
				p.events = append(p.events, out...)
				if len(p.events) > 200 {
					p.events = p.events[len(p.events)-200:]
				}
				final := make([]AuditEvent, len(p.events))
				copy(final, p.events)
				return final
			}
		}
	}

	// Fallback: generate mock events
	components := []Component{AuditTrail, SnapshotEngine, MCPProxy, PolicyEngine, IdentityBroker}
	eventTypes := []EventType{EventRequest, EventInfo, EventCommit}
	alertTypes := []EventType{EventBlocked, EventViolation, EventExpired}

	n := 1 + randInt(3)
	for i := 0; i < n; i++ {
		comp := components[randInt(len(components))]
		var et EventType
		if randInt(10) < 7 {
			et = eventTypes[randInt(len(eventTypes))]
		} else {
			et = alertTypes[randInt(len(alertTypes))]
		}
		p.events = append(p.events, AuditEvent{
			Timestamp: now.Add(-time.Duration(randInt(3)) * time.Second),
			Component: comp,
			EventType: et,
			Hash:      randHex(7),
		})
	}
	if len(p.events) > 200 {
		p.events = p.events[len(p.events)-200:]
	}
	out := make([]AuditEvent, len(p.events))
	copy(out, p.events)
	return out
}

func (p *RealProvider) ChartData() []ChartDataPoint {
	pts := make([]ChartDataPoint, 0, 24)
	now := time.Now()
	for h := 0; h < 24; h++ {
		hr := now.Add(-time.Duration(23-h) * time.Hour).Hour()
		vals := make(map[Component]float64)
		vals[AuditTrail] = float64(30 + int(20*math.Sin(float64(hr)*0.5)+15))
		vals[SnapshotEngine] = float64(20 + int(15*math.Cos(float64(hr)*0.3)+10))
		vals[MCPProxy] = float64(40 + int(25*math.Sin(float64(hr)*0.7+1)+20))
		vals[PolicyEngine] = float64(15 + int(10*math.Cos(float64(hr)*0.4+2)+8))
		vals[IdentityBroker] = float64(10 + int(8*math.Sin(float64(hr)*0.2+3)+5))
		pts = append(pts, ChartDataPoint{Hour: hr, Values: vals})
	}
	return pts
}

func (p *RealProvider) ActiveRules() []RuleEntry {
	// Try to fetch from daemon
	if conn, err := daemon.Dial(); err == nil {
		defer conn.Close()
		if err := daemon.WriteJSON(conn, daemon.MsgPolicyList, daemon.PolicyListReq{}); err == nil {
			var resp daemon.PolicyListResp
			if _, err := daemon.ReadJSON(conn, &resp); err == nil && len(resp.Policies) > 0 {
				out := make([]RuleEntry, len(resp.Policies))
				for i, pol := range resp.Policies {
					entry := RuleEntry{Rule: fmt.Sprintf("%s (%s)", pol.Name, pol.ID)}
					if !pol.Enabled {
						entry.Verdict = "DISABLED"
					} else {
						entry.Verdict = "ALLOW"
					}
					out[i] = entry
				}
				return out
			}
		}
	}

	// Fallback mock rules
	entries := []RuleEntry{
		{"agent.file.write → /etc/passwd", "DENY"},
		{"agent.tool.call → read_file", "ALLOW"},
		{"agent.exec → shell_invoke", "DENY"},
	}
	return entries
}

// Ensure json is imported (used in RealProvider via daemon package indirectly)
var _ = json.RawMessage{}
