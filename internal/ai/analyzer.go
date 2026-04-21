package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/flametest/market-lens/internal/data"
	"github.com/flametest/market-lens/internal/eventbus"
	"github.com/flametest/market-lens/pkg/model"
)

type Analyzer struct {
	repo        *data.Repository
	bus         *eventbus.InMemoryBus
	logger      *slog.Logger
	apiKey      string
	model       string
	baseURL     string
	maxTokens   int
	temperature float64
}

func NewAnalyzer(repo *data.Repository, bus *eventbus.InMemoryBus, logger *slog.Logger, apiKey, llmModel string) *Analyzer {
	baseURL := "https://api.openai.com/v1"
	if llmModel == "" {
		llmModel = "gpt-4"
	}
	return &Analyzer{
		repo:        repo,
		bus:         bus,
		logger:      logger,
		apiKey:      apiKey,
		model:       llmModel,
		baseURL:     baseURL,
		maxTokens:   500,
		temperature: 0.3,
	}
}

type AnalyzeRequest struct {
	Text string `json:"text"`
}

type llmRequest struct {
	Model       string        `json:"model"`
	Messages    []llmMessage  `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (a *Analyzer) Analyze(ctx context.Context, text string) (*model.SentimentResult, error) {
	if a.apiKey == "" {
		return a.localFallback(text)
	}

	prompt := fmt.Sprintf(`Analyze the following financial text for sentiment. Return a JSON object with:
- "score": float between -1.0 (very negative) and 1.0 (very positive)
- "label": "POSITIVE", "NEGATIVE", or "NEUTRAL"
- "keywords": array of key financial terms or stock symbols mentioned
- "summary": one-sentence summary
- "symbols": array of stock ticker symbols found in the text

Text: %s`, text)

	reqBody := llmRequest{
		Model: a.model,
		Messages: []llmMessage{
			{Role: "system", Content: "You are a financial sentiment analyzer. Always respond with valid JSON only."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llm returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var llmResp llmResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	result, err := a.parseResponse(llmResp.Choices[0].Message.Content, text)
	if err != nil {
		return nil, err
	}

	if a.repo != nil {
		if err := a.repo.SaveSentimentResult(ctx, *result); err != nil && a.logger != nil {
			a.logger.Error("failed to save sentiment result", slog.String("error", err.Error()))
		}
	}

	if a.bus != nil {
		a.bus.Publish(model.EventAnalysisSentiment, *result)
	}

	return result, nil
}

type sentimentJSON struct {
	Score    float64          `json:"score"`
	Label    model.SentimentLabel `json:"label"`
	Keywords []string         `json:"keywords"`
	Summary  string           `json:"summary"`
	Symbols  []string         `json:"symbols"`
}

func (a *Analyzer) parseResponse(content, text string) (*model.SentimentResult, error) {
	// Extract JSON from potentially markdown-wrapped response
	jsonStr := content
	if idx := strings.Index(content, "{"); idx >= 0 {
		jsonStr = content[idx:]
		if lastIdx := strings.LastIndex(jsonStr, "}"); lastIdx > 0 {
			jsonStr = jsonStr[:lastIdx+1]
		}
	}

	var sj sentimentJSON
	if err := json.Unmarshal([]byte(jsonStr), &sj); err != nil {
		return nil, fmt.Errorf("parse sentiment json: %w", err)
	}

	if sj.Label == "" {
		sj.Label = labelFromScore(sj.Score)
	}
	if sj.Label == "" {
		sj.Label = model.SentimentNeutral
	}

	return &model.SentimentResult{
		ID:        data.GenerateID(),
		Text:      text,
		Score:     sj.Score,
		Label:     sj.Label,
		Keywords:  sj.Keywords,
		Summary:   sj.Summary,
		Symbols:   sj.Symbols,
		Timestamp: time.Now().Unix(),
	}, nil
}

func labelFromScore(score float64) model.SentimentLabel {
	if score > 0.2 {
		return model.SentimentPositive
	}
	if score < -0.2 {
		return model.SentimentNegative
	}
	return model.SentimentNeutral
}

// localFallback does basic keyword-based sentiment when no LLM API key is available.
func (a *Analyzer) localFallback(text string) (*model.SentimentResult, error) {
	lower := strings.ToLower(text)

	score := 0.0
	positiveWords := []string{"bullish", "surge", "rally", "growth", "profit", "gain", "upgrade", "beat", "strong", "optimistic", "rise", "higher", "record", "outperform"}
	negativeWords := []string{"bearish", "crash", "decline", "loss", "drop", "fall", "downgrade", "miss", "weak", "pessimistic", "recession", "fear", "plunge", "slump"}

	for _, w := range positiveWords {
		if strings.Contains(lower, w) {
			score += 0.15
		}
	}
	for _, w := range negativeWords {
		if strings.Contains(lower, w) {
			score -= 0.15
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < -1.0 {
		score = -1.0
	}

	// Extract stock symbols (uppercase words, 1-5 chars)
	symbolRe := regexp.MustCompile(`\b[A-Z]{1,5}\b`)
	symbols := symbolRe.FindAllString(text, -1)
	// Filter common non-symbol words
	nonSymbols := map[string]bool{"THE": true, "AND": true, "FOR": true, "NOT": true, "BUT": true, "ARE": true, "WAS": true, "HAS": true, "HAD": true, "HIS": true, "HER": true}
	filtered := make([]string, 0)
	for _, s := range symbols {
		if !nonSymbols[s] {
			filtered = append(filtered, s)
		}
	}

	result := &model.SentimentResult{
		ID:        data.GenerateID(),
		Text:      text,
		Score:     score,
		Label:     labelFromScore(score),
		Keywords:  extractKeywords(lower),
		Summary:   summarize(text),
		Symbols:   filtered,
		Timestamp: time.Now().Unix(),
	}

	if a.repo != nil {
		a.repo.SaveSentimentResult(context.Background(), *result)
	}
	if a.bus != nil {
		a.bus.Publish(model.EventAnalysisSentiment, *result)
	}

	return result, nil
}

func extractKeywords(text string) []string {
	keywords := []string{}
	financialTerms := []string{"stock", "market", "earnings", "revenue", "profit", "loss", "dividend", "valuation", "growth", "gdp", "inflation", "interest rate", "fed", "treasury", "bond", "etf", "ipo", "merger", "acquisition"}
	for _, term := range financialTerms {
		if strings.Contains(text, term) {
			keywords = append(keywords, term)
		}
	}
	return keywords
}

func summarize(text string) string {
	if len(text) > 100 {
		return text[:97] + "..."
	}
	return text
}

func (a *Analyzer) GetHistory(ctx context.Context, symbol string, limit int) ([]model.SentimentResult, error) {
	if a.repo == nil {
		return nil, nil
	}
	return a.repo.GetSentimentResults(ctx, symbol, limit)
}

func (a *Analyzer) GetRiskEvents(ctx context.Context, limit int) ([]model.RiskEvent, error) {
	if a.repo == nil {
		return nil, nil
	}
	return a.repo.GetRiskEvents(ctx, limit)
}
