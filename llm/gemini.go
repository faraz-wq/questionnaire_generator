package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type GeminiClient struct {
	BaseClient
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewGeminiClient(apiKey, model string, timeoutSec int, logger *zap.Logger) *GeminiClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &GeminiClient{
		BaseClient: BaseClient{Parser: NewParser(logger), Logger: logger},
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:     logger,
	}
}

type geminiRequest struct {
	Contents         []geminiContent      `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (c *GeminiClient) Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     opts.Temperature,
			MaxOutputTokens: maxTokens,
		},
	}

	return c.doRequest(ctx, req)
}

func (c *GeminiClient) Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	req := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     opts.Temperature,
			MaxOutputTokens: maxTokens,
		},
	}

	raw, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	evalJSON, err := c.ParseEvalResult(raw)
	if err != nil {
		return nil, fmt.Errorf("parse eval result: %w", err)
	}

	return &EvalResult{
		Score:           evalJSON.Score,
		VagueFlag:       evalJSON.VagueFlag,
		ConceptsCovered: evalJSON.ConceptsCovered,
		Missing:         evalJSON.Missing,
		Reasoning:       evalJSON.Reasoning,
	}, nil
}

func (c *GeminiClient) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	return c.Parser.ParseQuestions(raw)
}

func (c *GeminiClient) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	return c.Parser.ParseEvalResult(raw)
}

func (c *GeminiClient) doRequest(ctx context.Context, req geminiRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	respBody, err := c.doHTTPWithRetry(ctx, c.httpClient, http.MethodPost, url, bodyBytes, headers)
	if err != nil {
		return "", err
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no candidates in gemini response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}