package providers

import "time"

type Component string

const (
	AuditTrail     Component = "audit-trail"
	SnapshotEngine Component = "snapshot-engine"
	MCPProxy       Component = "mcp-proxy"
	PolicyEngine   Component = "policy-engine"
	IdentityBroker Component = "identity-broker"
)

type EventType string

const (
	EventRequest        EventType = "REQUEST"
	EventBlocked        EventType = "BLOCKED"
	EventCommit         EventType = "COMMIT"
	EventSnapshot       EventType = "SNAPSHOT"
	EventViolation      EventType = "VIOLATION"
	EventInfo           EventType = "INFO"
	EventAlloc          EventType = "ALLOC"
	EventFree           EventType = "FREE"
	EventTrigger        EventType = "TRIGGER"
	EventAllowed        EventType = "ALLOWED"
	EventExpired        EventType = "EXPIRED"
	EventActive         EventType = "ACTIVE"
)

type AuditEvent struct {
	Timestamp time.Time
	Component Component
	EventType EventType
	Hash      string
}

type CPUStats struct {
	Model     string
	Cores     int
	UsagePct  float64
	PerCore   []float64
}

type GPUStats struct {
	Count      int
	Models     []string
	UsagePct   float64
	MemTotalMB int
	MemUsedMB  int
	TempC      float64
}

type MemStats struct {
	TotalGB  int
	UsedGB   int
	Pct      float64
}

type DiskStats struct {
	ReadSpeedMBs  float64
	WriteSpeedMBs float64
	UsedGB        int
	TotalGB       int
	Pct           float64
}

type NetStats struct {
	BytesInMB  float64
	BytesOutMB float64
	SpeedInMBs float64
	SpeedOutMBs float64
}

type ComponentStats struct {
	CPU            CPUStats
	GPU            GPUStats
	Mem            MemStats
	Disk           DiskStats
	Net            NetStats
	Procs          []ProcEntry
	ActiveRules    int
	Violations24h  int
	LastEnforcement time.Time
	Uptime         time.Duration
	ActiveAgents   int
	LastSnapshot   time.Time
	MCPStatus      string
	BrokerStatus   string
}

type RuleEntry struct {
	Rule    string
	Verdict string
}

type ChartDataPoint struct {
	Hour    int
	Values  map[Component]float64
}

type ProcEntry struct {
	Name   string
	PID    int
	CPU    float64
	MemMB  int
}

type DashboardProvider interface {
	Stats() ComponentStats
	RefreshHardware()
	RecentEvents() []AuditEvent
	ChartData() []ChartDataPoint
	ActiveRules() []RuleEntry
	TopProcesses() []ProcEntry
}
