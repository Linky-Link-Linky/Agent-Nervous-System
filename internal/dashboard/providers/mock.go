package providers

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"
)

type MockProvider struct {
	mu        sync.RWMutex
	startTime time.Time
	events    []AuditEvent
	evIdx     int
	ruleIdx   int
	chartBase map[int]map[Component]float64
}

func NewMockProvider() *MockProvider {
	start := time.Now().Add(-4 * time.Hour)
	base := make(map[int]map[Component]float64)
	now := time.Now()
	for h := 0; h < 24; h++ {
		t := now.Add(-time.Duration(23-h) * time.Hour)
		hr := t.Hour()
		vals := make(map[Component]float64)
		vals[AuditTrail] = float64(30 + int(20*math.Sin(float64(hr)*0.5)+15))
		vals[SnapshotEngine] = float64(20 + int(15*math.Cos(float64(hr)*0.3)+10))
		vals[MCPProxy] = float64(40 + int(25*math.Sin(float64(hr)*0.7+1)+20))
		vals[PolicyEngine] = float64(15 + int(10*math.Cos(float64(hr)*0.4+2)+8))
		vals[IdentityBroker] = float64(10 + int(8*math.Sin(float64(hr)*0.2+3)+5))
		base[h] = vals
	}

	return &MockProvider{
		startTime: start,
		events:    make([]AuditEvent, 0, 200),
		chartBase: base,
	}
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:n]
}

func randInt(n int) int {
	bi, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(bi.Int64())
}

func (m *MockProvider) Stats() ComponentStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)
	brokerStates := []string{"IDLE", "ACTIVE", "EXPIRED"}
	mcpStates := []string{"ACTIVE", "ACTIVE", "ACTIVE", "DEGRADED"}

	hw := detectHardware()

	return ComponentStats{
		CPUModel:       hw.CPUModel,
		CPUCores:       hw.CPUCores,
		CPUUsagePct:    23.5 + float64(randInt(200))/10,
		TotalRAMGB:     hw.RAMGB,
		UsedRAMGB:      hw.UsedRAMGB,
		GPUCount:       hw.GPUCount,
		GPUModels:      hw.GPUModels,
		ActiveRules:    12 + randInt(5),
		Violations24h:  8 + randInt(10),
		LastEnforcement: time.Now().Add(-time.Duration(randInt(300)) * time.Second),
		Uptime:         uptime,
		ActiveAgents:   3 + randInt(3),
		LastSnapshot:   time.Now().Add(-time.Duration(randInt(120)) * time.Second),
		MCPStatus:      mcpStates[randInt(len(mcpStates))],
		BrokerStatus:   brokerStates[randInt(len(brokerStates))],
	}
}

func (m *MockProvider) RecentEvents() []AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	components := []Component{AuditTrail, SnapshotEngine, MCPProxy, PolicyEngine, IdentityBroker}
	eventTypes := []EventType{EventRequest, EventInfo, EventCommit, EventAlloc, EventTrigger, EventAllowed}
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
		m.events = append(m.events, AuditEvent{
			Timestamp: now.Add(-time.Duration(randInt(3)) * time.Second),
			Component: comp,
			EventType: et,
			Hash:      randHex(7),
		})
		m.evIdx++
	}

	if len(m.events) > 200 {
		m.events = m.events[len(m.events)-200:]
	}

	out := make([]AuditEvent, len(m.events))
	copy(out, m.events)
	return out
}

func (m *MockProvider) ChartData() []ChartDataPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pts := make([]ChartDataPoint, 0, 24)
	for h := 0; h < 24; h++ {
		vals := make(map[Component]float64)
		for c, v := range m.chartBase[h] {
			vals[c] = v
		}
		pts = append(pts, ChartDataPoint{Hour: h, Values: vals})
	}
	return pts
}

func (m *MockProvider) ActiveRules() []RuleEntry {
	rules := []struct {
		name   string
		verdict string
	}{
		{"agent.file.write → /etc/passwd", "DENY"},
		{"agent.tool.call → read_file", "ALLOW"},
		{"agent.network → external_egress", "DENY"},
		{"agent.exec → shell_invoke", "DENY"},
		{"agent.db.read → users_table", "ALLOW"},
		{"agent.file.read → /etc/shadow", "DENY"},
		{"agent.api.call → internal_svc", "ALLOW"},
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	n := 3 + randInt(3)
	out := make([]RuleEntry, n)
	for i := 0; i < n; i++ {
		idx := (m.ruleIdx + i) % len(rules)
		out[i] = RuleEntry{Rule: rules[idx].name, Verdict: rules[idx].verdict}
	}
	m.ruleIdx = (m.ruleIdx + 1) % len(rules)
	return out
}
