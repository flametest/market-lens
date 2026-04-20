import type {
  Tick, Candle, Signal, StrategyConfig, BacktestRun,
  Position, Order, AccountInfo, SentimentResult, RiskEvent,
} from '../types'

const now = Date.now()
const hour = 3600000
const day = 86400000

export const mockWatchlist: Tick[] = [
  { symbol: 'AAPL', price: 198.42, volume: 52436789, change: 3.24, changePercent: 1.66, high: 199.62, low: 195.18, open: 195.50, timestamp: now - 1000 },
  { symbol: 'GOOGL', price: 168.35, volume: 23456789, change: -1.87, changePercent: -1.10, high: 170.92, low: 167.40, open: 170.22, timestamp: now - 2000 },
  { symbol: 'MSFT', price: 425.18, volume: 18923456, change: 5.67, changePercent: 1.35, high: 426.50, low: 418.90, open: 419.51, timestamp: now - 1500 },
  { symbol: 'AMZN', price: 186.92, volume: 32145678, change: 2.15, changePercent: 1.16, high: 188.30, low: 184.50, open: 184.77, timestamp: now - 3000 },
  { symbol: 'NVDA', price: 875.60, volume: 42356789, change: 12.30, changePercent: 1.42, high: 880.00, low: 860.20, open: 863.30, timestamp: now - 2500 },
  { symbol: 'META', price: 502.30, volume: 15678901, change: -4.20, changePercent: -0.83, high: 508.50, low: 500.10, open: 506.50, timestamp: now - 500 },
  { symbol: 'TSLA', price: 245.80, volume: 67890123, change: 8.90, changePercent: 3.76, high: 248.00, low: 235.50, open: 236.90, timestamp: now - 800 },
  { symbol: 'JPM', price: 198.60, volume: 8765432, change: 0.85, changePercent: 0.43, high: 199.40, low: 197.20, open: 197.75, timestamp: now - 1200 },
]

function generateCandles(count: number, basePrice: number): Candle[] {
  const candles: Candle[] = []
  let price = basePrice
  for (let i = count; i >= 0; i--) {
    const open = price
    const change = (Math.random() - 0.48) * 5
    const close = open + change
    const high = Math.max(open, close) + Math.random() * 3
    const low = Math.min(open, close) - Math.random() * 3
    const volume = Math.floor(Math.random() * 50000000) + 10000000
    candles.push({
      time: Math.floor((now - i * day) / 1000),
      open: Math.round(open * 100) / 100,
      high: Math.round(high * 100) / 100,
      low: Math.round(low * 100) / 100,
      close: Math.round(close * 100) / 100,
      volume,
    })
    price = close
  }
  return candles
}

export const mockCandles = generateCandles(120, 180)

export const mockSignals: Signal[] = [
  { id: 'sig-1', symbol: 'AAPL', source: 'MA Crossover', type: 'BUY', strength: 0.82, price: 198.42, timestamp: now - 60000 },
  { id: 'sig-2', symbol: 'NVDA', source: 'RSI', type: 'BUY', strength: 0.75, price: 875.60, timestamp: now - 120000 },
  { id: 'sig-3', symbol: 'GOOGL', source: 'MACD', type: 'SELL', strength: 0.68, price: 168.35, timestamp: now - 180000 },
  { id: 'sig-4', symbol: 'TSLA', source: 'AI Sentiment', type: 'BUY', strength: 0.71, price: 245.80, timestamp: now - 240000 },
  { id: 'sig-5', symbol: 'META', source: 'Bollinger', type: 'SELL', strength: 0.60, price: 502.30, timestamp: now - 300000 },
  { id: 'sig-6', symbol: 'MSFT', source: 'MA Crossover', type: 'BUY', strength: 0.90, price: 425.18, timestamp: now - 360000 },
  { id: 'sig-7', symbol: 'AMZN', source: 'AI Sentiment', type: 'BUY', strength: 0.55, price: 186.92, timestamp: now - 420000 },
]

