package policy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Effect is the result of evaluating a policy.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
	EffectWarn  Effect = "warn"
	EffectAudit Effect = "audit"
)

// Severity of a policy.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Operator for condition comparison.
type Operator string

const (
	OpEq       Operator = "eq"
	OpNeq      Operator = "neq"
	OpContains Operator = "contains"
	OpMatches  Operator = "matches"
	OpGt       Operator = "gt"
	OpLt       Operator = "lt"
	OpGte      Operator = "gte"
	OpLte      Operator = "lte"
	OpIn       Operator = "in"
	OpNotIn    Operator = "not_in"
)

// Condition is a single or compound condition.
type Condition struct {
	// Compound conditions
	All  []Condition `json:"all,omitempty"`
	Any  []Condition `json:"any,omitempty"`
	None []Condition `json:"none,omitempty"`

	// Leaf condition
	Fact     string      `json:"fact,omitempty"`
	Operator Operator    `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`

	// compiledOpMatches is set after JSON deserialization for OpMatches conditions.
	compiledOpMatches *regexp.Regexp
}

// Action taken when a policy matches.
type Action struct {
	Effect       Effect `json:"effect"`
	ErrorType    string `json:"error_type,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Policy is a single policy rule.
type Policy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `json:"priority"`
	Severity    Severity  `json:"severity,omitempty"`
	Conditions  Condition `json:"conditions"`
	Action      Action    `json:"action"`
	CreatedNS   int64     `json:"created_ns,omitempty"`
	UpdatedNS   int64     `json:"updated_ns,omitempty"`
}

// PolicyResult is the outcome of evaluating a single policy.
type PolicyResult struct {
	PolicyID     string `json:"policy_id"`
	PolicyName   string `json:"policy_name"`
	Effect       Effect `json:"effect"`
	Matched      bool   `json:"matched"`
	ErrorType    string `json:"error_type,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// FactProvider supplies facts for policy evaluation.
type FactProvider func(factPath string) (interface{}, bool)

// Engine evaluates policies against facts.
type Engine interface {
	Evaluate(p *Policy, facts FactProvider) *PolicyResult
}

// NociceptionError is sent when a policy denies an action.
type NociceptionError struct {
	PolicyID     string `json:"policy_id"`
	PolicyName   string `json:"policy_name"`
	Message      string `json:"message"`
	Severity     string `json:"severity"`
	TimestampNS  int64  `json:"timestamp_ns"`
}

func (e *NociceptionError) Error() string {
	return fmt.Sprintf("NociceptionError [%s]: %s (policy: %s)", e.Severity, e.Message, e.PolicyName)
}

// DefaultEngine evaluates policies using the default condition evaluator.
type DefaultEngine struct{}

func NewEngine() *DefaultEngine { return &DefaultEngine{} }

func (e *DefaultEngine) Evaluate(p *Policy, facts FactProvider) *PolicyResult {
	matched := evaluateCondition(p.Conditions, facts)
	return &PolicyResult{
		PolicyID:     p.ID,
		PolicyName:   p.Name,
		Effect:       p.Action.Effect,
		Matched:      matched,
		ErrorType:    p.Action.ErrorType,
		ErrorMessage: p.Action.ErrorMessage,
	}
}

func evaluateCondition(c Condition, facts FactProvider) bool {
	// Compound: all
	if len(c.All) > 0 {
		for _, sub := range c.All {
			if !evaluateCondition(sub, facts) {
				return false
			}
		}
		return true
	}
	// Compound: any
	if len(c.Any) > 0 {
		for _, sub := range c.Any {
			if evaluateCondition(sub, facts) {
				return true
			}
		}
		return false
	}
	// Compound: none
	if len(c.None) > 0 {
		for _, sub := range c.None {
			if evaluateCondition(sub, facts) {
				return false
			}
		}
		return true
	}
	// Leaf condition
	if c.Fact == "" {
		return true
	}
	actual, ok := facts(c.Fact)
	if !ok {
		return false
	}
	// Fast path: pre-compiled regex for OpMatches
	if c.Operator == OpMatches && c.compiledOpMatches != nil {
		return c.compiledOpMatches.MatchString(valueString(actual))
	}
	return compareValues(actual, c.Operator, c.Value)
}

func compareValues(actual interface{}, op Operator, expected interface{}) bool {
	switch op {
	case OpEq:
		return valueString(actual) == valueString(expected)
	case OpNeq:
		return valueString(actual) != valueString(expected)
	case OpContains:
		return strings.Contains(valueString(actual), valueString(expected))
	case OpMatches:
		s := valueString(actual)
		matched, err := regexp.MatchString(valueString(expected), s)
		if err != nil {
			return false
		}
		return matched
	case OpIn, OpNotIn:
		actualStr := valueString(actual)
		found := false
		switch list := expected.(type) {
		case []interface{}:
			for _, item := range list {
				if valueString(item) == actualStr {
					found = true
					break
				}
			}
		case string:
			for _, item := range strings.Split(list, ",") {
				if strings.TrimSpace(item) == actualStr {
					found = true
					break
				}
			}
		}
		if op == OpIn {
			return found
		}
		return !found
	case OpGt:
		return compareNumeric(actual, expected, func(a, b float64) bool { return a > b })
	case OpLt:
		return compareNumeric(actual, expected, func(a, b float64) bool { return a < b })
	case OpGte:
		return compareNumeric(actual, expected, func(a, b float64) bool { return a >= b })
	case OpLte:
		return compareNumeric(actual, expected, func(a, b float64) bool { return a <= b })
	}
	return false
}

// valueString converts an interface{} to a string without fmt.Sprintf allocation.
func valueString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case int:
		return strconv.Itoa(s)
	case int64:
		return strconv.FormatInt(s, 10)
	case uint64:
		return strconv.FormatUint(s, 10)
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case bool:
		if s {
			return "true"
		}
		return "false"
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%v", s)
	}
}

func compareNumeric(actual, expected interface{}, cmp func(float64, float64) bool) bool {
	a, aOK := toFloat64(actual)
	b, bOK := toFloat64(expected)
	if !aOK || !bOK {
		return false
	}
	return cmp(a, b)
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// CompileRegexp recursively pre-compiles all regex patterns in the condition tree.
func (c *Condition) CompileRegexp() error {
	if c.Operator == OpMatches {
		pattern, ok := c.Value.(string)
		if ok {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			c.compiledOpMatches = re
		}
	}
	for i := range c.All {
		if err := c.All[i].CompileRegexp(); err != nil {
			return err
		}
	}
	for i := range c.Any {
		if err := c.Any[i].CompileRegexp(); err != nil {
			return err
		}
	}
	for i := range c.None {
		if err := c.None[i].CompileRegexp(); err != nil {
			return err
		}
	}
	return nil
}
