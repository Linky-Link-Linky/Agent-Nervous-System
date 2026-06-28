package policy

import (
	"log/slog"
	"sync"
)

// Executor evaluates actions against all active policies.
type Executor struct {
	store  *Store
	engine Engine
}

// NewExecutor creates a policy executor.
func NewExecutor(store *Store) *Executor {
	return &Executor{
		store:  store,
		engine: NewEngine(),
	}
}

// EvalResult is the result of evaluating an action against all policies.
type EvalResult struct {
	Allowed       bool              `json:"allowed"`
	Denied        bool              `json:"denied"`
	Nociception   *NociceptionError `json:"nociception,omitempty"`
	PolicyResults []*PolicyResult   `json:"policy_results,omitempty"`
}

// Evaluate runs all enabled policies against the given facts.
func (ex *Executor) Evaluate(facts FactProvider) *EvalResult {
	policies, err := ex.store.ListEnabled()
	if err != nil {
		slog.Error("policy listing enabled policies failed", "error", err)
		return &EvalResult{Allowed: false, Denied: true}
	}

	res := &EvalResult{Allowed: true}
	for _, p := range policies {
		pr := ex.engine.Evaluate(p, facts)
		res.PolicyResults = append(res.PolicyResults, pr)
		if pr.Matched && pr.Effect == EffectDeny {
			res.Allowed = false
			res.Denied = true
			res.Nociception = &NociceptionError{
				PolicyID:   p.ID,
				PolicyName: p.Name,
				Message:    pr.ErrorMessage,
				Severity:   string(p.Severity),
			}
			// Deny is terminal — first denial wins
			return res
		}
	}
	return res
}

// MakeFacts builds a FactProvider from receipt/action data and context.
// PII scanning is deferred until the first access to a PII fact path,
// so it is skipped entirely when no policy rule checks PII.
func MakeFacts(agentID, actionType, phase, payloadSummary, parentAgentID string, context map[string]interface{}) FactProvider {
	lazyPII := sync.OnceValue(func() PIIClassification {
		return DetectPII(payloadSummary + agentID + actionType)
	})
	return func(factPath string) (interface{}, bool) {
		switch factPath {
		case "agent_id":
			return agentID, true
		case "action_type":
			return actionType, true
		case "phase":
			return phase, true
		case "payload_summary":
			return payloadSummary, true
		case "parent_agent_id":
			return parentAgentID, true
		case "context.contains_pii":
			return lazyPII().HasPII, true
		case "context.has_email":
			return lazyPII().HasEmail, true
		case "context.has_ssn":
			return lazyPII().HasSSN, true
		case "context.has_credit_card":
			return lazyPII().HasCreditCard, true
		case "context.has_ip":
			return lazyPII().HasIP, true
		case "context.has_phone":
			return lazyPII().HasPhone, true
		case "context.has_api_key":
			return lazyPII().HasAPIKey, true
		default:
			// Check custom context keys
			if context != nil {
				if v, ok := context[factPath]; ok {
					return v, true
				}
			}
			return nil, false
		}
	}
}
