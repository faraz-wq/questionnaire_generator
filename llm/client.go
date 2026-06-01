package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type GenerationOptions struct {
	Temperature float64
	MaxTokens   int
}

type EvalResult struct {
	Score           int                  `json:"score"`
	VagueFlag       bool                 `json:"vague_flag"`
	ConceptsCovered []string             `json:"concepts_covered"`
	Missing         []string             `json:"missing"`
	Reasoning       string               `json:"reasoning"`
	BehavioralScores map[string]int       `json:"behavioral_scores,omitempty"`
}

type QuestionJSON struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	Archetype        string   `json:"archetype"`
	Difficulty       string   `json:"difficulty"`
	Type             string   `json:"type"`
	IdealAnswerHint  string   `json:"ideal_answer_hint"`
	ExpectedConcepts []string `json:"expected_concepts"`
	FollowUpFlag     bool     `json:"follow_up_flag"`
	MaxFollowUps     int      `json:"max_follow_ups"`
}

type EvalResultJSON struct {
	Score           int                  `json:"score"`
	VagueFlag       bool                 `json:"vague_flag"`
	ConceptsCovered []string             `json:"concepts_covered"`
	Missing         []string             `json:"missing"`
	Reasoning       string               `json:"reasoning"`
	BehavioralScores map[string]int       `json:"behavioral_scores,omitempty"`
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string, opts GenerationOptions) (string, error)
	Evaluate(ctx context.Context, prompt string, opts GenerationOptions) (*EvalResult, error)
	ParseQuestions(raw string) ([]*QuestionJSON, error)
	ParseEvalResult(raw string) (*EvalResultJSON, error)
}

type BaseClient struct {
	Parser *Parser
	Logger *zap.Logger
}

func (b *BaseClient) doHTTPWithRetry(ctx context.Context, httpClient *http.Client, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	var lastErr error
	maxAttempts := 5
	baseDelay := 4 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1))
			jitter := time.Duration(rand.Int63n(int64(delay)))
			delay += jitter
			if b.Logger != nil {
				b.Logger.Warn("retrying LLM request",
					zap.Int("attempt", attempt+1),
					zap.Int("max_attempts", maxAttempts),
					zap.Duration("delay", delay),
					zap.Error(lastErr),
				)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				continue
			}
			return nil, lastErr
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("retry exhausted after %d attempts: %w", maxAttempts, lastErr)
}