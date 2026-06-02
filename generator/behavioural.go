package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
)

const behaviouralPromptTemplate = `%s

Context:
%s

Competencies to cover:
%s

Create %d high-quality behavioral interview questions using the STAR (Situation, Task, Action, Result) method. 
Each question must target one of the competencies listed above, and present a realistic real-estate/property consulting professional scenario or prompt requiring the candidate to detail their past experience using STAR.
Do NOT just ask generic questions; tailor them specifically to the real-estate context (e.g. handling difficult buyers, RERA compliance issues, negotiating prices, resolving documentation delays, lead conversion, managing multiple high-priority site visits).

Return a JSON array. Each question object:
- id: a unique UUID string
- text: the behavioral STAR question description (detailed, 2-3 sentences, prompting for a past experience)
- archetype: "behavioural"
- difficulty: one of "easy", "medium", or "hard"
- type: "open"
- ideal_answer_hint: "Use the STAR method: Situation, Task, Action, Result. Provide a specific, detailed example from past experience."
- expected_concepts: array of 3-5 key concepts/actions/lessons learned that an excellent response should demonstrate (e.g. "Situation", "Task", "Action taken", "Result", "Lessons learned")
- follow_up_flag: true
- max_follow_ups: 1

Output ONLY a valid JSON array, no markdown fences.`

type BehaviouralGenerator struct {
	client llm.LLMClient
}

func NewBehaviouralGenerator(client llm.LLMClient) *BehaviouralGenerator {
	return &BehaviouralGenerator{client: client}
}

func (bg *BehaviouralGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	competencies := input.Competencies
	if len(competencies) == 0 {
		competencies = []string{
			"handling customer objections",
			"negotiating property deals",
			"explaining legal and financial documents",
			"building long-term client relationships",
			"managing multiple leads simultaneously",
			"translating technical real-estate terms for customers",
			"resolving customer complaints effectively",
		}
	}

	task := input.Task
	count := task.Count
	if count <= 0 {
		count = 1
	}

	// Try generating via LLM first if client is available
	if bg.client != nil {
		labelPath := ""
		if len(task.NodeLabelPath) > 0 {
			labelPath = task.NodeLabelPath[len(task.NodeLabelPath)-1]
		}

		basePrompt := fmt.Sprintf(behaviouralPromptTemplate,
			task.FrameworkPrompt,
			task.KnowledgeContext,
			strings.Join(competencies, ", "),
			count,
		)

		var prompt string
		if task.Persona != nil {
			if task.Persona.Name != "" {
				prompt = fmt.Sprintf("You are %s, %s. %s\nTone: %s\n\n%s", task.Persona.Name, task.Persona.Role, task.Persona.Backstory, task.Persona.Tone, basePrompt)
			} else {
				prompt = fmt.Sprintf("You are a %s. %s\nTone: %s\n\n%s", task.Persona.Role, task.Persona.Backstory, task.Persona.Tone, basePrompt)
			}
		} else if labelPath != "" {
			prompt = fmt.Sprintf("Topic: %s\n\n%s", labelPath, basePrompt)
		} else {
			prompt = basePrompt
		}

		raw, err := bg.client.Generate(ctx, prompt, llm.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		})
		if err == nil {
			questionsJSON, err := bg.client.ParseQuestions(raw)
			if err == nil && len(questionsJSON) > 0 {
				var questions []*session.Question
				for _, qj := range questionsJSON {
					q := &session.Question{
						ID:               uuid.New().String(),
						Text:             qj.Text,
						Archetype:        "behavioural",
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
		}
	}

	// Fallback to Predefined High-Fidelity Templates if LLM fails or client is nil
	usedLen := 0
	var questions []*session.Question
	for i := 0; i < count; i++ {
		comp := competencies[usedLen%len(competencies)]
		usedLen++

		text := starQuestionForCompetency(comp)

		q := &session.Question{
			ID:               uuid.New().String(),
			Text:             text,
			Archetype:        "behavioural",
			Difficulty:       "medium",
			Type:             "open",
			IdealAnswerHint:  "Use the STAR method: Situation, Task, Action, Result. Provide a specific, detailed example from past experience.",
			ExpectedConcepts: []string{"Situation", "Task", "Action taken", "Result achieved", "Lessons learned"},
			FollowUpFlag:     true,
			MaxFollowUps:     1,
			Status:           session.StatusUnseen,
			NodePath:         task.NodePath,
			NodeLabelPath:    task.NodeLabelPath,
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func starQuestionForCompetency(comp string) string {
	templates := map[string]string{
		// Real-Estate readiness competencies
		"handling customer objections":                          "Tell me about a challenging situation where you had to handle tough customer objections regarding a property's pricing, delivery timeline, or RERA compliance. What was the scenario, how did you address it, and what was the outcome?",
		"negotiating property deals":                            "Describe a complex property deal negotiation where there was a major gap between the buyer's budget and the seller's expectations. How did you structure the negotiation, and what was the final result?",
		"explaining legal and financial documents":             "Give me an example of a time when you had to explain complex legal or financial terms (such as BSP breakups, RERA compliance, TDS rules, or conveyance deeds) to a confused or skeptical buyer. How did you handle it?",
		"building long-term client relationships":               "Tell me about a time you went above and beyond to build a long-term relationship with a customer or investor who was initially indifferent or dissatisfied. What actions did you take, and what was the result?",
		"managing multiple leads simultaneously":                 "Describe a high-pressure scenario where you had to manage multiple active leads, lead calls, or site visits simultaneously under a tight deadline. How did you prioritize, and what was the outcome?",
		"translating technical real-estate terms for customers":  "Give me an example of a time you had to translate dense builder terminology or technical real-estate metrics (like carpet vs super built-up area) into simple language for a first-time homebuyer. How did you ensure clarity?",
		"resolving customer complaints effectively":              "Tell me about a time you had to deal with an extremely frustrated customer who faced construction or documentation delays, or booking discrepancies. How did you defuse the situation and resolve their complaint?",
		
		// Generic/default competencies
		"problem solving":                            "Tell me about a time when you faced a difficult professional problem. What was the situation, what did you do, and what was the outcome?",
		"teamwork":                                   "Describe a situation where you had to work closely with a difficult team member. How did you handle it and what was the result?",
		"adaptability":                               "Give me an example of a time when you had to adapt quickly to a significant change at work. What happened and how did you respond?",
		"designing scalable systems":                 "Tell me about a system you designed or helped design for scalability. What challenges did you face and how did you address them?",
		"debugging complex issues":                   "Describe the most challenging bug you've had to debug. Walk me through your process and what you learned.",
		"collaborating in cross-functional teams":   "Give me an example of a time you collaborated with a non-technical team. How did you ensure effective communication?",
	}

	compLower := strings.ToLower(strings.TrimSpace(comp))
	if tmpl, ok := templates[compLower]; ok {
		return tmpl
	}
	return "Tell me about a time you demonstrated the competency of '" + comp + "'. Describe the situation, the task you faced, the actions you took, and the final results achieved."
}