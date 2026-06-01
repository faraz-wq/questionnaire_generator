package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type reasoningTemplate struct {
	Template   string         `yaml:"template"`
	Slots      map[string]interface{} `yaml:"slots"`
	Answer     interface{}    `yaml:"answer"`
	Rule       string         `yaml:"rule"`
	Difficulty string         `yaml:"difficulty"`
}

type ReasoningGenerator struct {
	templatesDir string
	loaded       map[string][]reasoningTemplate
}

func NewReasoningGenerator(templatesDir string) *ReasoningGenerator {
	return &ReasoningGenerator{
		templatesDir: templatesDir,
		loaded:       make(map[string][]reasoningTemplate),
	}
}

func (rg *ReasoningGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	nodePath := input.Task.NodePath
	templates, ok := rg.loaded[nodePath]
	if !ok {
		loaded, err := rg.loadTemplates(nodePath)
		if err != nil {
			return nil, fmt.Errorf("load reasoning templates for %s: %w", nodePath, err)
		}
		templates = loaded
		rg.loaded[nodePath] = templates
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no reasoning templates found for %s", nodePath)
	}

	var questions []*session.Question
	for i := 0; i < input.Task.Count; i++ {
		tmpl := templates[rand.Intn(len(templates))]

		text := tmpl.Template
		for key, val := range tmpl.Slots {
			placeholder := "{{" + key + "}}"
			text = strings.ReplaceAll(text, placeholder, fmt.Sprint(val))
		}

		concepts := []string{}
		if tmpl.Rule != "" {
			concepts = append(concepts, tmpl.Rule)
		}
		if len(concepts) < 3 {
			concepts = append(concepts, "logical reasoning", "pattern recognition", "deductive thinking")
			concepts = concepts[:3]
		}

		difficulty := tmpl.Difficulty
		if difficulty == "" {
			difficulty = "medium"
		}

		answerHint := fmt.Sprintf("The correct answer is %v", tmpl.Answer)

		q := &session.Question{
			ID:               uuid.New().String(),
			Text:             text,
			Archetype:        "reasoning",
			Difficulty:       difficulty,
			Type:             "open",
			IdealAnswerHint:  answerHint,
			ExpectedConcepts: concepts,
			FollowUpFlag:     false,
			MaxFollowUps:     0,
			Status:           session.StatusUnseen,
			NodePath:         input.Task.NodePath,
			NodeLabelPath:    input.Task.NodeLabelPath,
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (rg *ReasoningGenerator) loadTemplates(nodePath string) ([]reasoningTemplate, error) {
	safeName := filepath.Base(nodePath)
	path := filepath.Join(rg.templatesDir, safeName+".yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}

	var templates []reasoningTemplate
	if err := yaml.Unmarshal(data, &templates); err != nil {
		return nil, fmt.Errorf("parse template YAML: %w", err)
	}

	return templates, nil
}