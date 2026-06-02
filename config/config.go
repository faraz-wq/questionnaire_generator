package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                string
	LLMProvider        string
	GeminiModel        string
	GeminiAPIKey       string
	OpenAIModel        string
	OpenAIAPIKey       string
	OllamaBaseURL      string
	OllamaModel        string
	NvidiaAPIKey       string
	NvidiaModel        string
	NvidiaBaseURL      string
	Port               string
	LLMTimeoutSeconds  int
	MaxKnowledgeTokens map[string]int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Env:                getEnv("ENV", "PROD"),
		LLMProvider:        getEnv("LLM_PROVIDER", "ollama"),
		GeminiModel:        getEnv("GEMINI_MODEL", "gemini-1.5-flash"),
		GeminiAPIKey:       os.Getenv("GEMINI_API_KEY"),
		OpenAIModel:        getEnv("OPENAI_MODEL", "gpt-4o"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OllamaBaseURL:      getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:        getEnv("OLLAMA_MODEL", "llama3"),
		NvidiaAPIKey:       os.Getenv("NVIDIA_API_KEY"),
		NvidiaModel:        getEnv("NVIDIA_MODEL", "meta/llama-3.1-70b-instruct"),
		NvidiaBaseURL:      getEnv("NVIDIA_BASE_URL", "https://integrate.api.nvidia.com/v1"),
		Port:               getEnv("PORT", "8080"),
		LLMTimeoutSeconds:  getEnvInt("LLM_TIMEOUT_SECONDS", 120),
		MaxKnowledgeTokens: map[string]int{
			"gemini": getEnvInt("GEMINI_MAX_KNOWLEDGE_TOKENS", 2000),
			"openai": getEnvInt("OPENAI_MAX_KNOWLEDGE_TOKENS", 2000),
			"ollama": getEnvInt("OLLAMA_MAX_KNOWLEDGE_TOKENS", 2000),
			"nvidia": getEnvInt("NVIDIA_MAX_KNOWLEDGE_TOKENS", 2000),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	switch c.LLMProvider {
	case "gemini":
		if c.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY is required when LLM_PROVIDER=gemini")
		}
	case "openai":
		if c.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when LLM_PROVIDER=openai")
		}
	case "ollama":
	case "nvidia":
		if c.NvidiaAPIKey == "" {
			return fmt.Errorf("NVIDIA_API_KEY is required when LLM_PROVIDER=nvidia")
		}
	default:
		return fmt.Errorf("unsupported LLM_PROVIDER: %s (must be gemini, openai, ollama, or nvidia)", c.LLMProvider)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}