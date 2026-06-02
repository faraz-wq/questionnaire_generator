package main

import (
	"log"

	"github.com/faraz/questionnaire_generator/config"
	"github.com/faraz/questionnaire_generator/evaluator"
	"github.com/faraz/questionnaire_generator/followup"
	"github.com/faraz/questionnaire_generator/generator"
	"github.com/faraz/questionnaire_generator/handler"
	"github.com/faraz/questionnaire_generator/llm"
	"github.com/faraz/questionnaire_generator/session"
	"github.com/faraz/questionnaire_generator/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	client := createLLMClient(cfg, logger)

	sessionManager := session.NewSessionManager()

	dispatcher := generator.NewDispatcher()
	dispatcher.Register("KnowledgeGenerator", generator.NewKnowledgeGenerator(client, cfg.MaxKnowledgeTokens[cfg.LLMProvider]))
	dispatcher.Register("ReasoningGenerator", generator.NewReasoningGenerator("templates/reasoning"))
	dispatcher.Register("SituationalGenerator", generator.NewSituationalGenerator(client, "templates/situational", nil))
	dispatcher.Register("BehaviouralGenerator", generator.NewBehaviouralGenerator(client))
	dispatcher.Register("CaseGenerator", generator.NewCaseGenerator(client))

	eval := evaluator.NewEvaluator(client, logger)
	fupRouter := followup.NewFollowUpRouter(client, logger)
	selector := utils.NewNextQuestionSelector()

	h := handler.NewHandler(sessionManager, dispatcher, eval, fupRouter, selector, client, logger)

	r := gin.Default()

	r.POST("/sessions/init", h.InitSession)
	r.POST("/sessions/:id/turn", h.ProcessTurn)
	r.GET("/sessions/:id/summary", h.GetSummary)

	r.Static("/ui", "./ui")
	r.GET("/", func(c *gin.Context) {
		c.File("./ui/index.html")
	})

	logger.Info("server starting", zap.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func createLLMClient(cfg *config.Config, logger *zap.Logger) llm.LLMClient {
	switch cfg.LLMProvider {
	case "gemini":
		return llm.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.LLMTimeoutSeconds, logger)
	case "openai":
		return llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.LLMTimeoutSeconds, logger)
	case "ollama":
		return llm.NewOllamaClient(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.LLMTimeoutSeconds, logger)
	case "nvidia":
		return llm.NewNvidiaClient(cfg.NvidiaAPIKey, cfg.NvidiaModel, cfg.NvidiaBaseURL, cfg.LLMTimeoutSeconds, logger)
	default:
		return llm.NewOllamaClient(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.LLMTimeoutSeconds, logger)
	}
}