export const mockStrategies: StrategyConfig[] = [
  { id: 'strat-1', name: 'ma_crossover', displayName: 'MA Crossover', enabled: true, status: 'RUNNING', symbols: ['AAPL', 'MSFT'], params: { shortPeriod: 5, longPeriod: 20 }, aiEnabled: true, aiWeight: 0.3, pnl: 12450.80, winRate: 0.62, createdAt: '2026-03-15' },
  { id: 'strat-2', name: 'rsi_reversion', displayName: 'RSI Mean Reversion', enabled: true, status: 'RUNNING', symbols: ['NVDA', 'TSLA'], params: { period: 14, oversold: 30, overbought: 70 }, aiEnabled: false, aiWeight: 0, pnl: -2340.50, winRate: 0.45, createdAt: '2026-03-20' },
  { id: 'strat-3', name: 'ma_crossover_2', displayName: 'MA Crossover V2', enabled: false, status: 'STOPPED', symbols: ['GOOGL'], params: { shortPeriod: 10, longPeriod: 50 }, aiEnabled: true, aiWeight: 0.4, pnl: 5670.20, winRate: 0.58, createdAt: '2026-04-01' },
]

export const mockBacktests: BacktestRun[] = [
  { id: 'bt-1', strategyName: 'MA Crossover', symbol: 'AAPL', startDate: '2025-01-01', endDate: '2026-01-01', interval: '1d', aiMode: 'compare', status: 'COMPLETED', totalReturn: 24.5, annualReturn: 24.5, maxDrawdown: -8.3, sharpeRatio: 1.82, winRate: 0.62, totalTrades: 48, profitTrades: 30, lossTrades: 18, createdAt: '2026-04-18' },
  { id: 'bt-2', strategyName: 'RSI Mean Reversion', symbol: 'NVDA', startDate: '2025-06-01', endDate: '2026-03-01', interval: '1d', aiMode: 'on', status: 'COMPLETED', totalReturn: 18.7, annualReturn: 23.4, maxDrawdown: -12.1, sharpeRatio: 1.45, winRate: 0.55, totalTrades: 36, profitTrades: 20, lossTrades: 16, createdAt: '2026-04-15' },
  { id: 'bt-3', strategyName: 'MA Crossover V2', symbol: 'GOOGL', startDate: '2024-01-01', endDate: '2026-01-01', interval: '1d', aiMode: 'off', status: 'COMPLETED', totalReturn: 15.2, annualReturn: 7.6, maxDrawdown: -15.6, sharpeRatio: 0.98, winRate: 0.52, totalTrades: 62, profitTrades: 32, lossTrades: 30, createdAt: '2026-04-10' },
]

export const mockAccount: AccountInfo = {
  cash: 75430.50,
  totalValue: 125680.30,
  unrealizedPnl: 3245.80,
  realizedPnl: 8750.00,
  dailyPnl: 1230.40,
  currency: 'USD',
}

export const mockPositions: Position[] = [
  { symbol: 'AAPL', name: 'Apple Inc.', side: 'BUY', quantity: 50, avgPrice: 185.30, currentPrice: 198.42, unrealizedPnl: 655.99, unrealizedPnlPercent: 7.08 },
  { symbol: 'NVDA', name: 'NVIDIA Corp.', side: 'BUY', quantity: 20, avgPrice: 820.50, currentPrice: 875.60, unrealizedPnl: 1102.00, unrealizedPnlPercent: 6.72 },
  { symbol: 'MSFT', name: 'Microsoft Corp.', side: 'BUY', quantity: 30, avgPrice: 410.20, currentPrice: 425.18, unrealizedPnl: 449.40, unrealizedPnlPercent: 3.65 },
  { symbol: 'GOOGL', name: 'Alphabet Inc.', side: 'BUY', quantity: 80, avgPrice: 172.50, currentPrice: 168.35, unrealizedPnl: -332.00, unrealizedPnlPercent: -2.41 },
  { symbol: 'TSLA', name: 'Tesla Inc.', side: 'BUY', quantity: 25, avgPrice: 230.40, currentPrice: 245.80, unrealizedPnl: 385.00, unrealizedPnlPercent: 6.68 },
]

