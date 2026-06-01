package utils

import (
	"github.com/faraz/questionnaire_generator/domain"
)

type GeneratorTask struct {
	Archetype        string
	Source           string
	NodePath         string
	NodeLabelPath    []string
	Count            int
	GenerationPrompt string
	FrameworkPrompt  string
	KnowledgeContext string
	Persona          *domain.Persona
}

// WalkTree walks the taxonomy and returns a slice of GeneratorTask for each leaf,
// using the provided frameworkPrompt and knowledgeContext.
// If domainConfig is not nil, it resolves any PersonaID references.
func WalkTree(node *domain.TaxonomyNode, frameworkPrompt, knowledgeContext string, domainConfig *domain.DomainConfig) []GeneratorTask {
	return walkTree(node, []string{}, frameworkPrompt, knowledgeContext, domainConfig)
}

func walkTree(node *domain.TaxonomyNode, labelPath []string, frameworkPrompt, knowledgeContext string, domainConfig *domain.DomainConfig) []GeneratorTask {
	currentPath := append(labelPath, node.Label)

	if node.IsLeaf() {
		var tasks []GeneratorTask
		var persona *domain.Persona
		if domainConfig != nil && len(domainConfig.Personas) > 0 && node.PersonaID != "" {
			for _, p := range domainConfig.Personas {
				if p.ID == node.PersonaID {
					persona = &p
					break
				}
			}
		}
		for _, mixEntry := range node.ArchetypeMix {
			source := mixEntry.Source
			if source == "" {
				source = defaultSource(mixEntry.Archetype)
			}
			tasks = append(tasks, GeneratorTask{
				Archetype:        mixEntry.Archetype,
				Source:           source,
				NodePath:         node.ID,
				NodeLabelPath:    append([]string{}, currentPath...),
				Count:            mixEntry.Count,
				GenerationPrompt: node.GenerationPrompt,
				FrameworkPrompt:  frameworkPrompt,
				KnowledgeContext: knowledgeContext,
				Persona:          persona,
			})
		}
		return tasks
	}

	var tasks []GeneratorTask
	for _, child := range node.Children {
		tasks = append(tasks, walkTree(child, currentPath, frameworkPrompt, knowledgeContext, domainConfig)...)
	}
	return tasks
}

func defaultSource(archetype string) string {
	switch archetype {
	case "knowledge":
		return "kb_prompt"
	case "reasoning":
		return "parametric"
	case "situational":
		return "slot_fill"
	case "behavioural":
		return "star"
	case "case":
		return "free_llm"
	default:
		return "kb_prompt"
	}
}