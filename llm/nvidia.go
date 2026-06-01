package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type NvidiaClient struct {
	BaseClient
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewNvidiaClient(apiKey, model, baseURL string, timeoutSec int, logger *zap.Logger) *NvidiaClient {
	if baseURL == "" {
		baseURL = "https://integrate.api.nvidia.com/v1"
	}
	if model == "" {
		model = "meta/llama-3.1-70b-instruct"
	}
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &NvidiaClient{
		BaseClient: BaseClient{Parser: NewParser(logger), Logger: logger},
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:     logger,
	}
}

type nvidiaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type nvidiaRequest struct {
	Model       string          `json:"model"`
	Messages    []nvidiaMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type nvidiaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *NvidiaClient) Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	req := nvidiaRequest{
		Model: c.model,
		Messages: []nvidiaMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: opts.Temperature,
		MaxTokens:   maxTokens,
	}

	return c.doRequest(ctx, req)
}

func (c *NvidiaClient) Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	req := nvidiaRequest{
		Model: c.model,
		Messages: []nvidiaMessage{
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

func (c *NvidiaClient) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	return c.Parser.ParseQuestions(raw)
}

func (c *NvidiaClient) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	return c.Parser.ParseEvalResult(raw)
}

func (c *NvidiaClient) doRequest(ctx context.Context, req nvidiaRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
	}

	respBody, err := c.doHTTPWithRetry(ctx, c.httpClient, http.MethodPost, url, bodyBytes, headers)
	if err != nil {
		return "", err
	}

	var nvidiaResp nvidiaResponse
	if err := json.Unmarshal(respBody, &nvidiaResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(nvidiaResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in nvidia response")
	}

	return nvidiaResp.Choices[0].Message.Content, nil
}
