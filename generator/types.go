package generator

import (
	"context"

	"github.com/faraz/questionnaire_generator/session"
	"github.com/faraz/questionnaire_generator/utils"
)

type GeneratorInput struct {
	Task             utils.GeneratorTask
	Competencies     []string
	SituationalSlots map[string][]string
}

type Generator interface {
	Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error)
}

func DefaultArchetypeMapping(source string) string {
	switch source {
	case "kb_prompt":
		return "KnowledgeGenerator"
	case "parametric":
		return "ReasoningGenerator"
	case "slot_fill":
		return "SituationalGenerator"
	case "star":
		return "BehaviouralGenerator"
	case "free_llm":
		return "CaseGenerator"
	default:
		return "KnowledgeGenerator"
	}
}

func DispatchedArchetype(source string) string {
	return DefaultArchetypeMapping(source)
}

type Dispatcher struct {
	generators map[string]Generator
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		generators: make(map[string]Generator),
	}
}

func (d *Dispatcher) Register(name string, g Generator) {
	d.generators[name] = g
}

func (d *Dispatcher) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	name := DispatchedArchetype(input.Task.Source)
	g, ok := d.generators[name]
	if !ok {
		return nil, &GeneratorNotFoundError{Name: name}
	}
	return g.Generate(ctx, input)
}

type GeneratorNotFoundError struct {
	Name string
}

func (e *GeneratorNotFoundError) Error() string {
	return "generator not found: " + e.Name
}