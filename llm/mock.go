package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type MockLLMClient struct {
	GenerateFunc func(ctx context.Context, prompt string, opts GenerationOptions) (string, error)
	EvaluateFunc func(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error)
	parser       *Parser
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		parser: NewParser(nil),
	}
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt, opts)
	}

	topic := topicFromPrompt(prompt)
	questionsJSON := fmt.Sprintf(`[
		{
			"id": "q1",
			"text": "Mock question about %s?",
			"archetype": "knowledge",
			"difficulty": "medium",
			"type": "open",
			"ideal_answer_hint": "Explain clearly",
			"expected_concepts": ["concept1", "concept2", "concept3"],
			"follow_up_flag": false,
			"max_follow_ups": 1
		}
	]`, topic)

	return questionsJSON, nil
}

func (m *MockLLMClient) Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error) {
	if m.EvaluateFunc != nil {
		return m.EvaluateFunc(ctx, prompt, opts)
	}

	return &EvalResult{
		Score:           4,
		VagueFlag:       false,
		ConceptsCovered: []string{"concept1", "concept2"},
		Missing:         []string{"concept3"},
		Reasoning:       "Good answer, minor omissions.",
	}, nil
}

func (m *MockLLMClient) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	var questions []*QuestionJSON
	if err := json.Unmarshal([]byte(raw), &questions); err != nil {
		return nil, fmt.Errorf("parse questions: %w", err)
	}
	return questions, nil
}

func (m *MockLLMClient) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	var result EvalResultJSON
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse eval result: %w", err)
	}
	return &result, nil
}

func topicFromPrompt(prompt string) string {
	lines := strings.Split(prompt, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Topic:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Topic:"))
		}
	}
	return "unknown"
}