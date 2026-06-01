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
func WalkTree(node *domain.TaxonomyNode, frameworkPrompt, knowledgeContext string, domainConfig *domain.DomainConfig, archetypeCounts map[string]int) []GeneratorTask {
	return walkTree(node, []string{}, frameworkPrompt, knowledgeContext, domainConfig, archetypeCounts)
}

func walkTree(node *domain.TaxonomyNode, labelPath []string, frameworkPrompt, knowledgeContext string, domainConfig *domain.DomainConfig, archetypeCounts map[string]int) []GeneratorTask {
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
			count := mixEntry.Count
			if archetypeCounts != nil {
				if val, ok := archetypeCounts[mixEntry.Archetype]; ok {
					count = val
				}
			}
			tasks = append(tasks, GeneratorTask{
				Archetype:        mixEntry.Archetype,
				Source:           source,
				NodePath:         node.ID,
				NodeLabelPath:    append([]string{}, currentPath...),
				Count:            count,
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
		tasks = append(tasks, walkTree(child, currentPath, frameworkPrompt, knowledgeContext, domainConfig, archetypeCounts)...)
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