package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SummaryResponse struct {
	SessionID  string                 `json:"session_id"`
	LeafScores map[string]interface{} `json:"leaf_scores"`
	Overall    float64                `json:"overall"`
	Transcript []TranscriptEntry      `json:"transcript"`
	Narrative  string                 `json:"narrative,omitempty"`
}

type TranscriptEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Score    int    `json:"score"`
}

func (h *Handler) GetSummary(c *gin.Context) {
	sessionID := c.Param("id")

	state, exists := h.sessionManager.Get(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	includeNarrative := c.Query("include_narrative") == "true"

	leafScores := utils.ComputeLeafScores(state.History)

	leafScoreMap := make(map[string]interface{})
	for path, data := range leafScores {
		if data.Count > 0 {
			leafScoreMap[path] = map[string]interface{}{
				"average": fmt.Sprintf("%.2f", data.Total/float64(data.Count)),
				"count":   data.Count,
			}
		}
	}

	overall := 0.0
	var sum float64
	var count int
	for _, data := range leafScores {
		if data.Count > 0 {
			sum += data.Total / float64(data.Count)
			count++
		}
	}
	if count > 0 {
		overall = sum / float64(count)
	}

	var transcript []TranscriptEntry
	for _, h := range state.History {
		score := 0
		if h.Eval != nil {
			score = h.Eval.Score
		}
		transcript = append(transcript, TranscriptEntry{
			Question: h.Question.Text,
			Answer:   h.Answer,
			Score:    score,
		})
	}

	resp := SummaryResponse{
		SessionID:  sessionID,
		LeafScores: leafScoreMap,
		Overall:    overall,
		Transcript: transcript,
	}

	if includeNarrative && len(transcript) > 0 {
		narrative, err := h.generateNarrative(state, transcript, overall, leafScoreMap)
		if err != nil {
			h.logger.Warn("narrative generation failed", zap.Error(err))
		} else {
			resp.Narrative = narrative
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) generateNarrative(state interface{}, transcript []TranscriptEntry, overall float64, leafScores map[string]interface{}) (string, error) {
	var sb strings.Builder
	sb.WriteString("Interview Transcript:\n\n")
	for i, t := range transcript {
		sb.WriteString(fmt.Sprintf("Q%d: %s\n", i+1, t.Question))
		sb.WriteString(fmt.Sprintf("A%d (score %d/5): %s\n\n", i+1, t.Score, t.Answer))
	}

	summaryStr := ""
	for path, data := range leafScores {
		summaryStr += fmt.Sprintf("  - %s: %v\n", path, data)
	}

	prompt := fmt.Sprintf(`Based on this interview transcript and scores, write a 2-3 paragraph summary assessing the candidate.
Overall score: %.2f/5

Category scores:
%s

%s

Provide a concise narrative assessment of the candidate's strengths, weaknesses, and overall suitability.
Return ONLY the narrative text, no JSON, no markdown.`, overall, summaryStr, sb.String())

	st, _ := state.(interface{ GetFrameworkPrompt() string })
	_ = st

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	raw, err := h.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   500,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(raw), nil
}