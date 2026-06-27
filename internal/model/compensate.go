package model

type CompensateStep struct {
	ChainIndex int    `json:"chain_index"`
	ActionType string `json:"action_type"`
	HasComp    bool   `json:"has_comp"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stderr     string `json:"stderr"`
}

type CompensatePlan struct {
	Steps     []CompensateStep `json:"steps"`
	WouldRun  int              `json:"would_run"`
	Skipped   int              `json:"skipped"`
}

type CompensateResult struct {
	Steps  []CompensateStep `json:"steps"`
	Ran    int              `json:"ran"`
	Failed int              `json:"failed"`
}
