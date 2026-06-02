package session

import (
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
	CorrectIndex     int               `json:"-"`                  // index of correct option (0-3)
	Feedbacks        []string          `json:"-"`                  // feedback explanations for why each option is correct or incorrect
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
	LimitTotal      int                   `json:"limit_total"`
}

func (q *Question) Clone() *Question {
	if q == nil {
		return nil
	}
	
	clone := &Question{
		ID:               q.ID,
		Text:             q.Text,
		Archetype:        q.Archetype,
		Difficulty:       q.Difficulty,
		Type:             q.Type,
		IdealAnswerHint:  q.IdealAnswerHint,
		FollowUpFlag:     q.FollowUpFlag,
		MaxFollowUps:     q.MaxFollowUps,
		Status:           q.Status,
		FollowUpsUsed:    q.FollowUpsUsed,
		NodePath:         q.NodePath,
		CorrectIndex:     q.CorrectIndex,
	}

	if q.ExpectedConcepts != nil {
		clone.ExpectedConcepts = append([]string(nil), q.ExpectedConcepts...)
	}
	if q.AnswerReceived != nil {
		val := *q.AnswerReceived
		clone.AnswerReceived = &val
	}
	if q.EvalScore != nil {
		val := *q.EvalScore
		clone.EvalScore = &val
	}
	if q.NodeLabelPath != nil {
		clone.NodeLabelPath = append([]string(nil), q.NodeLabelPath...)
	}
	if q.FollowUpTemplate != nil {
		clone.FollowUpTemplate = &FollowUpTemplate{
			ID:      q.FollowUpTemplate.ID,
			Trigger: q.FollowUpTemplate.Trigger,
			Text:    q.FollowUpTemplate.Text,
		}
	}
	if q.Persona != nil {
		val := *q.Persona
		clone.Persona = &val
	}
	if q.Options != nil {
		clone.Options = append([]string(nil), q.Options...)
	}
	if q.Feedbacks != nil {
		clone.Feedbacks = append([]string(nil), q.Feedbacks...)
	}

	return clone
}

func (s *SessionState) Clone() *SessionState {
	if s == nil {
		return nil
	}

	clone := &SessionState{
		ID:              s.ID,
		DomainID:        s.DomainID,
		FrameworkPrompt: s.FrameworkPrompt,
		AskedTotal:      s.AskedTotal,
		FollowUpDepth:   s.FollowUpDepth,
		LimitTotal:      s.LimitTotal,
	}

	if s.Pool != nil {
		clone.Pool = make([]*Question, len(s.Pool))
		for i, q := range s.Pool {
			clone.Pool[i] = q.Clone()
		}
	}

	if s.History != nil {
		clone.History = make([]HistoryEntry, len(s.History))
		for i, h := range s.History {
			clone.History[i] = HistoryEntry{
				Question: h.Question.Clone(),
				Answer:   h.Answer,
			}
			if h.Eval != nil {
				clone.History[i].Eval = &llm.EvalResult{
					Score:           h.Eval.Score,
					VagueFlag:       h.Eval.VagueFlag,
					ConceptsCovered: append([]string(nil), h.Eval.ConceptsCovered...),
					Missing:         append([]string(nil), h.Eval.Missing...),
					Reasoning:       h.Eval.Reasoning,
				}
			}
		}
	}

	if s.Coverage != nil {
		clone.Coverage = s.Coverage.Clone()
	}

	if s.CurrentQuestion != nil {
		clone.CurrentQuestion = s.CurrentQuestion.Clone()
	}

	if s.GenerationLog != nil {
		clone.GenerationLog = make([]GenerationLogEntry, len(s.GenerationLog))
		copy(clone.GenerationLog, s.GenerationLog)
	}

	return clone
}

type CoverageData struct {
	LeafScores   map[string]float64 `json:"leaf_scores"`
	LeafCounts   map[string]int     `json:"leaf_counts"`
	NodeWeights  map[string]float64 `json:"node_weights"`
}

func (c *CoverageData) Clone() *CoverageData {
	if c == nil {
		return nil
	}
	clone := &CoverageData{
		LeafScores:  make(map[string]float64, len(c.LeafScores)),
		LeafCounts:  make(map[string]int, len(c.LeafCounts)),
		NodeWeights: make(map[string]float64, len(c.NodeWeights)),
	}
	for k, v := range c.LeafScores {
		clone.LeafScores[k] = v
	}
	for k, v := range c.LeafCounts {
		clone.LeafCounts[k] = v
	}
	for k, v := range c.NodeWeights {
		clone.NodeWeights[k] = v
	}
	return clone
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