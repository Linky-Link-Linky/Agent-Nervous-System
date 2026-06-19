package policy

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestConditionAllMatch(t *testing.T) {
	p := &Policy{
		ID: "test-all", Name: "Test All Match", Enabled: true, Priority: 100,
		Conditions: Condition{
			All: []Condition{
				{Fact: "color", Operator: OpEq, Value: "red"},
				{Fact: "size", Operator: OpGt, Value: float64(5)},
			},
		},
		Action: Action{Effect: EffectDeny, ErrorMessage: "nope"},
	}
	eng := NewEngine()
	facts := func(path string) (interface{}, bool) {
		switch path {
		case "color":
			return "red", true
		case "size":
			return float64(10), true
		}
		return nil, false
	}
	res := eng.Evaluate(p, facts)
	if !res.Matched {
		t.Fatal("expected match")
	}
	if res.Effect != EffectDeny {
		t.Fatal("expected deny")
	}
}

func TestConditionAllMismatch(t *testing.T) {
	p := &Policy{
		ID: "test-all-mismatch", Name: "Test All Mismatch", Enabled: true,
		Conditions: Condition{
			All: []Condition{
				{Fact: "color", Operator: OpEq, Value: "red"},
				{Fact: "size", Operator: OpGt, Value: float64(5)},
			},
		},
		Action: Action{Effect: EffectDeny},
	}
	eng := NewEngine()
	facts := func(path string) (interface{}, bool) {
		switch path {
		case "color":
			return "blue", true
		case "size":
			return float64(10), true
		}
		return nil, false
	}
	res := eng.Evaluate(p, facts)
	if res.Matched {
		t.Fatal("expected no match")
	}
}

func TestConditionAny(t *testing.T) {
	p := &Policy{
		ID: "test-any", Name: "Test Any", Enabled: true,
		Conditions: Condition{
			Any: []Condition{
				{Fact: "role", Operator: OpEq, Value: "admin"},
				{Fact: "role", Operator: OpEq, Value: "root"},
			},
		},
		Action: Action{Effect: EffectAllow},
	}
	eng := NewEngine()
	facts := func(path string) (interface{}, bool) {
		if path == "role" {
			return "admin", true
		}
		return nil, false
	}
	res := eng.Evaluate(p, facts)
	if !res.Matched {
		t.Fatal("expected match via any")
	}
}

func TestConditionNone(t *testing.T) {
	p := &Policy{
		ID: "test-none", Name: "Test None", Enabled: true,
		Conditions: Condition{
			None: []Condition{
				{Fact: "banned", Operator: OpEq, Value: true},
			},
		},
		Action: Action{Effect: EffectAllow},
	}
	eng := NewEngine()
	facts := func(path string) (interface{}, bool) {
		if path == "banned" {
			return false, true
		}
		return nil, false
	}
	res := eng.Evaluate(p, facts)
	if !res.Matched {
		t.Fatal("expected match (none matched)")
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		name     string
		actual   interface{}
		op       Operator
		expected interface{}
		want     bool
	}{
		{"eq string match", "hello", OpEq, "hello", true},
		{"eq string mismatch", "hello", OpEq, "world", false},
		{"neq match", "hello", OpNeq, "world", true},
		{"contains match", "hello world", OpContains, "world", true},
		{"contains mismatch", "hello world", OpContains, "xyz", false},
		{"gt match", float64(10), OpGt, float64(5), true},
		{"gt mismatch", float64(3), OpGt, float64(5), false},
		{"lt match", float64(3), OpLt, float64(5), true},
		{"gte match", float64(5), OpGte, float64(5), true},
		{"lte match", float64(5), OpLte, float64(5), true},
		{"in comma string", "b", OpIn, "a,b,c", true},
		{"not in", "d", OpNotIn, "a,b,c", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareValues(tt.actual, tt.op, tt.expected)
			if got != tt.want {
				t.Errorf("compareValues(%v, %s, %v) = %v, want %v", tt.actual, tt.op, tt.expected, got, tt.want)
			}
		})
	}
}

func TestPIIEmail(t *testing.T) {
	pii := DetectPII("contact me at test@example.com for details")
	if !pii.HasEmail {
		t.Fatal("expected email detected")
	}
	if !pii.HasPII {
		t.Fatal("expected PII detected")
	}
}

func TestPIISSN(t *testing.T) {
	pii := DetectPII("SSN: 123-45-6789")
	if !pii.HasSSN {
		t.Fatal("expected SSN detected")
	}
}

func TestPIIAPIKey(t *testing.T) {
	pii := DetectPII("sk-proj-ABCDEF1234567890abcdef123456")
	if !pii.HasAPIKey {
		t.Fatal("expected API key detected")
	}
}

func TestPIIIP(t *testing.T) {
	pii := DetectPII("connecting from 192.168.1.1")
	if !pii.HasIP {
		t.Fatal("expected IP detected")
	}
	if !pii.HasPII {
		t.Fatal("expected HasPII to include IPs")
	}
}

