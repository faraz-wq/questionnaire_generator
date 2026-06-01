package evaluator

import (
	"context"
	"fmt"

	"github.com/faraz/questionnaire_generator/domain"
	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"go.uber.org/zap"
)

type Evaluator struct {
	client llm.LLMClient
	logger *zap.Logger
}

func NewEvaluator(client llm.LLMClient, logger *zap.Logger) *Evaluator {
	return &Evaluator{client: client, logger: logger}
}

func (e *Evaluator) Evaluate(ctx context.Context, frameworkPrompt string, question *session.Question, answer string, history []session.HistoryEntry) (*llm.EvalResult, error) {
	prompt := e.buildEvaluationPrompt(frameworkPrompt, question.Text, question.ExpectedConcepts, answer, history, question.Persona)

	evalResult, err := e.client.Evaluate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.2,
		MaxTokens:   500,
	})
	if err != nil {
		e.logger.Warn("LLM evaluation failed, returning default",
			zap.Error(err),
			zap.String("question_id", question.ID),
		)
		return &llm.EvalResult{
			Score:           3,
			VagueFlag:       false,
			ConceptsCovered: []string{},
			Missing:         question.ExpectedConcepts,
			Reasoning:       "Evaluation unavailable (LLM error); default score assigned.",
		}, nil
	}

	return evalResult, nil
}

func (e *Evaluator) buildEvaluationPrompt(frameworkPrompt, questionText string, expectedConcepts []string, answer string, history []session.HistoryEntry, persona *domain.Persona) string {
	conceptsStr := ""
	for i, c := range expectedConcepts {
		if i > 0 {
			conceptsStr += ", "
		}
		conceptsStr += c
	}

	historyStr := ""
	if len(history) > 0 {
		historyStr = "Previous Q&A:\n"
		for _, h := range history {
			historyStr += fmt.Sprintf("Q: %s\nA: %s\n", h.Question.Text, h.Answer)
		}
	}

	personaStr := ""
	if persona != nil {
		if persona.Name != "" {
			personaStr = fmt.Sprintf("The question was asked by %s, a %s. %s\n", persona.Name, persona.Role, persona.Backstory)
		} else {
			personaStr = fmt.Sprintf("The question was asked by a %s. %s\n", persona.Role, persona.Backstory)
		}
	}

	// Behavioral dimensions – we assume the domain's behavioral dimensions are known globally? 
	// For simplicity, we ask the LLM to score on a fixed set of dimensions that the domain may have defined.
	// In a more sophisticated implementation, we would pass the domain's BehavioralDimensions list here.
	// For now, we hardcode the five dimensions we plan to use; the evaluator will ignore extra dimensions if not present.
	behavioralInstr := `
Evaluate the candidate's answer on the following behavioral dimensions (1=poor, 5=excellent):
- integrity: upholding ethical and legal standards, even under pressure
- empathy: acknowledging and addressing the customer's emotional state and concerns
- communication: explaining complex concepts in simple, customer-friendly language
- judgment: choosing an effective, situation-appropriate approach
- ownership: taking responsibility for resolving the issue and following through

Return a JSON object:
- score: integer 1-5 (1=poor, 5=excellent) for overall answer quality
- vague_flag: true if the answer is vague or avoids specifics, false otherwise
- concepts_covered: array of expected concepts that were adequately covered
- missing: array of expected concepts that were NOT covered
- reasoning: a brief explanation of the score
- behavioral_scores: a map from dimension ID to integer score (1-5) for each of the dimensions listed above
`

	return fmt.Sprintf(`%s

%s
%s
Question: %s

Expected concepts: %s

Candidate's answer: %s

%s

Focus on accuracy, depth, and relevance to the expected concepts.
Output ONLY valid JSON, no markdown.`,
		frameworkPrompt,
		personaStr,
		historyStr,
		questionText,
		conceptsStr,
		answer,
		behavioralInstr,
	)
}