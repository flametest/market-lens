import type {
  Tick, Candle, Signal, StrategyConfig, BacktestRun,
  Position, Order, SentimentResult, RiskEvent,
} from '../types'

const BASE = '/api/v1'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.message || `HTTP ${res.status}`)
  }
  const json = await res.json()
  return json.data !== undefined ? json.data : json
}

// Market
export async function fetchSymbols(q?: string): Promise<any[]> {
  const params = q ? `?q=${encodeURIComponent(q)}` : ''
  return request(`/market/symbols${params}`)
}

export async function fetchCandles(symbol: string, interval = '1d', start?: number, end?: number): Promise<Candle[]> {
  const params = new URLSearchParams({ interval })
  if (start) params.set('start', String(start))
  if (end) params.set('end', String(end))
  const raw: any[] = await request(`/market/symbols/${symbol}/candles?${params}`)
  return raw.map(c => ({
    time: Math.floor(c.timestamp / 1000),
    open: Number(c.open),
    high: Number(c.high),
    low: Number(c.low),
    close: Number(c.close),
    volume: Number(c.volume),
  }))
}

export async function fetchQuote(symbol: string): Promise<Tick | null> {
  try {
    const raw = await request(`/market/symbols/${symbol}/quote`)
    if (!raw) return null
    return {
      symbol: raw.symbol || symbol,
      price: Number(raw.price || 0),
      volume: Number(raw.volume || 0),
      change: Number(raw.change || 0),
      changePercent: Number(raw.changePercent || 0),
      high: Number(raw.high || 0),
      low: Number(raw.low || 0),
      open: Number(raw.open || 0),
      timestamp: raw.timestamp || Date.now(),
    }
  } catch {
    return null
  }
}

export async function fetchTicks(symbol: string): Promise<Tick | null> {
  try {
    const raw = await request(`/market/symbols/${symbol}/ticks`)
    if (!raw) return null
    return {
      symbol: raw.symbol,
      price: Number(raw.price),
      volume: Number(raw.volume),
      change: Number(raw.change),
      changePercent: Number(raw.changePercent),
      high: Number(raw.high),
      low: Number(raw.low),
      open: Number(raw.open),
      timestamp: raw.timestamp,
    }
  } catch {
    return null
  }
}

// Signals
export async function fetchSignals(symbol?: string, limit = 50): Promise<Signal[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  const path = symbol ? `/signals/${symbol}` : '/signals'
  const raw: any[] = await request(`${path}?${params}`)
  return (raw || []).map(s => ({
    ...s,
    price: Number(s.price),
    strength: Number(s.strength),
  }))
}

// Strategies
export async function fetchStrategies(): Promise<StrategyConfig[]> {
  const raw: any[] = await request('/strategies')
  return (raw || []).map(s => ({
    id: s.id,
    name: s.name,
    displayName: s.displayName || s.name,
    enabled: s.enabled,
    status: s.enabled ? 'RUNNING' as const : 'STOPPED' as const,
    symbols: s.symbols || [],
    params: s.params || {},
    aiEnabled: s.ai?.enabled || false,
    aiWeight: s.ai?.signalWeight || 0,
    pnl: 0,
    winRate: 0,
    createdAt: s.createdAt ? new Date(s.createdAt * 1000).toISOString().slice(0, 10) : '',
  }))
}

export async function createStrategy(strategy: {
  name: string
  displayName: string
  symbols: string[]
  params: Record<string, string>
  enabled: boolean
  ai: { enabled: boolean; signalWeight: number }
}): Promise<any> {
  return request('/strategies', {
    method: 'POST',
    body: JSON.stringify(strategy),
  })
}

export async function updateStrategy(id: string, strategy: {
  name: string
  displayName: string
  enabled: boolean
  symbols: string[]
  params: Record<string, any>
  ai: { enabled: boolean; signalWeight: number }
}): Promise<any> {
  return request(`/strategies/${id}`, {
    method: 'PUT',
    body: JSON.stringify(strategy),
  })
}

// Backtest
export async function fetchBacktests(): Promise<BacktestRun[]> {
  const raw: any[] = await request('/backtest')
  return (raw || []).map(mapBacktestRun)
}

