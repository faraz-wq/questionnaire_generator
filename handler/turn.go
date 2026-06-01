package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/faraz/questionnaire_generator/session"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	evalResult, err := h.evaluator.Evaluate(ctx, state.FrameworkPrompt, currentQ, req.Answer, state.History)
	if err != nil {
		h.logger.Error("evaluation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "evaluation failed"})
		return
	}

	score := evalResult.Score
	currentQ.EvalScore = &score
	currentQ.FollowUpsUsed++

	state.History = append(state.History, session.HistoryEntry{
		Question: currentQ,
		Answer:   req.Answer,
		Eval:     evalResult,
	})

	h.selector.UpdateCoverage(state.Coverage, currentQ, score)

	fupCtx, fupCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer fupCancel()
	followUpDecision, err := h.followUpRouter.Route(fupCtx, currentQ, evalResult, state.FollowUpDepth)
	if err != nil {
		h.logger.Warn("follow-up routing failed", zap.Error(err))
	}

	var nextQ *session.Question
	followUpFired := false

	if followUpDecision != nil && followUpDecision.Fire && followUpDecision.Question != nil {
		nextQ = followUpDecision.Question
		followUpFired = true
		state.FollowUpDepth++
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