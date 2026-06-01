package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type situationalTemplateData struct {
	Slots map[string][]string `yaml:"slots"`
}

type SituationalGenerator struct {
	client        llm.LLMClient
	templatesDir  string
	domainSlots   map[string][]string
	loaded        map[string]situationalTemplateData
}

func NewSituationalGenerator(client llm.LLMClient, templatesDir string, domainSlots map[string][]string) *SituationalGenerator {
	return &SituationalGenerator{
		client:       client,
		templatesDir: templatesDir,
		domainSlots:  domainSlots,
		loaded:       make(map[string]situationalTemplateData),
	}
}

func (sg *SituationalGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	nodePath := input.Task.NodePath

	slotData, err := sg.getSlotData(nodePath)
	if err != nil {
		return nil, fmt.Errorf("get slot data: %w", err)
	}

	allSlots := make(map[string][]string)
	for k, v := range input.SituationalSlots {
		allSlots[k] = v
	}
	for k, v := range slotData.Slots {
		allSlots[k] = v
	}

	var questions []*session.Question
	for i := 0; i < input.Task.Count; i++ {
		selected := make(map[string]string)
		for slotName, options := range allSlots {
			selected[slotName] = options[rand.Intn(len(options))]
		}

		slotDesc := ""
		for name, val := range selected {
			slotDesc += fmt.Sprintf("  - %s: %s\n", name, val)
		}

		// Build base prompt from template
		basePrompt := fmt.Sprintf(`%s

Context:
%s

Topic: %s

Create a situational interview question using these parameters:
%s

The question should describe a realistic scenario using the role, constraint, stakeholder, and pressure.
Return a JSON array containing one object with these fields:
- id: a UUID string
- text: the question text (a scenario the candidate must respond to)
- archetype: "situational"
- difficulty: "medium"
- type: "scenario"
- ideal_answer_hint: brief hint on what a good answer looks like
- expected_concepts: array of 3-5 concepts expected in the answer
- follow_up_flag: false
- max_follow_ups: 0

Output ONLY valid JSON array, no markdown.`,
			input.Task.FrameworkPrompt,
			input.Task.KnowledgeContext,
			input.Task.NodeLabelPath[len(input.Task.NodeLabelPath)-1],
			slotDesc,
		)

		// Prepend persona preamble if present
		var prompt string
		if input.Task.Persona != nil {
			if input.Task.Persona.Name != "" {
				prompt = fmt.Sprintf("You are %s, %s. %s\nTone: %s\n\n%s", input.Task.Persona.Name, input.Task.Persona.Role, input.Task.Persona.Backstory, input.Task.Persona.Tone, basePrompt)
			} else {
				prompt = fmt.Sprintf("You are a %s. %s\nTone: %s\n\n%s", input.Task.Persona.Role, input.Task.Persona.Backstory, input.Task.Persona.Tone, basePrompt)
			}
		} else {
			prompt = basePrompt
		}

		raw, err := sg.client.Generate(ctx, prompt, llm.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   500,
		})
		if err != nil {
			return nil, fmt.Errorf("LLM generate situational: %w", err)
		}

		questionsJSON, err := sg.client.ParseQuestions(raw)
		if err != nil || len(questionsJSON) == 0 {
			return nil, fmt.Errorf("parse situational questions: %w", err)
		}

		qj := questionsJSON[0]
		q := &session.Question{
			ID:               uuid.New().String(),
			Text:             qj.Text,
			Archetype:        "situational",
			Difficulty:       qj.Difficulty,
			Type:             "scenario",
			IdealAnswerHint:  qj.IdealAnswerHint,
			ExpectedConcepts: qj.ExpectedConcepts,
			FollowUpFlag:     qj.FollowUpFlag,
			MaxFollowUps:     qj.MaxFollowUps,
			Status:           session.StatusUnseen,
			NodePath:         input.Task.NodePath,
			NodeLabelPath:    input.Task.NodeLabelPath,
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (sg *SituationalGenerator) getSlotData(nodePath string) (situationalTemplateData, error) {
	if data, ok := sg.loaded[nodePath]; ok {
		return data, nil
	}

	safeName := filepath.Base(nodePath)
	path := filepath.Join(sg.templatesDir, safeName+".yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			sg.loaded[nodePath] = situationalTemplateData{}
			return situationalTemplateData{}, nil
		}
		return situationalTemplateData{}, fmt.Errorf("read situational template: %w", err)
	}

	var t situationalTemplateData
	if err := yaml.Unmarshal(data, &t); err != nil {
		return situationalTemplateData{}, fmt.Errorf("parse situational template: %w", err)
	}

	sg.loaded[nodePath] = t
	return t, nil
}