export async function fetchBacktest(id: string): Promise<BacktestRun> {
  const raw = await request(`/backtest/${id}`)
  return mapBacktestRun(raw)
}

export async function runBacktest(config: any): Promise<BacktestRun> {
  const raw = await request('/backtest', {
    method: 'POST',
    body: JSON.stringify(config),
  })
  return mapBacktestRun(raw)
}

function mapBacktestRun(raw: any): BacktestRun {
  const r = raw.result || {}
  return {
    id: raw.id,
    strategyName: raw.config?.strategy?.displayName || raw.config?.strategy?.name || '',
    symbol: raw.config?.symbol || '',
    startDate: raw.config?.startDate ? new Date(raw.config.startDate).toISOString().slice(0, 10) : '',
    endDate: raw.config?.endDate ? new Date(raw.config.endDate).toISOString().slice(0, 10) : '',
    interval: raw.config?.interval || '1d',
    aiMode: raw.aiMode || 'off',
    status: raw.status === 'completed' ? 'COMPLETED' : raw.status === 'failed' ? 'FAILED' : 'RUNNING',
    totalReturn: Number(r.totalReturn || 0) * 100,
    annualReturn: Number(r.annualReturn || 0) * 100,
    maxDrawdown: Number(r.maxDrawdown || 0) * 100,
    sharpeRatio: Number(r.sharpeRatio || 0),
    winRate: Number(r.winRate || 0),
    totalTrades: r.totalTrades || 0,
    profitTrades: r.profitTrades || 0,
    lossTrades: r.lossTrades || 0,
    createdAt: raw.createdAt ? new Date(raw.createdAt * 1000).toISOString().slice(0, 10) : '',
  }
}

// Trading
export async function fetchAccount(): Promise<any> {
  return request('/trading/account')
}

export async function fetchPositions(): Promise<Position[]> {
  const raw: any[] = await request('/trading/positions')
  return (raw || []).map(p => ({
    symbol: p.symbol,
    name: p.symbol,
    side: p.side,
    quantity: Number(p.quantity),
    avgPrice: Number(p.avgPrice),
    currentPrice: Number(p.avgPrice),
    unrealizedPnl: Number(p.unrealizedPnl),
    unrealizedPnlPercent: Number(p.unrealizedPnl),
  }))
}

export async function fetchOrders(): Promise<Order[]> {
  const raw: any[] = await request('/trading/orders')
  return (raw || []).map(o => ({
    ...o,
    quantity: Number(o.quantity),
    filledQty: Number(o.filledQty),
    price: Number(o.price),
    avgFillPrice: Number(o.avgFillPrice),
    fee: Number(o.fee),
    createdAt: o.createdAt ? new Date(Number(o.createdAt) * 1000).toISOString() : '',
  }))
}

export async function submitOrder(order: { symbol: string; side: string; type: string; quantity: number; price: number }): Promise<Order> {
  return request('/trading/orders', {
    method: 'POST',
    body: JSON.stringify(order),
  })
}

export async function cancelOrder(id: string): Promise<any> {
  return request(`/trading/orders/${id}`, {
    method: 'DELETE',
  })
}

// AI
export async function analyzeSentiment(text: string): Promise<SentimentResult> {
  const raw = await request('/ai/analyze', {
    method: 'POST',
    body: JSON.stringify({ text }),
  })
  return {
    ...raw,
    timestamp: raw.timestamp ? new Date(raw.timestamp * 1000).toISOString() : new Date().toISOString(),
  }
}

export async function fetchSentiments(symbol?: string, limit = 50): Promise<SentimentResult[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (symbol) params.set('symbol', symbol)
  const raw: any[] = await request(`/ai/sentiment?${params}`)
  return (raw || []).map(s => ({
    ...s,
    timestamp: s.timestamp ? new Date(s.timestamp * 1000).toISOString() : '',
  }))
}

export async function fetchRiskEvents(limit = 50): Promise<RiskEvent[]> {
  const raw: any[] = await request(`/ai/risk-events?limit=${limit}`)
  return raw || []
}

export async function fetchAIConfig(): Promise<any> {
  return request('/ai/config')
}

export async function updateAIConfig(config: any): Promise<any> {
  return request('/ai/config', {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}
