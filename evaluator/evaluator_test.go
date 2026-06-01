package evaluator

import (
	"context"
	"testing"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"go.uber.org/zap"
)

func TestEvaluatorSuccess(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	eval := NewEvaluator(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		ExpectedConcepts: []string{"goroutines", "channels", "interfaces"},
		NodePath:         "test.tech",
		NodeLabelPath:    []string{"Test", "Tech"},
	}

	result, err := eval.Evaluate(context.Background(), "You are an interviewer", question, "Go is a language with goroutines and channels.", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Score != 4 {
		t.Errorf("expected score 4, got %d", result.Score)
	}
	if result.VagueFlag != false {
		t.Errorf("expected vague_flag false, got %v", result.VagueFlag)
	}
}

func TestEvaluatorFallbackOnLLMFailure(t *testing.T) {
	logger := zap.NewNop()
	mockClient := &llm.MockLLMClient{
		EvaluateFunc: func(ctx context.Context, prompt string, opts llm.GenerationOptions) (*llm.EvalResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	eval := NewEvaluator(mockClient, logger)

	question := &session.Question{
		ID:               "q1",
		Text:             "What is Go?",
		ExpectedConcepts: []string{"goroutines", "channels", "interfaces"},
		NodePath:         "test.tech",
		NodeLabelPath:    []string{"Test"},
	}

	result, err := eval.Evaluate(context.Background(), "You are an interviewer", question, "An answer.", nil)
	if err != nil {
		t.Fatalf("expected no error (fallback), got %v", err)
	}

	if result.Score != 3 {
		t.Errorf("expected fallback score 3, got %d", result.Score)
	}
}

func TestBuildEvaluationPrompt(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	eval := NewEvaluator(mockClient, logger)

	question := &session.Question{
		Text:             "Test question?",
		ExpectedConcepts: []string{"a", "b", "c"},
	}

	prompt := eval.buildEvaluationPrompt(
		"You are an interviewer",
		question.Text,
		question.ExpectedConcepts,
		"Test answer",
		[]session.HistoryEntry{
			{
				Question: &session.Question{Text: "Previous Q"},
				Answer:   "Previous A",
			},
		},
		nil,
	)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}

func TestEvaluatorWithHistory(t *testing.T) {
	logger := zap.NewNop()
	mockClient := llm.NewMockLLMClient()
	eval := NewEvaluator(mockClient, logger)

	question := &session.Question{
		ID:               "q2",
		Text:             "What is a goroutine?",
		ExpectedConcepts: []string{"lightweight thread", "concurrency", "scheduler"},
		NodePath:         "test.tech",
		NodeLabelPath:    []string{"Test"},
	}

	history := []session.HistoryEntry{
		{
			Question: &session.Question{Text: "What is Go?"},
			Answer:   "A programming language.",
			Eval:     &llm.EvalResult{Score: 3},
		},
	}

	result, err := eval.Evaluate(context.Background(), "You are an interviewer", question, "Lightweight threads managed by Go runtime.", history)
	if err != nil {
		t.Fatalf("expected no error with history: %v", err)
	}

	if result.Score < 1 || result.Score > 5 {
		t.Errorf("expected score between 1 and 5, got %d", result.Score)
	}
}