func TestPIIPhone(t *testing.T) {
	pii := DetectPII("call +15551234567 for support")
	if !pii.HasPhone {
		t.Fatal("expected phone detected")
	}
	if !pii.HasPII {
		t.Fatal("expected HasPII to include phones")
	}
}

func TestPIINone(t *testing.T) {
	pii := DetectPII("this is just a normal text with no sensitive data")
	if pii.HasPII {
		t.Fatal("expected no PII detected")
	}
}

func TestStoreInsertAndList(t *testing.T) {
	dir, err := os.MkdirTemp("", "policy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema); err != nil {
		t.Fatal(err)
	}

	store := NewStore(db)
	p := &Policy{
		ID: "test-store", Name: "Test Store", Enabled: true, Priority: 100,
		Conditions: Condition{All: []Condition{{Fact: "x", Operator: OpEq, Value: "y"}}},
		Action:     Action{Effect: EffectDeny, ErrorMessage: "denied"},
	}
	if err := store.Insert(p); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("test-store")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Store" {
		t.Fatalf("got name %q, want %q", got.Name, "Test Store")
	}
	if !got.Enabled {
		t.Fatal("expected enabled")
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(list))
	}
	enabled, err := store.ListEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled policy, got %d", len(enabled))
	}
	if err := store.Delete("test-store"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("test-store"); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestMakeFactsPII(t *testing.T) {
	facts := MakeFacts("agent1", "http.send", "pre", "customer email: john@example.com", "", nil)
	v, ok := facts("context.contains_pii")
	if !ok {
		t.Fatal("expected context.contains_pii")
	}
	if !v.(bool) {
		t.Fatal("expected PII detected in payload summary")
	}
	email, ok := facts("context.has_email")
	if !ok {
		t.Fatal("expected context.has_email")
	}
	if !email.(bool) {
		t.Fatal("expected email flag set")
	}
	// Ensure aggregate and individual are consistent
	allFacts := []string{"context.has_email", "context.has_ssn", "context.has_credit_card",
		"context.has_phone", "context.has_ip", "context.has_api_key"}
	anyIndividual := false
	for _, f := range allFacts {
		if v, ok := facts(f); ok && v.(bool) {
			anyIndividual = true
			break
		}
	}
	if !anyIndividual {
		t.Fatal("expected at least one individual PII flag to match when contains_pii is true")
	}
}

func TestMakeFactsNoPII(t *testing.T) {
	facts := MakeFacts("agent1", "file.write", "pre", "just a log message", "", nil)
	v, ok := facts("context.contains_pii")
	if !ok {
		t.Fatal("expected context.contains_pii")
	}
	if v.(bool) {
		t.Fatal("expected no PII in safe payload")
	}
}

func TestExecutorBlocksPII(t *testing.T) {
	dir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema); err != nil {
		t.Fatal(err)
	}

	store := NewStore(db)
	executor := NewExecutor(store)
	if err := store.Insert(&Policy{
		ID: "no-pii-open", Name: "Block PII Open Weights", Enabled: true, Priority: 100,
		Conditions: Condition{
			All: []Condition{
				{Fact: "model.weight_type", Operator: OpEq, Value: "open"},
				{Fact: "context.contains_pii", Operator: OpEq, Value: true},
			},
		},
		Action: Action{Effect: EffectDeny, ErrorType: "NociceptionError",
			ErrorMessage: "PII blocked on open-weight model"},
	}); err != nil {
		t.Fatal(err)
	}
	ctx := map[string]interface{}{"model.weight_type": "open"}
	facts := MakeFacts("agent1", "http.send", "pre", "email: user@example.com", "", ctx)
	res := executor.Evaluate(facts)
	if res.Allowed {
		t.Fatal("expected action to be denied")
	}
	if !res.Denied {
		t.Fatal("expected denied flag")
	}
	if res.Nociception == nil {
		t.Fatal("expected nociception error")
	}
	if res.Nociception.PolicyID != "no-pii-open" {
		t.Fatalf("expected policy ID 'no-pii-open', got %q", res.Nociception.PolicyID)
	}
}

func TestExecutorAllowsClean(t *testing.T) {
	dir, err := os.MkdirTemp("", "executor-clean-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema); err != nil {
		t.Fatal(err)
	}

	store := NewStore(db)
	executor := NewExecutor(store)
	store.Insert(&Policy{
		ID: "no-pii-open", Name: "Block PII Open Weights", Enabled: true, Priority: 100,
		Conditions: Condition{
			All: []Condition{
				{Fact: "model.weight_type", Operator: OpEq, Value: "open"},
				{Fact: "context.contains_pii", Operator: OpEq, Value: true},
			},
		},
		Action: Action{Effect: EffectDeny, ErrorMessage: "blocked"},
	})
	ctx := map[string]interface{}{"model.weight_type": "closed"}
	facts := MakeFacts("agent1", "file.read", "pre", "no pii here at all", "", ctx)
	res := executor.Evaluate(facts)
	if !res.Allowed {
		t.Fatal("expected action to be allowed (model not open-weight)")
	}
}
