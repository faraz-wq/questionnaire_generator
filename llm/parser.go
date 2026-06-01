package llm

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

type Parser struct {
	logger *zap.Logger
}

func NewParser(logger *zap.Logger) *Parser {
	return &Parser{logger: logger}
}

func (p *Parser) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	cleaned := p.stripMarkdownCodeFences(raw)

	var questions []*QuestionJSON
	if err := json.Unmarshal([]byte(cleaned), &questions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
	}

	for i, q := range questions {
		if err := p.validateQuestion(q); err != nil {
			return nil, fmt.Errorf("question %d validation failed: %w", i, err)
		}
	}

	return questions, nil
}

func (p *Parser) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	cleaned := p.stripMarkdownCodeFences(raw)

	var result EvalResultJSON
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal eval result: %w", err)
	}

	if err := p.validateEvalResult(&result); err != nil {
		return nil, fmt.Errorf("eval result validation failed: %w", err)
	}

	return &result, nil
}

func (p *Parser) stripMarkdownCodeFences(input string) string {
	if len(input) < 6 {
		return input
	}

	if input[0] == '`' && input[1] == '`' && input[2] == '`' {
		for i := 3; i < len(input); i++ {
			if input[i] == '\n' {
				start := i + 1
				if len(input) > start+3 && input[len(input)-3] == '`' && input[len(input)-2] == '`' && input[len(input)-1] == '`' {
					end := len(input) - 3
					if end > 0 && input[end-1] == '\n' {
						end--
					}
					return input[start:end]
				}
				return input[start:]
			}
		}
	}

	return input
}

func (p *Parser) validateQuestion(q *QuestionJSON) error {
	if q.ID == "" {
		return fmt.Errorf("id is required")
	}
	if q.Text == "" {
		return fmt.Errorf("text is required")
	}
	validArchetypes := map[string]bool{
		"knowledge": true, "reasoning": true, "situational": true,
		"behavioural": true, "case": true, "follow_up": true,
		"applied": true, "scenario": true, "conceptual": true,
		"aptitude": true, "applied judgment": true,
	}
	if !validArchetypes[q.Archetype] {
		return fmt.Errorf("invalid archetype: %q", q.Archetype)
	}
	validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true}
	if !validDifficulties[q.Difficulty] {
		return fmt.Errorf("invalid difficulty: %q", q.Difficulty)
	}
	validTypes := map[string]bool{"open": true, "scenario": true, "factual": true, "knowledge": true, "reasoning": true}
	if !validTypes[q.Type] {
		return fmt.Errorf("invalid type: %q", q.Type)
	}
	if q.IdealAnswerHint == "" {
		return fmt.Errorf("ideal_answer_hint is required")
	}
	if len(q.ExpectedConcepts) < 1 || len(q.ExpectedConcepts) > 10 {
		return fmt.Errorf("expected_concepts must have 1-10 items, got %d", len(q.ExpectedConcepts))
	}
	if q.MaxFollowUps < 0 || q.MaxFollowUps > 5 {
		return fmt.Errorf("max_follow_ups must be 0-5, got %d", q.MaxFollowUps)
	}
	return nil
}

func (p *Parser) validateEvalResult(e *EvalResultJSON) error {
	if e.Score < 1 || e.Score > 5 {
		return fmt.Errorf("score must be 1-5, got %d", e.Score)
	}
	if e.Reasoning == "" {
		return fmt.Errorf("reasoning is required")
	}
	if e.BehavioralScores != nil {
		for dimID, score := range e.BehavioralScores {
			if score < 1 || score > 5 {
				return fmt.Errorf("behavioral score for %s must be 1-5, got %d", dimID, score)
			}
		}
	}
	return nil
}