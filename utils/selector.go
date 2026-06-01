package utils

import (
	"math"
	"math/rand"

	"github.com/faraz/questionnaire_generator/session"
)

type NextQuestionSelector struct{}

func NewNextQuestionSelector() *NextQuestionSelector {
	return &NextQuestionSelector{}
}

func (s *NextQuestionSelector) Select(pool []*session.Question, coverage *session.CoverageData, askedTotal int) *session.Question {
	var unseen []*session.Question
	for _, q := range pool {
		if q.Status == session.StatusUnseen {
			unseen = append(unseen, q)
		}
	}

	if len(unseen) == 0 {
		return nil
	}

	if len(unseen) == 1 {
		return unseen[0]
	}

	for _, q := range unseen {
		score := s.leafScore(q, coverage)
		ramp := s.difficultyRamp(q, askedTotal)
		adjustedScore := score * ramp * (1 + rand.Float64()*0.1)
		_ = adjustedScore
	}

	type candidate struct {
		q     *session.Question
		score float64
	}

	var candidates []candidate
	for _, q := range unseen {
		score := s.leafScore(q, coverage)
		ramp := s.difficultyRamp(q, askedTotal)
		adjusted := score * ramp * (1 + rand.Float64()*0.05)
		candidates = append(candidates, candidate{q: q, score: adjusted})
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return best.q
}

func (s *NextQuestionSelector) leafScore(q *session.Question, coverage *session.CoverageData) float64 {
	if coverage == nil || coverage.LeafScores == nil {
		return 0.5
	}

	currentScore, ok := coverage.LeafScores[q.NodePath]
	if !ok {
		return 0.5
	}

	total, ok := coverage.LeafCounts[q.NodePath]
	if !ok || total == 0 {
		return 0.5
	}

	avgScore := currentScore / float64(total)

	weight, ok := coverage.NodeWeights[q.NodePath]
	if !ok {
		weight = 0.5
	}

	return (1.0 - avgScore/5.0) * weight
}

func (s *NextQuestionSelector) difficultyRamp(q *session.Question, askedTotal int) float64 {
	switch q.Difficulty {
	case "easy":
		if askedTotal < 2 {
			return 1.5
		}
		return 0.5
	case "medium":
		if askedTotal >= 2 && askedTotal < 6 {
			return 1.3
		}
		return 1.0
	case "hard":
		if askedTotal >= 4 {
			return 1.3
		}
		return 0.7
	default:
		return 1.0
	}
}

func (s *NextQuestionSelector) UpdateCoverage(coverage *session.CoverageData, question *session.Question, score int) {
	if coverage == nil {
		return
	}

	coverage.LeafScores[question.NodePath] += float64(score)
	coverage.LeafCounts[question.NodePath]++

	if question.EvalScore != nil {
		coverage.LeafScores[question.NodePath] += float64(*question.EvalScore)
		coverage.LeafCounts[question.NodePath]++
	}
}

func ComputeLeafScores(history []session.HistoryEntry) map[string]struct {
	Total float64
	Count int
} {
	result := make(map[string]struct {
		Total float64
		Count int
	})
	for _, h := range history {
		if h.Eval == nil {
			continue
		}
		nodePath := h.Question.NodePath
		entry := result[nodePath]
		entry.Total += float64(h.Eval.Score)
		entry.Count++
		result[nodePath] = entry
	}
	return result
}

func ComputeCategoryScores(leafScores map[string]struct {
	Total float64
	Count int
}, nodeWeights map[string]float64) map[string]float64 {
	categories := make(map[string]float64)
	for path, data := range leafScores {
		if data.Count == 0 {
			continue
		}
		avg := data.Total / float64(data.Count)
		weight := 0.5
		if w, ok := nodeWeights[path]; ok {
			weight = w
		}
		categories[path] = math.Round(avg*weight*100) / 100
	}
	return categories
}

func ComputeOverall(categoryScores map[string]float64) float64 {
	if len(categoryScores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range categoryScores {
		sum += s
	}
	return math.Round(sum/float64(len(categoryScores))*100) / 100
}