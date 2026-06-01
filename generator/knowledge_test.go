package generator

import (
	"context"
	"testing"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/utils"
)

func TestKnowledgeGenerator(t *testing.T) {
	mockClient := llm.NewMockLLMClient()
	kg := NewKnowledgeGenerator(mockClient, 2000)

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:        "knowledge",
			Source:           "kb_prompt",
			NodePath:         "test.leaf",
			NodeLabelPath:    []string{"Test", "Leaf"},
			Count:            1,
			GenerationPrompt: "Test prompt",
			FrameworkPrompt:  "You are a test interviewer",
			KnowledgeContext: "Test knowledge context",
		},
	}

	questions, err := kg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}

	q := questions[0]
	if q.Archetype != "knowledge" {
		t.Errorf("expected archetype knowledge, got %s", q.Archetype)
	}
	if q.Status != "unseen" {
		t.Errorf("expected status unseen, got %s", q.Status)
	}
	if q.NodePath != "test.leaf" {
		t.Errorf("expected node_path test.leaf, got %s", q.NodePath)
	}
}

func TestKnowledgeGeneratorTruncation(t *testing.T) {
	mockClient := llm.NewMockLLMClient()
	kg := NewKnowledgeGenerator(mockClient, 2) // Very small limit

	longContext := "This is a very long context that will definitely need truncation because it contains far more characters than can fit within the tiny token limit we have set for this test case."
	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:        "knowledge",
			Source:           "kb_prompt",
			NodePath:         "test.leaf",
			NodeLabelPath:    []string{"Test"},
			Count:            1,
			GenerationPrompt: "Test",
			FrameworkPrompt:  "Test",
			KnowledgeContext: longContext,
		},
	}

	questions, err := kg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error despite truncation: %v", err)
	}

	if len(questions) != 1 {
		t.Fatalf("expected 1 question after truncation, got %d", len(questions))
	}
}

func TestDispatcherFound(t *testing.T) {
	dispatcher := NewDispatcher()
	mockClient := llm.NewMockLLMClient()
	dispatcher.Register("KnowledgeGenerator", NewKnowledgeGenerator(mockClient, 2000))

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:        "knowledge",
			Source:           "kb_prompt",
			NodePath:         "test.leaf",
			NodeLabelPath:    []string{"Test"},
			Count:            1,
			GenerationPrompt: "Test",
			FrameworkPrompt:  "Test",
			KnowledgeContext: "Test",
		},
	}

	questions, err := dispatcher.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(questions) == 0 {
		t.Fatal("expected at least 1 question")
	}
}

func TestDispatcherNotFound(t *testing.T) {
	dispatcher := NewDispatcher()

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype: "knowledge",
			Source:    "unknown_source",
			NodePath:  "test.leaf",
			Count:     1,
		},
	}

	_, err := dispatcher.Generate(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing generator")
	}
}

func TestDispatcherDefaultSourceMapping(t *testing.T) {
	tests := []struct {
		source         string
		expectedMapper string
	}{
		{"kb_prompt", "KnowledgeGenerator"},
		{"parametric", "ReasoningGenerator"},
		{"slot_fill", "SituationalGenerator"},
		{"star", "BehaviouralGenerator"},
		{"free_llm", "CaseGenerator"},
		{"unknown", "KnowledgeGenerator"},
	}

	for _, tt := range tests {
		result := DefaultArchetypeMapping(tt.source)
		if result != tt.expectedMapper {
			t.Errorf("DefaultArchetypeMapping(%q) = %q, want %q", tt.source, result, tt.expectedMapper)
		}
	}
}