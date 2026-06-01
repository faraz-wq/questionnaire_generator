package domain

type DomainConfig struct {
	DomainID          string             `yaml:"domain_id"`
	FrameworkPrompt   string             `yaml:"framework_prompt"`
	KnowledgeContext  string             `yaml:"knowledge_context"`
	Taxonomy          *TaxonomyNode      `yaml:"taxonomy"`
	Competencies      []string           `yaml:"competencies"`
	SituationalSlots  map[string][]string `yaml:"situational_slots"`

	// NEW: optional sections
	Personas             []Persona             `yaml:"personas,omitempty"`
	BehavioralDimensions []BehavioralDimension `yaml:"behavioral_dimensions,omitempty"`
}

type Persona struct {
	ID        string `yaml:"id"`               // required, unique identifier
	Name      string `yaml:"name,omitempty"`   // optional; if empty, only Role is shown
	Role      string `yaml:"role"`             // required (e.g., "First-time Homebuyer")
	Backstory string `yaml:"backstory"`       // required: short bio used in LLM prompt
	Tone      string `yaml:"tone"`             // required: adjectives describing voice (e.g., "Curious, slightly anxious")
}

type BehavioralDimension struct {
	ID          string `yaml:"id"`               // required, unique key
	Label       string `yaml:"label"`            // human‑readable name
	Description string `yaml:"description"`     // what to look for in the answer
}

type TaxonomyNode struct {
	ID               string              `yaml:"id"`
	Label            string              `yaml:"label"`
	Weight           float64             `yaml:"weight"`
	Children         []*TaxonomyNode     `yaml:"children"`
	ArchetypeMix     []ArchetypeMixEntry `yaml:"archetype_mix"`
	GenerationPrompt string              `yaml:"generation_prompt"`
	FollowUpFlag     bool                `yaml:"follow_up_flag"`
	MaxFollowUps     int                 `yaml:"max_follow_ups"`
	FollowUpTemplates []FollowUpTemplate `yaml:"follow_up_templates"`

	// NEW: optional persona reference
	PersonaID string `yaml:"persona,omitempty"` // if set, question inherits this persona
}

type ArchetypeMixEntry struct {
	Archetype string `yaml:"archetype"`
	Source    string `yaml:"source"`
	Count     int    `yaml:"count"`
}

type FollowUpTemplate struct {
	ID      string `yaml:"id"`
	Trigger string `yaml:"trigger"`
	Text    string `yaml:"text"`
}

func (n *TaxonomyNode) IsLeaf() bool {
	return len(n.Children) == 0
}