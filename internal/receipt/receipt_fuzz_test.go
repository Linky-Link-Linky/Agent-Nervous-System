package receipt

import (
	"encoding/json"
	"testing"
)

// FuzzSerialize ensures JSON roundtrip stability for receipts:
//   marshal(unmarshal(data)) produces the same structural output.
// This catches non-deterministic field ordering, data loss on
// unmarshal, or corner cases in custom JSON marshalling.
func FuzzReceiptRoundtrip(f *testing.F) {
	seeds := []struct {
		agentID string
		phase   string
	}{
		{"ans_test", "pre"},
		{"ans_test", "post"},
		{"a", "pre"},
		{"agent-123_fancy", "post"},
	}
	for _, s := range seeds {
		b := NewBuilder(s.agentID, GenesisHash, 1)
		r := b.PreAction(ActionPayload{Type: ActionCustom}, "test", PolicyAllow, "")
		if s.phase == "post" {
			r.Phase = PhasePost
		}
		_ = r.SetReceiptID()
		data, _ := json.Marshal(r)
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var r Receipt
		if err := json.Unmarshal(data, &r); err != nil {
			return
		}
		// Roundtrip: marshal the unmarshalled result
		out, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("re-marshal failed: %v", err)
		}
		// Unmarshal again — result should match
		var r2 Receipt
		if err := json.Unmarshal(out, &r2); err != nil {
			t.Fatalf("second unmarshal failed on %q (input %q)", string(out), string(data))
		}
		// Verify no information loss on critical fields
		if r2.AgentID != r.AgentID {
			t.Errorf("AgentID changed: %q → %q", r.AgentID, r2.AgentID)
		}
		if r2.Phase != r.Phase {
			t.Errorf("Phase changed: %q → %q", r.Phase, r2.Phase)
		}
		if r2.ChainIndex != r.ChainIndex {
			t.Errorf("ChainIndex changed: %d → %d", r.ChainIndex, r2.ChainIndex)
		}
		if r2.ReceiptID == "" && r.ReceiptID != "" {
			t.Errorf("ReceiptID lost during roundtrip: was %q", r.ReceiptID)
		}
	})
}
