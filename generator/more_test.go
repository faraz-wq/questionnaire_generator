package generator

import (
	"context"
	"strings"
	"testing"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/utils"
)

func TestBehaviouralGenerator(t *testing.T) {
	bg := NewBehaviouralGenerator(nil)

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:     "behavioural",
			Source:        "star",
			NodePath:      "test.behavioural",
			NodeLabelPath: []string{"Test", "Behavioural"},
			Count:         3,
		},
		Competencies: []string{"problem solving", "teamwork", "adaptability"},
	}

	questions, err := bg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}

	for i, q := range questions {
		if q.Archetype != "behavioural" {
			t.Errorf("question %d: expected archetype behavioural, got %s", i, q.Archetype)
		}
		if q.Text == "" {
			t.Errorf("question %d: expected non-empty text", i)
		}
		if q.FollowUpFlag != true {
			t.Errorf("question %d: expected follow_up_flag to be true", i)
		}
	}
}

func TestBehaviouralGeneratorEmptyCompetencies(t *testing.T) {
	bg := NewBehaviouralGenerator(nil)

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:     "behavioural",
			Source:        "star",
			NodePath:      "test.behavioural",
			NodeLabelPath: []string{"Test"},
			Count:         1,
		},
	}

	questions, err := bg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error with empty competencies, got %v", err)
	}

	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
}

func TestCaseGenerator(t *testing.T) {
	mockClient := llm.NewMockLLMClient()
	cg := NewCaseGenerator(mockClient)

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:        "case",
			Source:           "free_llm",
			NodePath:         "test.case",
			NodeLabelPath:    []string{"Test", "Case"},
			Count:            1,
			FrameworkPrompt:  "You are a test interviewer",
			KnowledgeContext: "Test context",
		},
	}

	questions, err := cg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}

	if questions[0].Archetype != "case" {
		t.Errorf("expected archetype case (from task), got %s", questions[0].Archetype)
	}
}

func TestBehaviouralGeneratorRealEstateFallback(t *testing.T) {
	bg := NewBehaviouralGenerator(nil)

	input := GeneratorInput{
		Task: utils.GeneratorTask{
			Archetype:     "behavioural",
			Source:        "star",
			NodePath:      "test.behavioural",
			NodeLabelPath: []string{"Test", "Behavioural"},
			Count:         2,
		},
		Competencies: []string{"handling customer objections", "negotiating property deals"},
	}

	questions, err := bg.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}

	expectedTexts := []string{
		"Tell me about a challenging situation where you had to handle tough customer objections regarding a property's pricing",
		"Describe a complex property deal negotiation where there was a major gap between the buyer's budget and the seller's expectations",
	}

	for i, q := range questions {
		if !strings.Contains(q.Text, expectedTexts[i]) {
			t.Errorf("question %d: expected text to contain %q, got %q", i, expectedTexts[i], q.Text)
		}
	}
}