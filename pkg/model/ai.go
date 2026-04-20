package model

type SentimentLabel string

const (
	SentimentPositive SentimentLabel = "POSITIVE"
	SentimentNegative SentimentLabel = "NEGATIVE"
	SentimentNeutral  SentimentLabel = "NEUTRAL"
)

type SentimentResult struct {
	ID        string         `json:"id"`
	Text      string         `json:"text"`
	Score     float64        `json:"score"`
	Label     SentimentLabel `json:"label"`
	Keywords  []string       `json:"keywords"`
	Summary   string         `json:"summary"`
	Symbols   []string       `json:"symbols"`
	Timestamp int64          `json:"timestamp"`
}

type RiskEvent struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	SentimentScore float64 `json:"sentimentScore"`
	Action         string  `json:"action"`
	Reason         string  `json:"reason"`
	Timestamp      int64   `json:"timestamp"`
}

type AggregatedSignal struct {
	Signal             Signal   `json:"signal"`
	OriginalConfidence float64  `json:"originalConfidence"`
	AdjustedConfidence float64  `json:"adjustedConfidence"`
	AdjustmentFactor   float64  `json:"adjustmentFactor"`
	SentimentScore     float64  `json:"sentimentScore"`
	AdjustmentReason   string   `json:"adjustmentReason"`
}
