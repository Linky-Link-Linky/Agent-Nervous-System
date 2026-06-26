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
	mu          sync.RWMutex
	startTime   time.Time
	events      []AuditEvent
	evIdx       int
	ruleIdx     int
	chartBase   map[int]map[Component]float64
	cachedStats ComponentStats
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

	m := &MockProvider{
		startTime: start,
		events:    make([]AuditEvent, 0, 200),
		chartBase: base,
	}
	m.RefreshHardware()
	return m
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
	return m.cachedStats
}

func (m *MockProvider) RefreshHardware() {
	uptime := time.Since(m.startTime)
	brokerStates := []string{"IDLE", "ACTIVE", "EXPIRED"}
	mcpStates := []string{"ACTIVE", "ACTIVE", "ACTIVE", "DEGRADED"}
	hw := sampleHardware()

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
		Disk: DiskStats{
			ReadSpeedMBs:  hw.DiskReadMBs,
			WriteSpeedMBs: hw.DiskWriteMBs,
			UsedGB:        hw.DiskUsedGB,
			TotalGB:       hw.DiskTotalGB,
			Pct:           hw.DiskPct,
		},
		Net: NetStats{
			BytesInMB:    hw.NetInMB,
			BytesOutMB:   hw.NetOutMB,
			SpeedInMBs:   hw.NetSpeedInMBs,
			SpeedOutMBs:  hw.NetSpeedOutMBs,
		},
		Procs: toProcEntries(hw.Procs),
		ActiveRules:    12 + randInt(5),
		Violations24h:  8 + randInt(10),
		LastEnforcement: time.Now().Add(-time.Duration(randInt(300)) * time.Second),
		Uptime:         uptime,
		ActiveAgents:   3 + randInt(3),
		LastSnapshot:   time.Now().Add(-time.Duration(randInt(120)) * time.Second),
		MCPStatus:      mcpStates[randInt(len(mcpStates))],
		BrokerStatus:   brokerStates[randInt(len(brokerStates))],
	}

	m.mu.Lock()
	m.cachedStats = s
	m.mu.Unlock()
}

func toProcEntries(samples []procSample) []ProcEntry {
	out := make([]ProcEntry, len(samples))
	for i, s := range samples {
		out[i] = ProcEntry{Name: s.Name, PID: s.PID, CPU: s.CPU, MemMB: s.MemMB}
	}
	return out
}

func (m *MockProvider) TopProcesses() []ProcEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cachedStats.Procs
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func (m *MockProvider) RecentEvents() []AuditEvent {
	return nil
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
