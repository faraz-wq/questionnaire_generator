package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type OpenAIClient struct {
	BaseClient
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewOpenAIClient(apiKey, model string, timeoutSec int, logger *zap.Logger) *OpenAIClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &OpenAIClient{
		BaseClient: BaseClient{Parser: NewParser(logger), Logger: logger},
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:     logger,
	}
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string         `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64        `json:"temperature"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	req := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: opts.Temperature,
		MaxTokens:   maxTokens,
	}

	return c.doRequest(ctx, req)
}

func (c *OpenAIClient) Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	req := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: opts.Temperature,
		MaxTokens:   maxTokens,
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

func (c *OpenAIClient) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	return c.Parser.ParseQuestions(raw)
}

func (c *OpenAIClient) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	return c.Parser.ParseEvalResult(raw)
}

func (c *OpenAIClient) doRequest(ctx context.Context, req openAIRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := "https://api.openai.com/v1/chat/completions"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
	}

	respBody, err := c.doHTTPWithRetry(ctx, c.httpClient, http.MethodPost, url, bodyBytes, headers)
	if err != nil {
		return "", err
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in openai response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}