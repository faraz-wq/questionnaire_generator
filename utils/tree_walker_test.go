package utils

import (
	"testing"

	"github.com/faraz/questionnaire_generator/domain"
)

func TestWalkTreeLeafNode(t *testing.T) {
	tree := &domain.TaxonomyNode{
		ID:    "root",
		Label: "Root",
		Weight: 1.0,
		Children: []*domain.TaxonomyNode{
			{
				ID:    "leaf1",
				Label: "Leaf One",
				Weight: 1.0,
				ArchetypeMix: []domain.ArchetypeMixEntry{
					{Archetype: "knowledge", Count: 2},
					{Archetype: "reasoning", Source: "parametric", Count: 1},
				},
				GenerationPrompt: "Test prompt",
			},
		},
	}

	tasks := WalkTree(tree, "fp", "kc", nil, nil)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].Archetype != "knowledge" {
		t.Errorf("expected knowledge, got %s", tasks[0].Archetype)
	}
	if tasks[0].Source != "kb_prompt" {
		t.Errorf("expected kb_prompt, got %s", tasks[0].Source)
	}
	if tasks[1].Source != "parametric" {
		t.Errorf("expected parametric, got %s", tasks[1].Source)
	}
	if tasks[0].FrameworkPrompt != "fp" {
		t.Errorf("expected FrameworkPrompt to be 'fp', got %q", tasks[0].FrameworkPrompt)
	}
	if tasks[0].KnowledgeContext != "kc" {
		t.Errorf("expected KnowledgeContext to be 'kc', got %q", tasks[0].KnowledgeContext)
	}
}

func TestWalkTreeDeepTree(t *testing.T) {
	tree := &domain.TaxonomyNode{
		ID:    "root",
		Label: "Root",
		Weight: 1.0,
		Children: []*domain.TaxonomyNode{
			{
				ID:    "category",
				Label: "Category",
				Weight: 0.5,
				Children: []*domain.TaxonomyNode{
					{
						ID:    "leaf_a",
						Label: "Leaf A",
						Weight: 1.0,
						ArchetypeMix: []domain.ArchetypeMixEntry{
							{Archetype: "knowledge", Count: 1},
						},
						GenerationPrompt: "prompt a",
					},
				},
			},
			{
				ID:    "leaf_b",
				Label: "Leaf B",
				Weight: 0.5,
				ArchetypeMix: []domain.ArchetypeMixEntry{
					{Archetype: "behavioural", Count: 3},
				},
				GenerationPrompt: "prompt b",
			},
		},
	}

	tasks := WalkTree(tree, "fp", "kc", nil, nil)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].NodePath != "leaf_a" {
		t.Errorf("expected leaf_a, got %s", tasks[0].NodePath)
	}
	if tasks[1].NodePath != "leaf_b" {
		t.Errorf("expected leaf_b, got %s", tasks[1].NodePath)
	}
	if tasks[1].Source != "star" {
		t.Errorf("expected star for behavioural, got %s", tasks[1].Source)
	}
}

func TestWalkTreeLabelPath(t *testing.T) {
	tree := &domain.TaxonomyNode{
		ID:    "root",
		Label: "Top",
		Weight: 1.0,
		Children: []*domain.TaxonomyNode{
			{
				ID:    "sub",
				Label: "Middle",
				Weight: 1.0,
				Children: []*domain.TaxonomyNode{
					{
						ID:    "leaf",
						Label: "Bottom",
						Weight: 1.0,
						ArchetypeMix: []domain.ArchetypeMixEntry{
							{Archetype: "knowledge", Count: 1},
						},
					},
				},
			},
		},
	}

	tasks := WalkTree(tree, "fp", "kc", nil, nil)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	expectedPath := []string{"Top", "Middle", "Bottom"}
	if len(tasks[0].NodeLabelPath) != len(expectedPath) {
		t.Fatalf("expected label path length %d, got %d", len(expectedPath), len(tasks[0].NodeLabelPath))
	}
	for i, label := range expectedPath {
		if tasks[0].NodeLabelPath[i] != label {
			t.Errorf("label[%d]: expected %q, got %q", i, label, tasks[0].NodeLabelPath[i])
		}
	}
}

func TestWalkTreeArchetypeOverrides(t *testing.T) {
	tree := &domain.TaxonomyNode{
		ID:    "root",
		Label: "Root",
		Weight: 1.0,
		Children: []*domain.TaxonomyNode{
			{
				ID:    "leaf1",
				Label: "Leaf One",
				Weight: 1.0,
				ArchetypeMix: []domain.ArchetypeMixEntry{
					{Archetype: "knowledge", Count: 2},
					{Archetype: "reasoning", Count: 1},
				},
			},
		},
	}

	overrides := map[string]int{
		"knowledge": 5,
	}

	tasks := WalkTree(tree, "fp", "kc", nil, overrides)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].Archetype == "knowledge" && tasks[0].Count != 5 {
		t.Errorf("expected knowledge count to be overridden to 5, got %d", tasks[0].Count)
	}
	if tasks[1].Archetype == "reasoning" && tasks[1].Count != 1 {
		t.Errorf("expected reasoning count to remain 1, got %d", tasks[1].Count)
	}
}