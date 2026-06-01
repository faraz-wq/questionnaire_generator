package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/faraz/questionnaire_generator/domain"
	"github.com/faraz/questionnaire_generator/evaluator"
	"github.com/faraz/questionnaire_generator/followup"
	"github.com/faraz/questionnaire_generator/generator"
	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/faraz/questionnaire_generator/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	sessionManager *session.SessionManager
	dispatcher     *generator.Dispatcher
	evaluator      *evaluator.Evaluator
	followUpRouter *followup.FollowUpRouter
	selector       *utils.NextQuestionSelector
	client         llm.LLMClient
	logger         *zap.Logger
}

func NewHandler(
	sessionManager *session.SessionManager,
	dispatcher *generator.Dispatcher,
	evaluator *evaluator.Evaluator,
	followUpRouter *followup.FollowUpRouter,
	selector *utils.NextQuestionSelector,
	client llm.LLMClient,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		dispatcher:     dispatcher,
		evaluator:      evaluator,
		followUpRouter: followUpRouter,
		selector:       selector,
		client:         client,
		logger:         logger,
	}
}

type InitRequest struct {
	DomainConfigPath string         `json:"domain_config_path" binding:"required"`
	ArchetypeCounts  map[string]int `json:"archetype_counts"`
}

type InitResponse struct {
	SessionID     string                       `json:"session_id"`
	Pool          []*session.Question          `json:"pool"`
	FirstQuestion *session.Question            `json:"first_question"`
	GenerationLog []session.GenerationLogEntry `json:"generation_log"`
}

func (h *Handler) InitSession(c *gin.Context) {
	var req InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain_config_path is required"})
		return
	}

	cfg, err := domain.LoadDomainConfig(req.DomainConfigPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build a map for quick persona lookup if needed by generators (optional)
	var personaMap map[string]*domain.Persona
	if len(cfg.Personas) > 0 {
		personaMap = make(map[string]*domain.Persona, len(cfg.Personas))
		for i := range cfg.Personas {
			p := &cfg.Personas[i]
			personaMap[p.ID] = p
		}
	}

	tasks := utils.WalkTree(cfg.Taxonomy, cfg.FrameworkPrompt, cfg.KnowledgeContext, cfg, req.ArchetypeCounts)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	type result struct {
		questions []*session.Question
		err       error
		task      utils.GeneratorTask
	}

	resultCh := make(chan result, len(tasks))
	sem := make(chan struct{}, 3)
	for _, task := range tasks {
		sem <- struct{}{}
		go func(t utils.GeneratorTask) {
			defer func() { <-sem }()
			questions, err := h.dispatcher.Generate(ctx, generator.GeneratorInput{
				Task:             t,
				Competencies:     cfg.Competencies,
				SituationalSlots: cfg.SituationalSlots,
			})
			resultCh <- result{questions: questions, err: err, task: t}
		}(task)
	}
	var pool []*session.Question
	var genLog []session.GenerationLogEntry
	var leafIDs []string

	for range tasks {
		r := <-resultCh
		entry := session.GenerationLogEntry{
			NodePath:  r.task.NodePath,
			Archetype: r.task.Archetype,
			Count:     r.task.Count,
		}
		if r.err != nil {
			entry.Error = r.err.Error()
			h.logger.Warn("generator failed for leaf",
				zap.String("node_path", r.task.NodePath),
				zap.String("archetype", r.task.Archetype),
				zap.Error(r.err),
			)
		} else {
			entry.Count = len(r.questions)
			pool = append(pool, r.questions...)
			leafIDs = append(leafIDs, r.task.NodePath)
		}
		genLog = append(genLog, entry)
	}

	// Pre-generate multiple choice options for all questions concurrently
	if len(pool) > 0 {
		var wg sync.WaitGroup
		semOptions := make(chan struct{}, 5) // limit to 5 concurrent LLM calls
		for _, q := range pool {
			wg.Add(1)
			semOptions <- struct{}{}
			go func(question *session.Question) {
				defer func() {
					<-semOptions
					wg.Done()
				}()
				h.populateOptionsIfNeeded(ctx, question, cfg.FrameworkPrompt)
			}(q)
		}
		wg.Wait()
	}

	coverage := session.NewCoverage(leafIDs)

	selector := utils.NewNextQuestionSelector()
	firstQ := selector.Select(pool, coverage, 0)
	if firstQ != nil {
		firstQ.Status = session.StatusAsked
		selector.UpdateCoverage(coverage, firstQ, 0)
	}

	state := &session.SessionState{
		DomainID:        cfg.DomainID,
		FrameworkPrompt: cfg.FrameworkPrompt,
		Pool:            pool,
		History:         []session.HistoryEntry{},
		Coverage:        coverage,
		CurrentQuestion: firstQ,
		GenerationLog:   genLog,
		AskedTotal:      1,
		FollowUpDepth:   0,
	}

	sessionID, err := h.sessionManager.Create(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, InitResponse{
		SessionID:     sessionID,
		Pool:          pool,
		FirstQuestion: firstQ,
		GenerationLog: genLog,
	})
}