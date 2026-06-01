package handler

import (
	"net/http"
	"strings"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/gin-gonic/gin"
)

type TurnRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type TurnResponse struct {
	SessionID       string              `json:"session_id"`
	EvalResult      interface{}         `json:"eval_result"`
	NextQuestion    *session.Question   `json:"next_question"`
	FollowUpFired   bool                `json:"follow_up_fired"`
	FollowUpDepth   int                 `json:"follow_up_depth"`
	AskedTotal      int                 `json:"asked_total"`
	InterviewDone   bool                `json:"interview_done"`
}

func (h *Handler) ProcessTurn(c *gin.Context) {
	sessionID := c.Param("id")

	state, exists := h.sessionManager.Get(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	var req TurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "answer is required"})
		return
	}

	currentQ := state.CurrentQuestion
	if currentQ == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no current question"})
		return
	}

	currentQ.AnswerReceived = &req.Answer
	currentQ.Status = session.StatusAsked

	chosenIndex := -1
	for idx, opt := range currentQ.Options {
		if opt == req.Answer {
			chosenIndex = idx
			break
		}
	}

	score := 1
	conceptsCovered := []string{}
	missingConcepts := currentQ.ExpectedConcepts

	if chosenIndex == currentQ.CorrectIndex {
		score = 5
		conceptsCovered = currentQ.ExpectedConcepts
		missingConcepts = []string{}
	}

	// Format evaluator reasoning to only say Correct/Incorrect and have explanation of the correct answer as a statement
	correctFeedback := ""
	if currentQ.CorrectIndex >= 0 && currentQ.CorrectIndex < len(currentQ.Feedbacks) {
		correctFeedback = currentQ.Feedbacks[currentQ.CorrectIndex]
	}

	explanation := cleanExplanation(correctFeedback)
	if explanation == "" {
		explanation = "This choice represents the most compliant, accurate, and professional response."
	}

	prefix := "Incorrect. "
	if chosenIndex == currentQ.CorrectIndex {
		prefix = "Correct. "
	}
	reasoning := prefix + explanation

	evalResult := &llm.EvalResult{
		Score:           score,
		VagueFlag:       false,
		ConceptsCovered: conceptsCovered,
		Missing:         missingConcepts,
		Reasoning:       reasoning,
	}

	currentQ.EvalScore = &score
	currentQ.FollowUpsUsed++

	state.History = append(state.History, session.HistoryEntry{
		Question: currentQ,
		Answer:   req.Answer,
		Eval:     evalResult,
	})

	h.selector.UpdateCoverage(state.Coverage, currentQ, score)

	var nextQ *session.Question
	followUpFired := false

	if state.AskedTotal >= 6 {
		nextQ = nil
		state.FollowUpDepth = 0
	} else {
		nextQ = h.selector.Select(state.Pool, state.Coverage, state.AskedTotal)
		state.FollowUpDepth = 0
	}

	state.CurrentQuestion = nextQ
	if nextQ != nil {
		nextQ.Status = session.StatusAsked
		state.AskedTotal++
	}

	h.sessionManager.Update(sessionID, state)

	interviewDone := nextQ == nil

	evalResponse := map[string]interface{}{
		"score":            evalResult.Score,
		"vague_flag":       evalResult.VagueFlag,
		"concepts_covered": evalResult.ConceptsCovered,
		"missing":          evalResult.Missing,
		"reasoning":        evalResult.Reasoning,
		"feedbacks":        currentQ.Feedbacks,
		"correct_index":    currentQ.CorrectIndex,
	}

	c.JSON(http.StatusOK, TurnResponse{
		SessionID:     sessionID,
		EvalResult:    evalResponse,
		NextQuestion:  nextQ,
		FollowUpFired: followUpFired,
		FollowUpDepth: state.FollowUpDepth,
		AskedTotal:    state.AskedTotal,
		InterviewDone: interviewDone,
	})
}

func cleanExplanation(fb string) string {
	fb = strings.TrimSpace(fb)
	
	prefixes := []string{
		"correct!",
		"correct.",
		"correct :",
		"correct:",
		"correct response!",
		"correct response:",
		"correct response.",
		"correct response",
		"exceptional response!",
		"exceptional response:",
		"exceptional response.",
		"exceptional:",
		"exceptional!",
	}
	
	fbLower := strings.ToLower(fb)
	for _, p := range prefixes {
		if strings.HasPrefix(fbLower, p) {
			fb = fb[len(p):]
			fb = strings.TrimSpace(fb)
			fb = strings.TrimLeft(fb, "!.:- ")
			break
		}
	}
	
	if len(fb) == 0 {
		return ""
	}
	
	runes := []rune(fb)
	if len(runes) > 0 {
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		fb = string(runes)
	}
	
	return fb
}