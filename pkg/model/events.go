package model

const (
	EventMarketTick        = "market.tick"
	EventMarketCandle      = "market.candle"
	EventAnalysisSignal    = "analysis.signal"
	EventAnalysisSentiment = "analysis.sentiment"
	EventSignalAggregated  = "analysis.signal.aggregated"
	EventStrategyDecision  = "strategy.order_decision"
	EventRiskApproved      = "risk.approved"
	EventRiskRejected      = "risk.rejected"
	EventRiskAIWarning     = "risk.ai_warning"
	EventExecutionFill     = "execution.fill"
	EventExecutionReject   = "execution.reject"
	EventBacktestProgress  = "backtest.progress"
)
