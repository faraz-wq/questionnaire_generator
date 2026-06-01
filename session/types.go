package session

import (
	"encoding/json"

	"github.com/faraz/questionnaire_generator/domain"
	"github.com/faraz/questionnaire_generator/llm"
)

type Status string

const (
	StatusUnseen  Status = "unseen"
	StatusAsked   Status = "asked"
	StatusSkipped Status = "skipped"
)

type Question struct {
	ID               string            `json:"id"`
	Text             string            `json:"text"`
	Archetype        string            `json:"archetype"`
	Difficulty       string            `json:"difficulty"`
	Type             string            `json:"type"`
	IdealAnswerHint  string            `json:"ideal_answer_hint"`
	ExpectedConcepts []string          `json:"expected_concepts"`
	FollowUpFlag     bool              `json:"follow_up_flag"`
	MaxFollowUps     int               `json:"max_follow_ups"`
	Status           Status            `json:"status"`
	AnswerReceived   *string           `json:"answer_received"`
	EvalScore        *int              `json:"eval_score"`
	FollowUpsUsed    int               `json:"follow_ups_used"`
	NodePath         string            `json:"node_path"`
	NodeLabelPath    []string          `json:"node_label_path"`
	FollowUpTemplate *FollowUpTemplate `json:"follow_up_template,omitempty"`
	ParentQuestionID string            `json:"parent_question_id,omitempty"`
	Persona          *domain.Persona   `json:"persona,omitempty"` // optional: the persona that framed this question
	Options          []string          `json:"options,omitempty"`  // generated multiple-choice options for simulator mode
}

type FollowUpTemplate struct {
	ID      string
	Trigger string
	Text    string
}

type HistoryEntry struct {
	Question      *Question       `json:"question"`
	Answer        string          `json:"answer"`
	Eval          *llm.EvalResult `json:"eval"`
}

type GenerationLogEntry struct {
	NodePath  string `json:"node_path"`
	Archetype string `json:"archetype"`
	Count     int    `json:"count"`
	Error     string `json:"error,omitempty"`
}

type SessionState struct {
	ID              string                `json:"id"`
	DomainID        string                `json:"domain_id"`
	FrameworkPrompt string                `json:"framework_prompt"`
	Pool            []*Question           `json:"pool"`
	History         []HistoryEntry        `json:"history"`
	Coverage        *CoverageData         `json:"coverage"`
	CurrentQuestion *Question             `json:"current_question"`
	GenerationLog   []GenerationLogEntry  `json:"generation_log"`
	AskedTotal      int                   `json:"asked_total"`
	FollowUpDepth   int                   `json:"follow_up_depth"`
}

func (s *SessionState) Clone() *SessionState {
	data, _ := json.Marshal(s)
	var clone SessionState
	json.Unmarshal(data, &clone)
	return &clone
}

type CoverageData struct {
	LeafScores   map[string]float64 `json:"leaf_scores"`
	LeafCounts   map[string]int     `json:"leaf_counts"`
	NodeWeights  map[string]float64 `json:"node_weights"`
}

func NewCoverage(leafIDs []string) *CoverageData {
	c := &CoverageData{
		LeafScores:  make(map[string]float64),
		LeafCounts:  make(map[string]int),
		NodeWeights: make(map[string]float64),
	}
	for _, id := range leafIDs {
		c.LeafScores[id] = 0
		c.LeafCounts[id] = 0
	}
	return c
}