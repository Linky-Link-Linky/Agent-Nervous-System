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

type ComponentStats struct {
	CPUModel       string
	CPUCores       int
	CPUUsagePct    float64
	TotalRAMGB     int
	UsedRAMGB      int
	GPUCount       int
	GPUModels      []string
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

type DashboardProvider interface {
	Stats() ComponentStats
	RecentEvents() []AuditEvent
	ChartData() []ChartDataPoint
	ActiveRules() []RuleEntry
}
