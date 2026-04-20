export type SignalType = 'BUY' | 'SELL' | 'NEUTRAL'
export type OrderSide = 'BUY' | 'SELL'
export type OrderType = 'MARKET' | 'LIMIT'
export type OrderStatus = 'NEW' | 'FILLED' | 'CANCELLED' | 'REJECTED'
export type StrategyStatus = 'RUNNING' | 'STOPPED' | 'ERROR'
export type BacktestStatus = 'RUNNING' | 'COMPLETED' | 'FAILED'
export type SentimentLabel = 'POSITIVE' | 'NEGATIVE' | 'NEUTRAL'
export type AIMode = 'off' | 'on' | 'compare'
export type Interval = '1m' | '5m' | '15m' | '30m' | '1h' | '4h' | '1d' | '1w' | '1M'

export interface SymbolInfo {
  code: string
  name: string
  market: string
  exchange: string
  type: string
  currency: string
  enabled: boolean
}

export interface Tick {
  symbol: string
  price: number
  volume: number
  change: number
  changePercent: number
  high: number
  low: number
  open: number
  timestamp: number
}

export interface Candle {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface Signal {
  id: string
  symbol: string
  source: string
  type: SignalType
  strength: number
  price: number
  timestamp: number
}

export interface StrategyConfig {
  id: string
  name: string
  displayName: string
  enabled: boolean
  status: StrategyStatus
  symbols: string[]
  params: Record<string, number>
  aiEnabled: boolean
  aiWeight: number
  pnl: number
  winRate: number
  createdAt: string
}

export interface BacktestRun {
  id: string
  strategyName: string
  symbol: string
  startDate: string
  endDate: string
  interval: Interval
  aiMode: AIMode
  status: BacktestStatus
  totalReturn: number
  annualReturn: number
  maxDrawdown: number
  sharpeRatio: number
  winRate: number
  totalTrades: number
  profitTrades: number
  lossTrades: number
  createdAt: string
}

export interface Position {
  symbol: string
  name: string
  side: OrderSide
  quantity: number
  avgPrice: number
  currentPrice: number
  unrealizedPnl: number
  unrealizedPnlPercent: number
}

export interface Order {
  id: string
  symbol: string
  side: OrderSide
  type: OrderType
  status: OrderStatus
  quantity: number
  filledQty: number
  price: number
  avgFillPrice: number
  fee: number
  strategy: string
  createdAt: string
}

export interface AccountInfo {
  cash: number
  totalValue: number
  unrealizedPnl: number
  realizedPnl: number
  dailyPnl: number
  currency: string
}

export interface SentimentResult {
  id: string
  text: string
  score: number
  label: SentimentLabel
  keywords: string[]
  summary: string
  symbols: string[]
  timestamp: string
}

export interface RiskEvent {
  id: string
  type: 'panic' | 'overheat' | 'pause'
  sentimentScore: number
  action: string
  reason: string
  timestamp: string
}
