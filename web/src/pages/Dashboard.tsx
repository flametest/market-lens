import { useEffect, useState } from 'react'
import { TrendingUp, TrendingDown, Activity, Brain, Zap, Target } from 'lucide-react'
import MetricCard from '../components/common/MetricCard'
import { mockAccount, mockWatchlist, mockSignals, mockStrategies, mockSentiments } from '../mock/data'
import { fetchSignals, fetchStrategies, fetchSentiments, fetchSymbols, fetchQuote } from '../api'
import { getWSClient } from '../ws'
import type { Tick, Signal, StrategyConfig, SentimentResult } from '../types'

export default function Dashboard() {
  const [watchlist, setWatchlist] = useState<Tick[]>(mockWatchlist)
  const [signals, setSignals] = useState<Signal[]>(mockSignals)
  const [strategies, setStrategies] = useState<StrategyConfig[]>(mockStrategies)
  const [sentiments, setSentiments] = useState<SentimentResult[]>(mockSentiments)
  const account = mockAccount

  useEffect(() => {
    fetchSignals().then(data => { if (data && data.length > 0) setSignals(data) }).catch(() => {})
    fetchStrategies().then(data => { if (data && data.length > 0) setStrategies(data) }).catch(() => {})
    fetchSentiments().then(data => { if (data && data.length > 0) setSentiments(data) }).catch(() => {})

    fetchSymbols().then(raw => {
      if (!raw || raw.length === 0) return
      Promise.all(raw.map((s: any) => fetchQuote(s.code).catch(() => null)))
        .then(ticks => {
          const valid = ticks.filter((t): t is Tick => t !== null && t.price > 0)
          if (valid.length > 0) setWatchlist(valid)
        })
    }).catch(() => {})
  }, [])

  useEffect(() => {
    const ws = getWSClient()
    const unsub = ws.on('tick', (data: any) => {
      setWatchlist(prev => prev.map(t =>
        t.symbol === data.symbol
          ? { ...t, price: Number(data.price) || t.price, volume: Number(data.volume) || t.volume, high: Number(data.high) || t.high, low: Number(data.low) || t.low }
          : t
      ))
    })
    const unsubSignal = ws.on('signal', (data: any) => {
      setSignals(prev => [{
        id: data.id || String(Date.now()),
        symbol: data.symbol,
        source: data.source || '',
        type: data.type || 'NEUTRAL',
        strength: Number(data.strength) || 0,
        price: Number(data.price) || 0,
        timestamp: data.timestamp ? data.timestamp / 1000000 : Date.now(),
      }, ...prev].slice(0, 50))
    })
    return () => { unsub(); unsubSignal() }
  }, [])

  const topGainers = [...watchlist].sort((a, b) => b.changePercent - a.changePercent).slice(0, 3)
  const topLosers = [...watchlist].sort((a, b) => a.changePercent - b.changePercent).slice(0, 3)
  const recentSignals = signals.slice(0, 5)
  const activeStrategies = strategies.filter(s => s.status === 'RUNNING')

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Dashboard</h1>
        <span className="text-xs text-[var(--color-text-muted)]">
          {new Date().toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
        </span>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <MetricCard
          label="Total Value"
          value={`$${account.totalValue.toLocaleString()}`}
          change={2.45}
          icon={<TrendingUp size={14} />}
        />
        <MetricCard
          label="Daily PnL"
          value={`$${account.dailyPnl.toLocaleString()}`}
          change={1.23}
          icon={<Activity size={14} />}
        />
        <MetricCard
          label="Active Signals"
          value={String(signals.length)}
          icon={<Zap size={14} />}
        />
        <MetricCard
          label="Running Strategies"
          value={`${activeStrategies.length}/${strategies.length}`}
          icon={<Target size={14} />}
        />
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="card col-span-2">
          <h2 className="font-heading text-sm font-semibold mb-4 text-[var(--color-text-primary)]">
            Recent Signals
          </h2>
          <table>
            <thead>
              <tr>
                <th>Symbol</th>
                <th>Source</th>
                <th>Signal</th>
                <th>Strength</th>
                <th>Price</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {recentSignals.map((s) => (
                <tr key={s.id}>
                  <td className="font-data text-[var(--color-text-primary)] font-medium">{s.symbol}</td>
                  <td>{s.source}</td>
                  <td>
                    <span className={`badge ${s.type === 'BUY' ? 'badge-buy' : s.type === 'SELL' ? 'badge-sell' : 'badge-neutral'}`}>
                      {s.type}
                    </span>
                  </td>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="w-16 h-1.5 bg-[var(--color-border)] rounded-full overflow-hidden">
                        <div
                          className={`h-full rounded-full ${s.type === 'BUY' ? 'bg-[var(--color-profit)]' : 'bg-[var(--color-loss)]'}`}
                          style={{ width: `${s.strength * 100}%` }}
                        />
                      </div>
                      <span className="font-data text-xs">{(s.strength * 100).toFixed(0)}%</span>
                    </div>
                  </td>
                  <td className="font-data">${s.price.toFixed(2)}</td>
                  <td className="text-xs">{new Date(s.timestamp).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="space-y-4">
          <div className="card">
            <h2 className="font-heading text-sm font-semibold mb-3 text-[var(--color-text-primary)]">
              Top Gainers
            </h2>
            <div className="space-y-3">
              {topGainers.map((t) => (
                <div key={t.symbol} className="flex items-center justify-between">
                  <div>
                    <span className="font-data text-sm font-medium text-[var(--color-text-primary)]">{t.symbol}</span>
                    <div className="text-xs text-[var(--color-text-muted)]">${t.price.toFixed(2)}</div>
                  </div>
                  <div className="flex items-center gap-1 text-profit">
                    <TrendingUp size={12} />
                    <span className="font-data text-xs font-medium">+{t.changePercent.toFixed(2)}%</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="card">
            <h2 className="font-heading text-sm font-semibold mb-3 text-[var(--color-text-primary)]">
              Top Losers
            </h2>
            <div className="space-y-3">
              {topLosers.map((t) => (
                <div key={t.symbol} className="flex items-center justify-between">
                  <div>
                    <span className="font-data text-sm font-medium text-[var(--color-text-primary)]">{t.symbol}</span>
                    <div className="text-xs text-[var(--color-text-muted)]">${t.price.toFixed(2)}</div>
                  </div>
                  <div className="flex items-center gap-1 text-loss">
                    <TrendingDown size={12} />
                    <span className="font-data text-xs font-medium">{t.changePercent.toFixed(2)}%</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="card">
          <h2 className="font-heading text-sm font-semibold mb-4 text-[var(--color-text-primary)]">
            Active Strategies
          </h2>
          <div className="space-y-3">
            {strategies.map((s) => (
              <div key={s.id} className="flex items-center justify-between p-3 rounded-lg bg-[var(--color-elevated)] border border-[var(--color-border)]">
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${s.status === 'RUNNING' ? 'bg-[var(--color-profit)] animate-pulse-soft' : 'bg-[var(--color-text-muted)]'}`} />
                  <div>
                    <span className="text-sm font-medium text-[var(--color-text-primary)]">{s.displayName}</span>
                    <div className="text-xs text-[var(--color-text-muted)]">{s.symbols.join(', ')}</div>
                  </div>
                </div>
                <div className="text-right">
                  <span className={`badge ${s.status === 'RUNNING' ? 'badge-running' : 'badge-stopped'}`}>
                    {s.status}
                  </span>
                  <div className={`font-data text-xs mt-1 ${s.pnl >= 0 ? 'text-profit' : 'text-loss'}`}>
                    {s.pnl >= 0 ? '+' : ''}${s.pnl.toLocaleString()}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Brain size={14} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)]">
              AI Sentiment
            </h2>
          </div>
          <div className="space-y-3">
            {sentiments.slice(0, 3).map((s) => (
              <div key={s.id} className="p-3 rounded-lg bg-[var(--color-elevated)] border border-[var(--color-border)]">
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-xs font-medium text-[var(--color-text-primary)]">
                    {s.symbols.join(', ')}
                  </span>
                  <span className={`badge ${s.label === 'POSITIVE' ? 'badge-positive' : 'badge-negative'}`}>
                    {s.score > 0 ? '+' : ''}{s.score.toFixed(2)}
                  </span>
                </div>
                <p className="text-xs text-[var(--color-text-secondary)] line-clamp-2">
                  {s.summary}
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
