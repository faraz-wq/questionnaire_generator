package utils

import (
	"testing"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
)

func TestNextQuestionSelectorSelectsUnseen(t *testing.T) {
	pool := []*session.Question{
		{ID: "q1", Text: "Q1", NodePath: "a", NodeLabelPath: []string{"A"}, Status: session.StatusUnseen, Difficulty: "medium"},
		{ID: "q2", Text: "Q2", NodePath: "b", NodeLabelPath: []string{"B"}, Status: session.StatusAsked, Difficulty: "medium"},
		{ID: "q3", Text: "Q3", NodePath: "c", NodeLabelPath: []string{"C"}, Status: session.StatusUnseen, Difficulty: "medium"},
	}

	coverage := session.NewCoverage([]string{"a", "b", "c"})
	coverage.NodeWeights["a"] = 0.5
	coverage.NodeWeights["c"] = 1.0

	selector := NewNextQuestionSelector()
	selected := selector.Select(pool, coverage, 0)

	if selected == nil {
		t.Fatal("expected a question to be selected")
	}
	if selected.Status != session.StatusUnseen {
		t.Errorf("expected unseen question, got status %s", selected.Status)
	}
	if selected.ID == "q2" {
		t.Error("should not select already-asked question")
	}
}

func TestNextQuestionSelectorEmptyPool(t *testing.T) {
	pool := []*session.Question{
		{ID: "q1", Text: "Q1", NodePath: "a", Status: session.StatusAsked},
	}

	coverage := session.NewCoverage([]string{"a"})
	selector := NewNextQuestionSelector()
	selected := selector.Select(pool, coverage, 0)

	if selected != nil {
		t.Error("expected nil when no unseen questions")
	}
}

func TestNextQuestionSelectorSingleQuestion(t *testing.T) {
	pool := []*session.Question{
		{ID: "q1", Text: "Q1", NodePath: "a", NodeLabelPath: []string{"A"}, Status: session.StatusUnseen, Difficulty: "easy"},
	}

	coverage := session.NewCoverage([]string{"a"})
	selector := NewNextQuestionSelector()
	selected := selector.Select(pool, coverage, 0)

	if selected == nil {
		t.Fatal("expected a question to be selected")
	}
	if selected.ID != "q1" {
		t.Errorf("expected q1, got %s", selected.ID)
	}
}

func TestDifficultyRamp(t *testing.T) {
	selector := NewNextQuestionSelector()

	qEasy := &session.Question{Difficulty: "easy", NodePath: "a"}
	qHard := &session.Question{Difficulty: "hard", NodePath: "c"}

	easyEarly := selector.difficultyRamp(qEasy, 0)
	easyLate := selector.difficultyRamp(qEasy, 5)
	if easyEarly <= easyLate {
		t.Errorf("easy questions should be preferred early (got %.2f) vs late (got %.2f)", easyEarly, easyLate)
	}

	hardEarly := selector.difficultyRamp(qHard, 0)
	hardLate := selector.difficultyRamp(qHard, 5)
	if hardEarly >= hardLate {
		t.Errorf("hard questions should be deprioritized early (got %.2f) vs late (got %.2f)", hardEarly, hardLate)
	}
}

func TestUpdateCoverage(t *testing.T) {
	coverage := session.NewCoverage([]string{"a"})
	selector := NewNextQuestionSelector()

	q := &session.Question{ID: "q1", NodePath: "a", NodeLabelPath: []string{"A"}}

	selector.UpdateCoverage(coverage, q, 4)
	if coverage.LeafScores["a"] != 4 {
		t.Errorf("expected leaf score 4, got %.1f", coverage.LeafScores["a"])
	}
	if coverage.LeafCounts["a"] != 1 {
		t.Errorf("expected leaf count 1, got %d", coverage.LeafCounts["a"])
	}
}

func TestComputeLeafScoresSelector(t *testing.T) {
	history := []session.HistoryEntry{
		{
			Question: &session.Question{NodePath: "a"},
			Eval:     &llm.EvalResult{Score: 4},
		},
		{
			Question: &session.Question{NodePath: "a"},
			Eval:     &llm.EvalResult{Score: 2},
		},
		{
			Question: &session.Question{NodePath: "b"},
			Eval:     &llm.EvalResult{Score: 5},
		},
	}

	scores := ComputeLeafScores(history)
	a := scores["a"]
	if a.Total != 6 || a.Count != 2 {
		t.Errorf("expected total=6 count=2 for a, got total=%.1f count=%d", a.Total, a.Count)
	}
	b := scores["b"]
	if b.Total != 5 || b.Count != 1 {
		t.Errorf("expected total=5 count=1 for b, got total=%.1f count=%d", b.Total, b.Count)
	}
}

func TestComputeOverallSelector(t *testing.T) {
	scores := map[string]float64{"a": 3.5, "b": 4.0}
	overall := ComputeOverall(scores)
	if overall <= 0 || overall > 5 {
		t.Errorf("expected overall between 0 and 5, got %.2f", overall)
	}
}

func TestComputeOverallEmptySelector(t *testing.T) {
	overall := ComputeOverall(map[string]float64{})
	if overall != 0 {
		t.Errorf("expected 0 for empty scores, got %.2f", overall)
	}
}