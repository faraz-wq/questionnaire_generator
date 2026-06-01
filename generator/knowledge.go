package generator

import (
	"context"
	"fmt"
	"log"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/faraz/questionnaire_generator/utils"
	"github.com/google/uuid"
)

const knowledgePromptTemplate = `%s

Context:
%s

Topic: %s

Generate %d INTERVIEW QUESTIONS about this topic. Return a JSON array. Each question object:
- id: a unique UUID string
- text: the question text
- archetype: "knowledge"
- difficulty: one of "easy", "medium", or "hard"
- type: one of "open", "scenario", or "factual"
- ideal_answer_hint: a brief hint describing an excellent answer
- expected_concepts: array of 3-5 key concepts the answer should cover
- follow_up_flag: boolean, set true if this question would benefit from follow-ups
- max_follow_ups: integer 1-3 if follow_up_flag is true

Output ONLY valid JSON array, no markdown fences or surrounding text.`

type KnowledgeGenerator struct {
	client         llm.LLMClient
	tokenEstimator utils.TokenEstimator
	maxTokens      int
}

func NewKnowledgeGenerator(client llm.LLMClient, maxTokens int) *KnowledgeGenerator {
	return &KnowledgeGenerator{
		client:         client,
		tokenEstimator: &utils.ApproximateTokenizer{},
		maxTokens:      maxTokens,
	}
}

func (kg *KnowledgeGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	task := input.Task

	context := task.KnowledgeContext
	tokens := kg.tokenEstimator.EstimateTokens(context)
	if tokens > kg.maxTokens {
		context = kg.tokenEstimator.Truncate(context, kg.maxTokens)
		log.Printf("knowledge context truncated: %d → %d tokens", tokens, kg.tokenEstimator.EstimateTokens(context))
	}

	labelPath := ""
	if len(task.NodeLabelPath) > 0 {
		labelPath = task.NodeLabelPath[len(task.NodeLabelPath)-1]
	}

	// Build base prompt from template
	basePrompt := fmt.Sprintf(knowledgePromptTemplate,
		task.FrameworkPrompt,
		context,
		labelPath,
		task.Count,
	)

	// Prepend persona preamble if present
	var prompt string
	if task.Persona != nil {
		if task.Persona.Name != "" {
			prompt = fmt.Sprintf("You are %s, %s. %s\nTone: %s\n\n%s", task.Persona.Name, task.Persona.Role, task.Persona.Backstory, task.Persona.Tone, basePrompt)
		} else {
			prompt = fmt.Sprintf("You are a %s. %s\nTone: %s\n\n%s", task.Persona.Role, task.Persona.Backstory, task.Persona.Tone, basePrompt)
		}
	} else {
		prompt = basePrompt
	}

	raw, err := kg.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generate: %w", err)
	}

	questionsJSON, err := kg.client.ParseQuestions(raw)
	if err != nil {
		return nil, fmt.Errorf("parse questions: %w", err)
	}

	var questions []*session.Question
	for _, qj := range questionsJSON {
		archetype := qj.Archetype
		if input.Task.Archetype != "" {
			archetype = input.Task.Archetype
		}
		q := &session.Question{
			ID:               uuid.New().String(),
			Text:             qj.Text,
			Archetype:        archetype,
			Difficulty:       qj.Difficulty,
			Type:             qj.Type,
			IdealAnswerHint:  qj.IdealAnswerHint,
			ExpectedConcepts: qj.ExpectedConcepts,
			FollowUpFlag:     qj.FollowUpFlag,
			MaxFollowUps:     qj.MaxFollowUps,
			Status:           session.StatusUnseen,
			NodePath:         task.NodePath,
			NodeLabelPath:    task.NodeLabelPath,
		}
		questions = append(questions, q)
	}

	return questions, nil
}