export const mockOrders: Order[] = [
  { id: 'ord-1', symbol: 'AAPL', side: 'BUY', type: 'MARKET', status: 'FILLED', quantity: 10, filledQty: 10, price: 198.42, avgFillPrice: 198.45, fee: 0.99, strategy: 'MA Crossover', createdAt: new Date(now - hour).toISOString() },
  { id: 'ord-2', symbol: 'GOOGL', side: 'SELL', type: 'MARKET', status: 'FILLED', quantity: 20, filledQty: 20, price: 168.35, avgFillPrice: 168.30, fee: 0.67, strategy: 'MA Crossover V2', createdAt: new Date(now - hour * 3).toISOString() },
  { id: 'ord-3', symbol: 'NVDA', side: 'BUY', type: 'LIMIT', status: 'NEW', quantity: 5, filledQty: 0, price: 860.00, avgFillPrice: 0, fee: 0, strategy: 'RSI Mean Reversion', createdAt: new Date(now - hour * 6).toISOString() },
  { id: 'ord-4', symbol: 'TSLA', side: 'BUY', type: 'MARKET', status: 'FILLED', quantity: 15, filledQty: 15, price: 236.90, avgFillPrice: 236.88, fee: 0.71, strategy: 'RSI Mean Reversion', createdAt: new Date(now - day).toISOString() },
  { id: 'ord-5', symbol: 'MSFT', side: 'BUY', type: 'MARKET', status: 'FILLED', quantity: 10, filledQty: 10, price: 419.51, avgFillPrice: 419.48, fee: 0.84, strategy: 'MA Crossover', createdAt: new Date(now - day * 2).toISOString() },
]

export const mockSentiments: SentimentResult[] = [
  { id: 'sent-1', text: 'Apple reports record Q2 earnings with strong iPhone sales growth in Asia Pacific region', score: 0.72, label: 'POSITIVE', keywords: ['record earnings', 'iPhone sales', 'Asia Pacific'], summary: 'Apple 财报超预期，iPhone 在亚太地区销量强劲增长，利好股价。', symbols: ['AAPL'], timestamp: new Date(now - hour).toISOString() },
  { id: 'sent-2', text: 'Federal Reserve signals potential rate pause amid cooling inflation data', score: 0.55, label: 'POSITIVE', keywords: ['rate pause', 'inflation', 'Fed'], summary: '美联储暗示可能暂停加息，通胀数据降温，市场情绪偏积极。', symbols: ['SPY', 'QQQ'], timestamp: new Date(now - hour * 4).toISOString() },
  { id: 'sent-3', text: 'NVIDIA faces new export restrictions to China on AI chip sales', score: -0.65, label: 'NEGATIVE', keywords: ['export restrictions', 'AI chips', 'China'], summary: 'NVIDIA 面临新的对华 AI 芯片出口限制，可能影响营收。', symbols: ['NVDA'], timestamp: new Date(now - hour * 8).toISOString() },
  { id: 'sent-4', text: 'Tesla announces new Gigafactory expansion plans in Southeast Asia', score: 0.48, label: 'POSITIVE', keywords: ['Gigafactory', 'expansion', 'Southeast Asia'], summary: '特斯拉宣布东南亚新超级工厂扩张计划，中长期利好。', symbols: ['TSLA'], timestamp: new Date(now - day).toISOString() },
]

export const mockRiskEvents: RiskEvent[] = [
  { id: 'risk-1', type: 'overheat', sentimentScore: 0.85, action: 'warn', reason: '市场情绪过热，AI 情绪分数 0.85 超过阈值 0.8，注意追高风险', timestamp: new Date(now - hour * 2).toISOString() },
  { id: 'risk-2', type: 'panic', sentimentScore: -0.82, action: 'reduce_position', reason: '市场恐慌情绪触发，AI 情绪分数 -0.82，自动降低持仓至 50%', timestamp: new Date(now - day).toISOString() },
]

export const equityCurveData = Array.from({ length: 60 }, (_, i) => {
  const date = new Date(now - (60 - i) * day)
  const progress = i / 59
  const trend = 100000 + progress * 24500
  const noise = Math.sin(i / 4) * 3000 + Math.sin(i / 7) * 1500 + (Math.random() - 0.5) * 2000
  const value = trend + noise
  return { date: date.toISOString().slice(0, 10), value: Math.round(value * 100) / 100 }
})

export const dailyPnlData = Array.from({ length: 30 }, (_, i) => {
  const date = new Date(now - (30 - i) * day)
  const pnl = (Math.random() - 0.45) * 2000
  return { date: date.toISOString().slice(0, 10), pnl: Math.round(pnl * 100) / 100 }
})
