package utils

import (
	"testing"
)

func TestComputeOverallUtil(t *testing.T) {
	scores := map[string]float64{"a": 3.5, "b": 4.0}
	overall := ComputeOverall(scores)
	if overall <= 0 || overall > 5 {
		t.Errorf("expected overall between 0 and 5, got %.2f", overall)
	}
}

func TestComputeOverallEmptyUtil(t *testing.T) {
	overall := ComputeOverall(map[string]float64{})
	if overall != 0 {
		t.Errorf("expected 0 for empty scores, got %.2f", overall)
	}
}