package generator

import (
	"context"

	"github.com/faraz/questionnaire_generator/session"
	"github.com/google/uuid"
)

type BehaviouralGenerator struct{}

func NewBehaviouralGenerator() *BehaviouralGenerator {
	return &BehaviouralGenerator{}
}

func (bg *BehaviouralGenerator) Generate(ctx context.Context, input GeneratorInput) ([]*session.Question, error) {
	competencies := input.Competencies
	if len(competencies) == 0 {
		competencies = []string{"problem solving", "teamwork", "adaptability"}
	}

	usedLen := 0
	var questions []*session.Question
	for i := 0; i < input.Task.Count; i++ {
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
			NodePath:         input.Task.NodePath,
			NodeLabelPath:    input.Task.NodeLabelPath,
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func starQuestionForCompetency(comp string) string {
	templates := map[string]string{
		"problem solving":                "Tell me about a time when you faced a difficult technical problem. What was the situation, what did you do, and what was the outcome?",
		"teamwork":                       "Describe a situation where you had to work closely with a difficult team member. How did you handle it and what was the result?",
		"adaptability":                   "Give me an example of a time when you had to adapt quickly to a significant change at work. What happened and how did you respond?",
		"designing scalable systems":                  "Tell me about a system you designed or helped design for scalability. What challenges did you face and how did you address them?",
		"debugging complex issues":                    "Describe the most challenging bug you've had to debug. Walk me through your process and what you learned.",
		"collaborating in cross-functional teams":     "Give me an example of a time you collaborated with a non-technical team. How did you ensure effective communication?",
	}

	if tmpl, ok := templates[comp]; ok {
		return tmpl
	}
	return "Tell me about a time you demonstrated " + comp + ". Describe the situation, your actions, and the result."
}