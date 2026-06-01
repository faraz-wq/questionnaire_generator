package utils

import (
	"testing"
)

func TestEstimateTokensEmpty(t *testing.T) {
	tok := &ApproximateTokenizer{}
	if n := tok.EstimateTokens(""); n != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", n)
	}
}

func TestEstimateTokens(t *testing.T) {
	tok := &ApproximateTokenizer{}
	text := "hello world" // 11 chars
	expected := (11 + 3) / 4 // 3
	if n := tok.EstimateTokens(text); n != expected {
		t.Errorf("expected %d tokens, got %d", expected, n)
	}
}

func TestTruncateNoTruncationNeeded(t *testing.T) {
	tok := &ApproximateTokenizer{}
	text := "short text"
	result := tok.Truncate(text, 100)
	if result != text {
		t.Errorf("expected unchanged text, got %q", result)
	}
}

func TestTruncateAtNewline(t *testing.T) {
	tok := &ApproximateTokenizer{}
	text := "This is a long text.\nWith a newline somewhere.\nMore content here that should get cut off."
	maxTokens := len(text)/4 - 5
	result := tok.Truncate(text, maxTokens)
	if len(result) >= len(text) {
		t.Errorf("expected truncated text to be shorter than original")
	}
}

func TestTruncateAtPeriod(t *testing.T) {
	tok := &ApproximateTokenizer{}
	text := "First sentence with no newlines. Second sentence that continues. Third part that should be cut off."
	maxTokens := 5
	result := tok.Truncate(text, maxTokens)
	if result != text[:maxTokens*4] {
		t.Errorf("expected truncated text to end at a reasonable boundary")
	}
}