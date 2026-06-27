package model

type CompensationStep struct {
    ChainIndex int    `json:"chain_index"`
    ActionType string `json:"action_type"`
    Command    string `json:"command"`
    HasComp    bool   `json:"has_compensation"`
}

type CompensationPlan struct {
    Steps    []*CompensationStep `json:"steps"`
    WouldRun int                 `json:"would_run"`
    Skipped  int                 `json:"skipped"`
}

type CompensationResult struct {
    Ran    int `json:"ran"`
    Failed int `json:"failed"`
    Steps  []struct {
        ChainIndex int    `json:"chain_index"`
        Command    string `json:"command"`
        ExitCode   int    `json:"exit_code"`
        Stderr     string `json:"stderr"`
    } `json:"steps"`
}
