package generator

import (
	"context"
	"fmt"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
)

const casePromptTemplate = `%s

Context:
%s

Topic: %s

Create %d case-study interview questions. Each question should describe a realistic scenario that the candidate must analyze and solve.
To raise the difficulty for advanced screening, inject realistic secondary "Pressure Multipliers" into the scenario (e.g., severe time pressure, conflicting stakeholder demands, or severe information asymmetry/undisclosed contract complications).

Return a JSON array. Each question object:
- id: a unique UUID string
- text: the case study scenario description (detailed, 2-4 sentences)
- archetype: "case"
- difficulty: one of "easy", "medium", or "hard"
- type: "scenario"
- ideal_answer_hint: a brief hint describing an excellent approach
- expected_concepts: array of 3-5 key concepts the answer should cover
- follow_up_flag: true
- max_follow_ups: 2

Output ONLY valid JSON array, no markdown fences.`

type CaseGenerator struct {
	client llm.LLMClient
}

func NewCaseGenerator(client llm.LLMClient) *CaseGenerator {
	return &CaseGenerator{client: client}
}

func (cg *CaseGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	task := input.Task

	labelPath := ""
	if len(task.NodeLabelPath) > 0 {
		labelPath = task.NodeLabelPath[len(task.NodeLabelPath)-1]
	}

	// Build base prompt from template
	basePrompt := fmt.Sprintf(casePromptTemplate,
		task.FrameworkPrompt,
		task.KnowledgeContext,
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

	raw, err := cg.client.Generate(ctx, prompt, llm.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generate: %w", err)
	}

	questionsJSON, err := cg.client.ParseQuestions(raw)
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