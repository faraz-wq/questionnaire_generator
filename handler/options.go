package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"go.uber.org/zap"
)

// populateOptionsIfNeeded ensures a question has multiple-choice options.
// If not already present, it generates them via the LLM.
func (h *Handler) populateOptionsIfNeeded(ctx context.Context, q *session.Question, frameworkPrompt string) {
	if q == nil {
		return
	}
	if len(q.Options) > 0 {
		return
	}

	h.logger.Info("generating multiple choice options for question",
		zap.String("question_id", q.ID),
		zap.String("archetype", q.Archetype),
	)

	options, err := h.generateOptions(ctx, q, frameworkPrompt)
	if err != nil {
		h.logger.Warn("failed to generate multiple choice options, using fallback",
			zap.Error(err),
			zap.String("question_id", q.ID),
		)
		q.Options = []string{
			"Understood. Let me look into that and get back to you.",
			"We have some great options matching your requirements. What is your ideal budget?",
			"I will send over our digital brochure and pricing lists immediately.",
			"Let's schedule a quick call or a site visit to discuss this in detail.",
		}
		return
	}
	q.Options = options
}

// generateOptions constructs the prompt for generating 4 high-fidelity responses and parses the result.
func (h *Handler) generateOptions(ctx context.Context, q *session.Question, frameworkPrompt string) ([]string, error) {
	prompt := fmt.Sprintf(`%s

You are a generator that outputs 4 multiple-choice options (A, B, C, D) for a real-estate agent readiness assessment.
The customer's question/scenario is:
"%s"

Expected concepts to be covered in candidate's response: %s
Ideal answer hint: %s
Archetype: %s
Difficulty: %s

Please generate 4 distinct, highly realistic multiple-choice options for a candidate agent to choose as their response:
- One option (A, B, C, or D) should be an exceptional, highly professional response (representing a score of 5/5) that perfectly covers the expected concepts and handles the customer with high empathy, domain expertise, and clear communication.
- One option should be a proficient, standard response (representing a score of 3/5) that is decent but misses some depth, secondary concepts, or compliance details.
- One option should be a weak response (representing a score of 2/5) that is a bit short, slightly unhelpful, or does not address the core customer concerns.
- One option should be a poor or vague response (representing a score of 1/5) that is generic, avoids details, or is unprofessional.

Requirements:
1. Make all 4 responses look highly realistic and professional to a non-expert, so the user has to think and apply judgment.
2. The responses should be written in first-person as a real estate agent.
3. Shuffle which position contains the exceptional response so it is not always the same.
4. Output ONLY a valid JSON array of exactly 4 strings. Do NOT wrap the JSON in markdown code blocks like `+"```json"+` or `+"```"+`. Do not include any explanation, introductory text, or formatting other than the JSON array itself.

Example output:
[
  "Yes, stamp duty is a state government tax...",
  "Sure, we have plenty of commercial office spaces...",
  "I will send you the document brochure immediately...",
  "Stamp duty is not included in the basic sales price..."
]`, frameworkPrompt, q.Text, strings.Join(q.ExpectedConcepts, ", "), q.IdealAnswerHint, q.Archetype, q.Difficulty)

	raw, err := h.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   800,
	})
	if err != nil {
		return nil, fmt.Errorf("llm generate options: %w", err)
	}

	cleaned := stripMarkdownCodeFences(raw)
	var options []string
	if err := json.Unmarshal([]byte(cleaned), &options); err != nil {
		return nil, fmt.Errorf("unmarshal options JSON: %w (raw response: %s)", err, raw)
	}

	if len(options) != 4 {
		return nil, fmt.Errorf("expected 4 options, got %d", len(options))
	}

	return options, nil
}

func stripMarkdownCodeFences(input string) string {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "```") {
		// find first newline
		if idx := strings.Index(input, "\n"); idx != -1 {
			input = input[idx+1:]
		}
		if strings.HasSuffix(input, "```") {
			input = input[:len(input)-3]
		}
	}
	return strings.TrimSpace(input)
}
