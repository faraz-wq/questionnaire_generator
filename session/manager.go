package session

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type SessionManager struct {
	sessions map[string]*SessionState
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionState),
	}
}

func (m *SessionManager) Get(id string) (*SessionState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return s.Clone(), true
}

func (m *SessionManager) Create(state *SessionState) (string, error) {
	if state.ID == "" {
		state.ID = uuid.New().String()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.sessions[state.ID]; exists {
		return "", fmt.Errorf("session %q already exists", state.ID)
	}
	m.sessions[state.ID] = state.Clone()
	return state.ID, nil
}

func (m *SessionManager) Update(id string, state *SessionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = state.Clone()
}

func (m *SessionManager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func (m *SessionManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}