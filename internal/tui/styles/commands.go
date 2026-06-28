package styles

type CommandDef struct {
	Name     string
	Desc     string
	Shortcut string
}

var CommandList = []CommandDef{
	{Name: "chain", Desc: "View receipt chain", Shortcut: "F2"},
	{Name: "stream", Desc: "Live receipt stream", Shortcut: "F3"},
	{Name: "snap", Desc: "Snapshot management", Shortcut: "F4"},
	{Name: "policy", Desc: "Policy CRUD and eval", Shortcut: "F5"},
	{Name: "token", Desc: "Ephemeral token manager", Shortcut: "F6"},
	{Name: "proxy", Desc: "MCP proxy dashboard", Shortcut: "F7"},
	{Name: "dashboard", Desc: "Daemon health overview", Shortcut: "F1"},
	{Name: "init", Desc: "Initialize ANS config"},
	{Name: "start", Desc: "Start ANS daemon"},
	{Name: "stop", Desc: "Stop ANS daemon"},
	{Name: "status", Desc: "Daemon status"},
	{Name: "doctor", Desc: "Run diagnostics"},
	{Name: "register", Desc: "Register agent"},
	{Name: "verify", Desc: "Verify receipt"},
	{Name: "update", Desc: "Self-update"},
	{Name: "export", Desc: "Export chain data"},
	{Name: "help", Desc: "Show help"},
}
