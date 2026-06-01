package session

import (
	"sync"
	"testing"
)

func TestSessionManagerCreate(t *testing.T) {
	sm := NewSessionManager()
	state := &SessionState{
		DomainID: "test",
		Pool:     []*Question{},
		History:  []HistoryEntry{},
	}

	id, err := sm.Create(state)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	ids := sm.List()
	if len(ids) != 1 {
		t.Fatalf("expected 1 session, got %d", len(ids))
	}
}

func TestSessionManagerGetNotFound(t *testing.T) {
	sm := NewSessionManager()
	_, ok := sm.Get("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent session")
	}
}

func TestSessionManagerGetAndUpdate(t *testing.T) {
	sm := NewSessionManager()
	state := &SessionState{
		ID:       "test-id",
		DomainID: "test",
	}

	_, err := sm.Create(state)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	retrieved, ok := sm.Get("test-id")
	if !ok {
		t.Fatal("expected session to be found")
	}
	if retrieved.DomainID != "test" {
		t.Errorf("expected DomainID 'test', got %q", retrieved.DomainID)
	}

	retrieved.DomainID = "modified"
	sm.Update("test-id", retrieved)

	updated, ok := sm.Get("test-id")
	if !ok {
		t.Fatal("expected session to be found after update")
	}
	if updated.DomainID != "modified" {
		t.Errorf("expected DomainID 'modified', got %q", updated.DomainID)
	}
}

func TestSessionManagerDelete(t *testing.T) {
	sm := NewSessionManager()
	state := &SessionState{ID: "to-delete"}

	_, err := sm.Create(state)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	sm.Delete("to-delete")
	_, ok := sm.Get("to-delete")
	if ok {
		t.Fatal("expected session to be deleted")
	}
}

func TestSessionManagerConcurrency(t *testing.T) {
	sm := NewSessionManager()

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			state := &SessionState{ID: "conc-" + string(rune('0'+idx))}
			_, _ = sm.Create(state)
		}(i)
	}
	wg.Wait()

	if len(sm.List()) != 10 {
		t.Errorf("expected 10 sessions, got %d", len(sm.List()))
	}

	var wg2 sync.WaitGroup
	for range 10 {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			states := sm.List()
			for _, id := range states {
				_, _ = sm.Get(id)
			}
		}()
	}
	wg2.Wait()
}

func TestSessionStateClone(t *testing.T) {
	state := &SessionState{
		ID:       "test",
		DomainID: "domain",
		AskedTotal: 5,
	}
	clone := state.Clone()
	clone.DomainID = "changed"

	if state.DomainID != "domain" {
		t.Error("original should not be affected by clone modification")
	}
}

func TestNewCoverage(t *testing.T) {
	ids := []string{"a", "b", "c"}
	c := NewCoverage(ids)
	if len(c.LeafScores) != 3 {
		t.Errorf("expected 3 leaf scores, got %d", len(c.LeafScores))
	}
	for _, id := range ids {
		if _, ok := c.LeafScores[id]; !ok {
			t.Errorf("expected leaf score entry for %s", id)
		}
	}
}