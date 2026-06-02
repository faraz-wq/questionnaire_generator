package session

import (
	"fmt"
	"sync"

	"github.com/faraz/questionnaire_generator/llm"
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

func (m *SessionManager) AddQuestionsToPool(id string, newQuestions []*Question, logs []GenerationLogEntry, leafIDs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return
	}
	for _, q := range newQuestions {
		s.Pool = append(s.Pool, q.Clone())
	}
	s.GenerationLog = append(s.GenerationLog, logs...)
	if s.Coverage != nil {
		for _, leafID := range leafIDs {
			if _, exists := s.Coverage.LeafScores[leafID]; !exists {
				s.Coverage.LeafScores[leafID] = 0
				s.Coverage.LeafCounts[leafID] = 0
			}
		}
	}
}

func (m *SessionManager) Update(id string, state *SessionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	existing, ok := m.sessions[id]
	if !ok {
		m.sessions[id] = state.Clone()
		return
	}

	// Smart Merge Strategy:
	// Update turn-related fields from state (which is the frontend turn representation)
	existing.History = make([]HistoryEntry, len(state.History))
	for i, h := range state.History {
		existing.History[i] = HistoryEntry{
			Question: h.Question.Clone(),
			Answer:   h.Answer,
		}
		if h.Eval != nil {
			existing.History[i].Eval = &llm.EvalResult{
				Score:           h.Eval.Score,
				VagueFlag:       h.Eval.VagueFlag,
				ConceptsCovered: append([]string(nil), h.Eval.ConceptsCovered...),
				Missing:         append([]string(nil), h.Eval.Missing...),
				Reasoning:       h.Eval.Reasoning,
			}
		}
	}
	existing.DomainID = state.DomainID
	existing.FrameworkPrompt = state.FrameworkPrompt
	existing.Coverage = state.Coverage.Clone()
	existing.CurrentQuestion = state.CurrentQuestion.Clone()
	existing.AskedTotal = state.AskedTotal
	existing.FollowUpDepth = state.FollowUpDepth
	existing.LimitTotal = state.LimitTotal

	// For pool questions: update the status and responses of questions that exist in state.Pool.
	// Keep all background-generated questions that exist in existing.Pool but not yet in state.Pool!
	for _, qState := range state.Pool {
		found := false
		for _, qEx := range existing.Pool {
			if qEx.ID == qState.ID {
				qEx.Status = qState.Status
				qEx.AnswerReceived = qState.AnswerReceived
				qEx.EvalScore = qState.EvalScore
				qEx.FollowUpsUsed = qState.FollowUpsUsed
				found = true
				break
			}
		}
		if !found {
			existing.Pool = append(existing.Pool, qState.Clone())
		}
	}
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