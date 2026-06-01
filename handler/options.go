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

type OptionsResponse struct {
	Options      []string `json:"options"`
	CorrectIndex int      `json:"correct_index"`
	Feedbacks    []string `json:"feedbacks"`
}

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

	res, err := h.generateOptions(ctx, q, frameworkPrompt)
	if err != nil {
		h.logger.Warn("failed to generate multiple choice options, using fallback",
			zap.Error(err),
			zap.String("question_id", q.ID),
		)
		q.Options = []string{
			"Understood. Let me look into that and get back to you with RERA documents.",
			"Stamp duty and registration are always included in the basic sales price.",
			"TDS is only applicable when the property value exceeds ₹20 lakhs.",
			"This project is selling out extremely fast, so you should pay the booking deposit today without waiting.",
		}
		q.CorrectIndex = 0
		q.Feedbacks = []string{
			"Correct! Taking ownership and promising RERA-compliant verified documents is the most professional response.",
			"Incorrect. Stamp duty is a government tax and is NOT included in the builder's basic sales price (BSP).",
			"Incorrect. Under compliance, TDS (Tax Deducted at Source) is applicable for property value exceeding ₹50 lakhs, not ₹20 lakhs.",
			"Incorrect. Pressure tactics without giving details can frustrate customers and harm long-term relationships.",
		}
		return
	}
	q.Options = res.Options
	q.CorrectIndex = res.CorrectIndex
	q.Feedbacks = res.Feedbacks
}

// generateOptions constructs the prompt for generating 4 high-fidelity responses (1 correct, 3 distractors) and feedbacks.
func (h *Handler) generateOptions(ctx context.Context, q *session.Question, frameworkPrompt string) (*OptionsResponse, error) {
	prompt := fmt.Sprintf(`%s

You are a multiple-choice question generator for a real-estate agent readiness assessment.
The customer's question/scenario is:
"%s"

Expected concepts to be covered: %s
Ideal answer hint: %s
Archetype: %s
Difficulty: %s

Please generate 4 distinct, highly realistic multiple-choice options along with a feedback array explaining why each choice is correct or incorrect:
1. ONE option must be the **CORRECT and EXCEPTIONAL** response (representing 5/5 score). It must perfectly cover the expected concepts, be factually accurate, and legally/mathematically compliant.
2. THREE options must be **WRONG / INCORRECT** distractors (representing 1/5 or 2/5 score).
   - They should look highly realistic and professional to a layperson, but contain subtle factual errors, compliance risks (e.g. violating RERA, wrong TDS threshold, misrepresenting stamp duty), bad customer handling, or sub-optimal math.
   - Do NOT make the wrong answers outrightly or cartoonishly obvious. They must be plausible distractors that require active thinking to spot.
3. For each of the 4 generated options, provide a clear, concise feedback sentence explaining exactly why it is correct or incorrect in the "feedbacks" array at the same index.

CRITICAL PRECISION REQUIREMENTS:
- **TRIPLE-CHECK CORRECT_INDEX MATCHING**: You must be extremely rigorous! Identify the exact value, formula, or phrase described in the "Ideal answer hint". Ensure that the correct option at `+"`options[correct_index]`"+` represents this correct value, and that the other three indices contain incorrect values. Do NOT set correct_index to point to a wrong distractor index!
- **FOR STRAIGHTFORWARD, FACTUAL, MATH, OR NUMERICAL QUESTIONS** (especially in 'knowledge' and 'reasoning' archetypes, or when the ideal answer hint is a simple number, formula, or short term):
  - Do NOT write long sentences in the options! The options must be extremely short, direct, and contain ONLY the values, numbers, or short direct phrases (e.g., '₹1,60,000', '25%%', 'RERA registration and milestones', 'Commercial', 'Min 7 members', 'Stamp Duty and Registration'). Keep them extremely clean, crisp, and direct!
- **FOR CONVERSATIONAL, SITUATIONAL, OR BEHAVIORAL SCENARIOS**:
  - The options should be written in first-person as a real estate agent's conversational dialogue response to the customer (detailed and realistic).
- Shuffle which index (0, 1, 2, or 3) contains the correct response.
- Output ONLY a valid JSON object matching the following schema. Do NOT wrap the JSON in markdown code blocks like `+"```json"+` or `+"```"+`. Do not include any extra conversational text.

Schema format:
{
  "options": [
    "Option 0 text...",
    "Option 1 text...",
    "Option 2 text...",
    "Option 3 text..."
  ],
  "correct_index": 1,
  "feedbacks": [
    "Feedback explanation for option 0...",
    "Correct! Explanation for option 1...",
    "Feedback explanation for option 2...",
    "Feedback explanation for option 3..."
  ]
}`, frameworkPrompt, q.Text, strings.Join(q.ExpectedConcepts, ", "), q.IdealAnswerHint, q.Archetype, q.Difficulty)

	raw, err := h.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.75,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("llm generate options: %w", err)
	}

	cleaned := stripMarkdownCodeFences(raw)
	var res OptionsResponse
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("unmarshal options JSON: %w (raw response: %s)", err, raw)
	}

	if len(res.Options) != 4 || len(res.Feedbacks) != 4 {
		return nil, fmt.Errorf("expected 4 options and 4 feedbacks, got %d and %d", len(res.Options), len(res.Feedbacks))
	}

	if res.CorrectIndex < 0 || res.CorrectIndex > 3 {
		return nil, fmt.Errorf("invalid correct index: %d", res.CorrectIndex)
	}

	return &res, nil
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
