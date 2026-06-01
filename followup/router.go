package followup

import (
	"context"
	"fmt"
	"strings"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FollowUpRouter struct {
	client llm.LLMClient
	logger *zap.Logger
}

func NewFollowUpRouter(client llm.LLMClient, logger *zap.Logger) *FollowUpRouter {
	return &FollowUpRouter{client: client, logger: logger}
}

type FollowUpDecision struct {
	Fire     bool
	Question *session.Question
}

func (fr *FollowUpRouter) Route(ctx context.Context, question *session.Question, evalResult *llm.EvalResult, followUpDepth int) (*FollowUpDecision, error) {
	if followUpDepth >= 3 {
		return &FollowUpDecision{Fire: false}, nil
	}

	shouldFire := false
	trigger := ""

	if evalResult.Score < 3 {
		shouldFire = true
		trigger = "score_low"
	} else if evalResult.VagueFlag {
		shouldFire = true
		trigger = "vague"
	}

	if !shouldFire && question.FollowUpFlag && followUpDepth == 0 {
		shouldFire = true
		trigger = "follow_up_flag"
	}

	if !shouldFire && question.MaxFollowUps > 0 && question.FollowUpsUsed < question.MaxFollowUps {
		shouldFire = true
		trigger = "caps_remain"
	}

	if !shouldFire {
		return &FollowUpDecision{Fire: false}, nil
	}

	if question.FollowUpTemplate != nil && question.FollowUpTemplate.Trigger == trigger {
		return fr.authoredFollowUp(question)
	}

	if question.FollowUpTemplate != nil && trigger == "score_low" && question.FollowUpTemplate.Trigger == "vague" {
		return fr.authoredFollowUp(question)
	}

	if question.FollowUpTemplate != nil && trigger == "vague" && question.FollowUpTemplate.Trigger == "score_low" {
		return fr.authoredFollowUp(question)
	}

	return fr.llmFollowUp(ctx, question, evalResult)
}

func (fr *FollowUpRouter) authoredFollowUp(question *session.Question) (*FollowUpDecision, error) {
	text := question.FollowUpTemplate.Text
	text = strings.ReplaceAll(text, "{{answer}}", "your previous answer")

	followUp := &session.Question{
		ID:               uuid.New().String(),
		Text:             text,
		Archetype:        "follow_up",
		Difficulty:       question.Difficulty,
		Type:             "open",
		IdealAnswerHint:  "Provide additional detail and clarity.",
		ExpectedConcepts: question.ExpectedConcepts,
		FollowUpFlag:     false,
		MaxFollowUps:     0,
		Status:           session.StatusUnseen,
		NodePath:         question.NodePath,
		NodeLabelPath:    question.NodeLabelPath,
		ParentQuestionID: question.ID,
	}

	return &FollowUpDecision{Fire: true, Question: followUp}, nil
}

func (fr *FollowUpRouter) llmFollowUp(ctx context.Context, question *session.Question, evalResult *llm.EvalResult) (*FollowUpDecision, error) {
	prompt := fmt.Sprintf(`The candidate gave a %d/5 answer to: %q

Their answer was evaluated with:
- Score: %d/5
- Concepts covered: %v
- Missing concepts: %v

Generate a follow-up question that probes the missing concepts or asks the candidate to elaborate.
Return ONLY the follow-up question as plain text, no JSON, no markdown.`,
		evalResult.Score, question.Text,
		evalResult.Score,
		evalResult.ConceptsCovered,
		evalResult.Missing,
	)

	raw, err := fr.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   200,
	})
	if err != nil {
		fr.logger.Warn("LLM follow-up generation failed", zap.Error(err))
		return &FollowUpDecision{Fire: false}, nil
	}

	followUpText := strings.TrimSpace(raw)

	followUp := &session.Question{
		ID:               uuid.New().String(),
		Text:             followUpText,
		Archetype:        "follow_up",
		Difficulty:       question.Difficulty,
		Type:             "open",
		IdealAnswerHint:  "Provide additional detail and clarity.",
		ExpectedConcepts: question.ExpectedConcepts,
		FollowUpFlag:     false,
		MaxFollowUps:     0,
		Status:           session.StatusUnseen,
		NodePath:         question.NodePath,
		NodeLabelPath:    question.NodeLabelPath,
		ParentQuestionID: question.ID,
	}

	return &FollowUpDecision{Fire: true, Question: followUp}, nil
}