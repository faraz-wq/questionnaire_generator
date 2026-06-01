package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type OllamaClient struct {
	BaseClient
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewOllamaClient(baseURL, model string, timeoutSec int, logger *zap.Logger) *OllamaClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &OllamaClient{
		BaseClient: BaseClient{Parser: NewParser(logger), Logger: logger},
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:     logger,
	}
}

type ollamaRequest struct {
	Model   string           `json:"model"`
	Prompt  string           `json:"prompt"`
	Stream  bool             `json:"stream"`
	Options ollamaOptions    `json:"options"`
	Format  *json.RawMessage `json:"format,omitempty"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	req := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: ollamaOptions{
			Temperature: opts.Temperature,
			NumPredict:  maxTokens,
		},
	}

	return c.doRequest(ctx, req)
}

func (c *OllamaClient) Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error) {
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	format := json.RawMessage(`"json"`)
	req := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: ollamaOptions{
			Temperature: opts.Temperature,
			NumPredict:  maxTokens,
		},
		Format: &format,
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

func (c *OllamaClient) ParseQuestions(raw string) ([]*QuestionJSON, error) {
	return c.Parser.ParseQuestions(raw)
}

func (c *OllamaClient) ParseEvalResult(raw string) (*EvalResultJSON, error) {
	return c.Parser.ParseEvalResult(raw)
}

func (c *OllamaClient) doRequest(ctx context.Context, req ollamaRequest) (string, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/generate"
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	respBody, err := c.doHTTPWithRetry(ctx, c.httpClient, http.MethodPost, url, bodyBytes, headers)
	if err != nil {
		return "", err
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if ollamaResp.Response == "" {
		return "", fmt.Errorf("empty response from ollama")
	}

	return ollamaResp.Response, nil
}