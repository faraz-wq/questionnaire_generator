package followup

import (
	"context"
	"testing"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"go.uber.org/zap"
)

func TestFollowUpRouterScoreLow(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
	}

	evalResult := &llm.EvalResult{
		Score:      2,
		VagueFlag:  false,
		Reasoning:  "Poor answer",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !decision.Fire {
		t.Fatal("expected follow-up to fire for low score")
	}
	if decision.Question == nil {
		t.Fatal("expected follow-up question to be non-nil")
	}
	if decision.Question.Archetype != "follow_up" {
		t.Errorf("expected archetype follow_up, got %s", decision.Question.Archetype)
	}
}

func TestFollowUpRouterVagueFlag(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
	}

	evalResult := &llm.EvalResult{
		Score:      4,
		VagueFlag:  true,
		Reasoning:  "Vague answer",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !decision.Fire {
		t.Fatal("expected follow-up to fire for vague flag")
	}
}

func TestFollowUpRouterAuthoredTemplate(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
		FollowUpTemplate: &session.FollowUpTemplate{
			ID:      "t1",
			Trigger: "score_low",
			Text:    "Can you explain your reasoning?",
		},
	}

	evalResult := &llm.EvalResult{
		Score:      2,
		VagueFlag:  false,
		Reasoning:  "Poor",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !decision.Fire {
		t.Fatal("expected follow-up to fire")
	}
	if decision.Question.Text != "Can you explain your reasoning?" {
		t.Errorf("expected authored template text, got %q", decision.Question.Text)
	}
}

func TestFollowUpRouterMaxDepth(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
	}

	evalResult := &llm.EvalResult{
		Score:      1,
		VagueFlag:  true,
		Reasoning:  "Terrible",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if decision.Fire {
		t.Fatal("expected follow-up NOT to fire at max depth")
	}
}

func TestFollowUpRouterNoFire(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
		FollowUpFlag:     false,
		MaxFollowUps:     0,
		FollowUpsUsed:    0,
	}

	evalResult := &llm.EvalResult{
		Score:      4,
		VagueFlag:  false,
		Reasoning:  "Great answer",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if decision.Fire {
		t.Fatal("expected follow-up NOT to fire for good answer")
	}
}

func TestFollowUpRouterLLMFallback(t *testing.T) {
	logger := zap.NewNop()
	mockClient := &llm.MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string, opts llm.GenerationOptions) (string, error) {
			return "", context.DeadlineExceeded
		},
	}
	router := NewFollowUpRouter(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		NodePath:         "test",
		NodeLabelPath:    []string{"Test"},
		ExpectedConcepts: []string{"a", "b", "c"},
	}

	evalResult := &llm.EvalResult{
		Score:      2,
		VagueFlag:  false,
		Reasoning:  "Poor",
	}

	decision, err := router.Route(context.Background(), question, evalResult, 0)
	if err != nil {
		t.Fatalf("expected no error on LLM failure, got %v", err)
	}

	if decision.Fire {
		t.Fatal("expected follow-up NOT to fire when LLM fails and no template")
	}
}