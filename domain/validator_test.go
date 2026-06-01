package domain

import (
	"testing"
)

func TestValidateValidDomain(t *testing.T) {
	cfg, err := LoadDomainConfig("../testdata/valid_domain.yaml")
	if err != nil {
		t.Fatalf("expected valid domain to load without error: %v", err)
	}
	if cfg.DomainID != "valid_test" {
		t.Errorf("expected domain_id 'valid_test', got %q", cfg.DomainID)
	}
}

func TestValidateInvalidDomain(t *testing.T) {
	_, err := LoadDomainConfig("../testdata/invalid_domain.yaml")
	if err == nil {
		t.Fatal("expected error for invalid domain, got nil")
	}
}

func TestValidateEmptyDomainID(t *testing.T) {
	cfg := &DomainConfig{
		Taxonomy: &TaxonomyNode{
			ID:    "root",
			Label: "Root",
			Weight: 1.0,
			Children: []*TaxonomyNode{
				{
					ID:    "leaf1",
					Label: "Leaf",
					Weight: 1.0,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 2},
					},
					GenerationPrompt: "test",
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing domain_id")
	}
}

func TestValidateDuplicateIDs(t *testing.T) {
	cfg := &DomainConfig{
		DomainID: "test",
		FrameworkPrompt: "p",
		KnowledgeContext: "k",
		Taxonomy: &TaxonomyNode{
			ID:    "root",
			Label: "Root",
			Weight: 1.0,
			Children: []*TaxonomyNode{
				{
					ID:    "dup",
					Label: "Child1",
					Weight: 0.5,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 1},
					},
					GenerationPrompt: "test",
				},
				{
					ID:    "dup",
					Label: "Child2",
					Weight: 0.5,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 1},
					},
					GenerationPrompt: "test",
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate IDs")
	}
}

func TestValidateInternalNodeHasArchetypeMix(t *testing.T) {
	cfg := &DomainConfig{
		DomainID: "test",
		FrameworkPrompt: "p",
		KnowledgeContext: "k",
		Taxonomy: &TaxonomyNode{
			ID:    "root",
			Label: "Root",
			Weight: 1.0,
			Children: []*TaxonomyNode{
				{
					ID:    "child",
					Label: "Child",
					Weight: 1.0,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 1},
					},
					GenerationPrompt: "test",
					Children: []*TaxonomyNode{
						{
							ID:    "grandchild",
							Label: "Grandchild",
							Weight: 1.0,
							ArchetypeMix: []ArchetypeMixEntry{
								{Archetype: "knowledge", Count: 1},
							},
							GenerationPrompt: "test",
						},
					},
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for internal node with archetype_mix")
	}
}

func TestValidateLeafMissingArchetypeMix(t *testing.T) {
	cfg := &DomainConfig{
		DomainID: "test",
		FrameworkPrompt: "p",
		KnowledgeContext: "k",
		Taxonomy: &TaxonomyNode{
			ID:    "root",
			Label: "Root",
			Weight: 1.0,
			Children: []*TaxonomyNode{
				{
					ID:    "leaf",
					Label: "Leaf",
					Weight: 1.0,
					ArchetypeMix: nil,
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for leaf without archetype_mix")
	}
}

func TestValidateWeightSum(t *testing.T) {
	cfg := &DomainConfig{
		DomainID: "test",
		FrameworkPrompt: "p",
		KnowledgeContext: "k",
		Taxonomy: &TaxonomyNode{
			ID:    "root",
			Label: "Root",
			Weight: 1.0,
			Children: []*TaxonomyNode{
				{
					ID:    "leaf1",
					Label: "Leaf1",
					Weight: 0.3,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 1},
					},
					GenerationPrompt: "test",
				},
				{
					ID:    "leaf2",
					Label: "Leaf2",
					Weight: 0.3,
					ArchetypeMix: []ArchetypeMixEntry{
						{Archetype: "knowledge", Count: 1},
					},
					GenerationPrompt: "test",
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for weights not summing to 1.0")
	}
}

func TestIsLeaf(t *testing.T) {
	leaf := &TaxonomyNode{ID: "leaf"}
	if !leaf.IsLeaf() {
		t.Error("node without children should be leaf")
	}

	internal := &TaxonomyNode{ID: "internal", Children: []*TaxonomyNode{{ID: "child"}}}
	if internal.IsLeaf() {
		t.Error("node with children should not be leaf")
	}
}