package llm

import (
	"testing"
)

func TestValidateQuestionValid(t *testing.T) {
	p := NewParser(nil)

	q := &QuestionJSON{
		ID:               "q1",
		Text:             "What is Go?",
		Archetype:        "knowledge",
		Difficulty:       "medium",
		Type:             "open",
		IdealAnswerHint:  "A good answer covers concurrency and simplicity",
		ExpectedConcepts: []string{"goroutines", "channels", "interfaces", "static typing"},
		FollowUpFlag:     false,
		MaxFollowUps:     1,
	}

	err := p.validateQuestion(q)
	if err != nil {
		t.Errorf("expected valid question to pass: %v", err)
	}
}

func TestValidateQuestionInvalidArchetype(t *testing.T) {
	p := NewParser(nil)

	q := &QuestionJSON{
		ID:               "q1",
		Text:             "What is Go?",
		Archetype:        "invalid_type",
		Difficulty:       "medium",
		Type:             "open",
		IdealAnswerHint:  "hint",
		ExpectedConcepts: []string{"a", "b", "c"},
		FollowUpFlag:     false,
		MaxFollowUps:     1,
	}

	err := p.validateQuestion(q)
	if err == nil {
		t.Fatal("expected error for invalid archetype")
	}
}

func TestValidateQuestionTooFewConcepts(t *testing.T) {
	p := NewParser(nil)

	q := &QuestionJSON{
		ID:               "q1",
		Text:             "What is Go?",
		Archetype:        "knowledge",
		Difficulty:       "medium",
		Type:             "open",
		IdealAnswerHint:  "hint",
		ExpectedConcepts: []string{},
		FollowUpFlag:     false,
		MaxFollowUps:     1,
	}

	err := p.validateQuestion(q)
	if err == nil {
		t.Fatal("expected error for too few expected_concepts")
	}
}

func TestValidateQuestionMaxFollowUpsZero(t *testing.T) {
	p := NewParser(nil)

	q := &QuestionJSON{
		ID:               "q1",
		Text:             "What is Go?",
		Archetype:        "knowledge",
		Difficulty:       "medium",
		Type:             "open",
		IdealAnswerHint:  "hint",
		ExpectedConcepts: []string{"a", "b", "c"},
		FollowUpFlag:     false,
		MaxFollowUps:     0,
	}

	err := p.validateQuestion(q)
	if err != nil {
		t.Errorf("expected max_follow_ups=0 to pass: %v", err)
	}
}

func TestValidateQuestionTooManyConcepts(t *testing.T) {
	p := NewParser(nil)

	q := &QuestionJSON{
		ID:               "q1",
		Text:             "What is Go?",
		Archetype:        "knowledge",
		Difficulty:       "medium",
		Type:             "open",
		IdealAnswerHint:  "hint",
		ExpectedConcepts: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
		FollowUpFlag:     false,
		MaxFollowUps:     1,
	}

	err := p.validateQuestion(q)
	if err == nil {
		t.Fatal("expected error for too many expected_concepts")
	}
}

func TestValidateEvalResult(t *testing.T) {
	p := NewParser(nil)

	e := &EvalResultJSON{
		Score:           4,
		VagueFlag:       false,
		ConceptsCovered: []string{"goroutines", "channels"},
		Missing:         []string{"interfaces"},
		Reasoning:       "Good answer but missed interfaces.",
	}

	err := p.validateEvalResult(e)
	if err != nil {
		t.Errorf("expected valid eval result to pass: %v", err)
	}
}

func TestValidateEvalResultInvalidScore(t *testing.T) {
	p := NewParser(nil)

	e := &EvalResultJSON{
		Score:     6,
		VagueFlag: false,
		Reasoning: "test",
	}

	err := p.validateEvalResult(e)
	if err == nil {
		t.Fatal("expected error for score > 5")
	}
}

func TestStripMarkdownCodeFences(t *testing.T) {
	p := NewParser(nil)

	input := "```json\n[{\"id\": \"1\"}]\n```"
	result := p.stripMarkdownCodeFences(input)
	if result != "[{\"id\": \"1\"}]" {
		t.Errorf("expected '[{\"id\": \"1\"}]', got %q", result)
	}
}

func TestStripMarkdownCodeFencesNoFences(t *testing.T) {
	p := NewParser(nil)

	input := "plain text"
	result := p.stripMarkdownCodeFences(input)
	if result != "plain text" {
		t.Errorf("expected 'plain text', got %q", result)
	}
}

func TestParseQuestionsInvalidJSON(t *testing.T) {
	p := NewParser(nil)

	_, err := p.ParseQuestions("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseEvalResultInvalidJSON(t *testing.T) {
	p := NewParser(nil)

	_, err := p.ParseEvalResult("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}