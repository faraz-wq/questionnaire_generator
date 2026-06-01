package utils

import "strings"

type TokenEstimator interface {
	EstimateTokens(text string) int
	Truncate(text string, maxTokens int) string
}

type ApproximateTokenizer struct{}

func (a *ApproximateTokenizer) EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4
}

func (a *ApproximateTokenizer) Truncate(text string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return text
	}

	truncated := text[:maxChars]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxChars/2 {
		return truncated[:lastNewline]
	}
	if lastPeriod := strings.LastIndex(truncated, "."); lastPeriod > maxChars/2 {
		return truncated[:lastPeriod+1]
	}
	return truncated
}