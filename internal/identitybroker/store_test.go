package identitybroker

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupStore(t *testing.T) *SQLiteTokenStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open(): %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema); err != nil {
		t.Fatalf("Exec(Schema): %v", err)
	}
	return NewSQLiteTokenStore(db)
}

func makeToken(id string) *Token {
	return &Token{
		ID: id, AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), LastUsedAt: time.Now(),
		UsageCount: 0, MaxUsage: 5, ProviderData: "prov-data", Metadata: "meta",
	}
}

func TestInsertAndGet(t *testing.T) {
	s := setupStore(t)
	tok := makeToken("tok_1")
	if err := s.Insert(tok); err != nil {
		t.Fatalf("Insert(): %v", err)
	}
	got, err := s.Get("tok_1")
	if err != nil {
		t.Fatalf("Get(): %v", err)
	}
	if got.ID != "tok_1" || got.AgentID != "ans_agent1" {
		t.Errorf("Get() = %+v", got)
	}
	if got.ProviderData != "prov-data" || got.Metadata != "meta" {
		t.Errorf("Get() ProviderData/Metadata mismatch: %+v", got)
	}
}

func TestInsertReplace(t *testing.T) {
	s := setupStore(t)
	if err := s.Insert(makeToken("tok_1")); err != nil {
		t.Fatal(err)
	}
	updated := makeToken("tok_1")
	updated.Resource = "s3://other"
	if err := s.Insert(updated); err != nil {
		t.Fatalf("Insert(replace): %v", err)
	}
	got, _ := s.Get("tok_1")
	if got.Resource != "s3://other" {
		t.Errorf("Get() after replace Resource = %q, want s3://other", got.Resource)
	}
}

func TestGetNotFound(t *testing.T) {
	s := setupStore(t)
	_, err := s.Get("tok_noexist")
	if err == nil {
		t.Error("Get() for missing token returned nil error")
	}
}

func TestUpdate(t *testing.T) {
	s := setupStore(t)
	tok := makeToken("tok_1")
	s.Insert(tok)
	tok.UsageCount = 3
	tok.LastUsedAt = time.Now()
	if err := s.Update(tok); err != nil {
		t.Fatalf("Update(): %v", err)
	}
	got, _ := s.Get("tok_1")
	if got.UsageCount != 3 {
		t.Errorf("UsageCount after update = %d, want 3", got.UsageCount)
	}
}

func TestDelete(t *testing.T) {
	s := setupStore(t)
	s.Insert(makeToken("tok_1"))
	if err := s.Delete("tok_1"); err != nil {
		t.Fatalf("Delete(): %v", err)
	}
	_, err := s.Get("tok_1")
	if err == nil {
		t.Error("Get() after Delete returned nil error")
	}
}

func TestList(t *testing.T) {
	s := setupStore(t)
	s.Insert(makeToken("tok_1"))
	s.Insert(makeToken("tok_2"))
	toks, err := s.List("", "", 10)
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(toks) != 2 {
		t.Fatalf("List() returned %d tokens, want 2", len(toks))
	}
}

func TestListFilterByAgent(t *testing.T) {
	s := setupStore(t)
	t1 := makeToken("tok_1")
	t2 := makeToken("tok_2")
	t2.AgentID = "ans_agent2"
	s.Insert(t1)
	s.Insert(t2)
	toks, err := s.List("ans_agent1", "", 10)
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(toks) != 1 {
		t.Fatalf("List(agent1) returned %d tokens, want 1", len(toks))
	}
}

func TestListLimit(t *testing.T) {
	s := setupStore(t)
	for i := 0; i < 5; i++ {
		s.Insert(makeToken("tok_" + string(rune('0'+i))))
	}
	toks, err := s.List("", "", 2)
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(toks) > 2 {
		t.Fatalf("List(limit=2) returned %d tokens", len(toks))
	}
}

func TestCleanupExpired(t *testing.T) {
	s := setupStore(t)
	fresh := makeToken("tok_fresh")
	expired := makeToken("tok_expired")
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	s.Insert(fresh)
	s.Insert(expired)
	count, err := s.CleanupExpired()
	if err != nil {
		t.Fatalf("CleanupExpired(): %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d tokens, want 1", count)
	}
	_, err = s.Get("tok_expired")
	if err == nil {
		t.Error("Get(expired) after cleanup returned nil error")
	}
}

func TestRevokeAllForAgent(t *testing.T) {
	s := setupStore(t)
	a1 := makeToken("tok_a1")
	a1.AgentID = "ans_agent1"
	a2 := makeToken("tok_a2")
	a2.AgentID = "ans_agent1"
	b1 := makeToken("tok_b1")
	b1.AgentID = "ans_agent2"
	s.Insert(a1)
	s.Insert(a2)
	s.Insert(b1)

	if err := s.RevokeAllForAgent("ans_agent1", "tester"); err != nil {
		t.Fatalf("RevokeAllForAgent(): %v", err)
	}

	got1, _ := s.Get("tok_a1")
	if !got1.Revoked {
		t.Error("tok_a1 not revoked")
	}
	got2, _ := s.Get("tok_a2")
	if !got2.Revoked {
		t.Error("tok_a2 not revoked")
	}
	gotB, _ := s.Get("tok_b1")
	if gotB.Revoked {
		t.Error("tok_b1 should not be revoked")
	}
}

func TestCleanupRevoked(t *testing.T) {
	s := setupStore(t)
	revoked := makeToken("tok_revoked")
	revoked.Revoked = true
	revoked.RevokedAt = time.Now().Add(-time.Hour)
	s.Insert(revoked)
	s.Insert(makeToken("tok_fresh"))

	count, err := s.CleanupExpired()
	if err != nil {
		t.Fatalf("CleanupExpired(): %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d, want 1 (revoked > 1h ago)", count)
	}
}
