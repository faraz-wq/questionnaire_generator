package domain

import (
	"fmt"
	"math"
	"os"

	"gopkg.in/yaml.v3"
)

const weightTolerance = 0.001

func LoadDomainConfig(path string) (*DomainConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read domain config: %w", err)
	}

	var cfg DomainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse domain config: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("domain config validation failed: %w", err)
	}

	return &cfg, nil
}

func Validate(cfg *DomainConfig) error {
	if cfg.DomainID == "" {
		return fmt.Errorf("domain_id is required")
	}
	if cfg.FrameworkPrompt == "" {
		return fmt.Errorf("framework_prompt is required")
	}
	if cfg.KnowledgeContext == "" {
		return fmt.Errorf("knowledge_context is required")
	}
	if cfg.Taxonomy == nil {
		return fmt.Errorf("taxonomy is required")
	}

	ids := make(map[string]bool)
	return validateNode(cfg.Taxonomy, nil, ids)
}

func validateNode(node *TaxonomyNode, parent *TaxonomyNode, ids map[string]bool) error {
	if node.ID == "" {
		return fmt.Errorf("node id is required")
	}
	if node.Label == "" {
		return fmt.Errorf("node label is required for id=%q", node.ID)
	}
	if ids[node.ID] {
		return fmt.Errorf("duplicate node id: %q", node.ID)
	}
	ids[node.ID] = true

	if len(node.Children) == 0 {
		if len(node.ArchetypeMix) == 0 {
			return fmt.Errorf("leaf node %q must have archetype_mix", node.ID)
		}
		for i, entry := range node.ArchetypeMix {
			if entry.Archetype == "" {
				return fmt.Errorf("archetype_mix[%d] of node %q: archetype is required", i, node.ID)
			}
			if entry.Count < 1 {
				return fmt.Errorf("archetype_mix[%d] of node %q: count must be >= 1", i, node.ID)
			}
			source := entry.Source
			if source == "" {
				source = defaultSource(entry.Archetype)
			}
			if (source == "kb_prompt" || source == "free_llm") && node.GenerationPrompt == "" {
				return fmt.Errorf("leaf node %q with source %q requires generation_prompt", node.ID, source)
			}
		}
	} else {
		if len(node.Children) < 1 {
			return fmt.Errorf("internal node %q must have at least one child", node.ID)
		}
		if node.ArchetypeMix != nil {
			return fmt.Errorf("internal node %q must not have archetype_mix", node.ID)
		}

		var siblingSum float64
		for _, child := range node.Children {
			siblingSum += child.Weight
		}
		if math.Abs(siblingSum-1.0) > weightTolerance {
			return fmt.Errorf("sibling weights under node %q sum to %.4f, expected 1.0", node.ID, siblingSum)
		}

		for _, child := range node.Children {
			if err := validateNode(child, node, ids); err != nil {
				return err
			}
		}
	}

	for _, ft := range node.FollowUpTemplates {
		if ft.ID == "" || ft.Trigger == "" || ft.Text == "" {
			return fmt.Errorf("follow_up_template in node %q requires id, trigger, and text", node.ID)
		}
	}

	return nil
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