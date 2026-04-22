import { useEffect, useState, useRef } from 'react'
import { Play, Download, BarChart3, ChevronDown, ChevronUp } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend } from 'recharts'
import { mockBacktests, equityCurveData } from '../mock/data'
import { fetchBacktests, runBacktest } from '../api'
import type { AIMode, BacktestRun } from '../types'

const aiModeLabels: Record<AIMode, string> = {
  off: 'Technical Only',
  on: 'Tech + AI',
  compare: 'Compare',
}

export default function Backtest() {
  const [backtests, setBacktests] = useState<BacktestRun[]>(mockBacktests)
  const [selectedBt, setSelectedBt] = useState<BacktestRun>(mockBacktests[0])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [equity, setEquity] = useState(equityCurveData)
  const [showTrades, setShowTrades] = useState(false)
  const formRef = useRef<HTMLFormElement>(null)

  // Form state
  const [formSymbol, setFormSymbol] = useState('AAPL')
  const [formStrategy, setFormStrategy] = useState('ma_crossover')
  const [formStartDate, setFormStartDate] = useState('2025-01-01')
  const [formEndDate, setFormEndDate] = useState('2026-01-01')
  const [formAiMode, setFormAiMode] = useState<AIMode>('off')
  const [formCash, setFormCash] = useState('100000')
  const [formCommission, setFormCommission] = useState('0.001')
  const [formSlippage, setFormSlippage] = useState('0.001')

  // Trades from raw backtest data
  const [trades, setTrades] = useState<any[]>([])
  const [compareEquity, setCompareEquity] = useState<any[]>([])

  useEffect(() => {
    fetchBacktests().then(data => {
      if (data && data.length > 0) {
        setBacktests(data)
        selectBacktest(data[0])
      }
    }).catch(() => {})
  }, [])

  const selectBacktest = (bt: BacktestRun) => {
    setSelectedBt(bt)
    // Fetch raw backtest data for trades
    if (bt.id) {
      fetchBacktest(bt.id).then((raw: any) => {
        setTrades(raw?.result?.trades || [])
        if (raw?.result?.equityCurve) {
          setEquity(raw.result.equityCurve.map((p: any) => ({
            date: new Date(p.timestamp * 1000).toISOString().slice(0, 10),
            value: Number(p.value),
          })))
        }
        if (raw?.compareResult?.equityCurve) {
          setCompareEquity(raw.compareResult.equityCurve.map((p: any) => ({
            date: new Date(p.timestamp * 1000).toISOString().slice(0, 10),
            compare: Number(p.value),
          })))
        } else {
          setCompareEquity([])
        }
      }).catch(() => {})
    }
  }

  const handleRun = async () => {
    setLoading(true)
    setError('')
    try {
      const result = await runBacktest({
        symbol: formSymbol,
        strategy: { name: formStrategy },
        startDate: new Date(formStartDate).getTime(),
        endDate: new Date(formEndDate).getTime(),
        interval: '1d',
        initialCash: formCash,
        commission: formCommission,
        slippage: formSlippage,
        risk: {},
        aiMode: formAiMode,
      })
      setBacktests(prev => [result, ...prev])
      selectBacktest(result)
    } catch (err: any) {
      setError(err?.message || 'Backtest failed. Ensure market data is available.')
    }
    setLoading(false)
  }

  const handleExport = () => {
    if (!selectedBt) return
    const blob = new Blob([JSON.stringify({ backtest: selectedBt, trades }, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `backtest-${selectedBt.id || 'result'}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // Merge equity + compare for dual-curve chart
  const mergedEquity = compareEquity.length > 0
    ? equity.map((e, i) => ({ ...e, ...(compareEquity[i] || {}) }))
    : equity

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Backtest</h1>
      </div>

      <div className="card">
        <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)] mb-4">
          New Backtest
        </h2>
        <div className="grid grid-cols-4 gap-4 mb-4">
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Symbol</label>
            <select className="select-field w-full" value={formSymbol} onChange={e => setFormSymbol(e.target.value)}>
              <option>AAPL</option><option>GOOGL</option><option>MSFT</option><option>NVDA</option><option>TSLA</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Strategy</label>
            <select className="select-field w-full" value={formStrategy} onChange={e => setFormStrategy(e.target.value)}>
              <option value="ma_crossover">MA Crossover</option>
              <option value="rsi_reversion">RSI Mean Reversion</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Date Range</label>
            <div className="flex gap-2">
              <input type="date" className="input-field text-xs" value={formStartDate} onChange={e => setFormStartDate(e.target.value)} />
              <input type="date" className="input-field text-xs" value={formEndDate} onChange={e => setFormEndDate(e.target.value)} />
            </div>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">AI Mode</label>
            <select className="select-field w-full" value={formAiMode} onChange={e => setFormAiMode(e.target.value as AIMode)}>
              <option value="off">Technical Only</option>
              <option value="on">Tech + AI</option>
              <option value="compare">Compare</option>
            </select>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-4 mb-4">
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Initial Cash</label>
            <input type="text" className="input-field" value={formCash} onChange={e => setFormCash(e.target.value)} />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Commission</label>
            <input type="text" className="input-field" value={formCommission} onChange={e => setFormCommission(e.target.value)} />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Slippage</label>
            <input type="text" className="input-field" value={formSlippage} onChange={e => setFormSlippage(e.target.value)} />
          </div>
          <div className="flex items-end">
            <button onClick={handleRun} disabled={loading} className="btn-primary flex items-center gap-2 w-full justify-center">
              <Play size={14} /> {loading ? 'Running...' : 'Run Backtest'}
            </button>
          </div>
        </div>
      </div>

      {error && (
        <div className="px-4 py-3 rounded-lg bg-[var(--color-loss)]/10 border border-[var(--color-loss)]/20 text-sm text-[var(--color-loss)]">
          {error}
        </div>
      )}

      {selectedBt && (
        <div className="animate-slide-up">
          <div className="flex items-center gap-2 mb-4">
            <BarChart3 size={16} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-lg font-bold text-[var(--color-text-primary)]">
              {selectedBt.strategyName} - {selectedBt.symbol}
            </h2>
            <span className={`badge ${selectedBt.aiMode === 'off' ? 'badge-neutral' : 'badge-running'}`}>
              {aiModeLabels[selectedBt.aiMode]}
            </span>
          </div>

          <div className="grid grid-cols-5 gap-4 mb-4">
            <MetricSmall label="Total Return" value={`${selectedBt.totalReturn.toFixed(1)}%`} positive={selectedBt.totalReturn >= 0} />
            <MetricSmall label="Annual Return" value={`${selectedBt.annualReturn.toFixed(1)}%`} positive={selectedBt.annualReturn >= 0} />
            <MetricSmall label="Max Drawdown" value={`${selectedBt.maxDrawdown.toFixed(1)}%`} positive={false} />
            <MetricSmall label="Sharpe Ratio" value={selectedBt.sharpeRatio.toFixed(2)} positive={selectedBt.sharpeRatio >= 1} />
            <MetricSmall label="Win Rate" value={`${(selectedBt.winRate * 100).toFixed(0)}%`} positive={selectedBt.winRate >= 0.5} />
          </div>

          <div className="card mb-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                Equity Curve
              </h3>
              <button onClick={handleExport} className="btn-ghost text-xs flex items-center gap-1.5">
                <Download size={12} /> Export JSON
              </button>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={mergedEquity}>
                <defs>
                  <linearGradient id="equityGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#6366f1" stopOpacity={0.2} />
                    <stop offset="100%" stopColor="#6366f1" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="compareGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#f59e0b" stopOpacity={0.2} />
                    <stop offset="100%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e2433" />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 10, fill: '#4a5068' }}
                  tickLine={false}
                  axisLine={{ stroke: '#1e2433' }}
                  tickFormatter={(v: string) => v.slice(5)}
                />
                <YAxis
                  tick={{ fontSize: 10, fill: '#4a5068' }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v: number) => `$${(v / 1000).toFixed(0)}k`}
                />
                <Tooltip
                  contentStyle={{
                    background: '#161a25',
                    border: '1px solid #1e2433',
                    borderRadius: '8px',
                    fontSize: '12px',
                  }}
                  labelStyle={{ color: '#7a8299' }}
                  formatter={(v) => [`$${Number(v).toLocaleString()}`]}
                />
                {compareEquity.length > 0 && <Legend />}
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#6366f1"
                  strokeWidth={2}
                  fill="url(#equityGrad)"
                  name="Technical"
                />
                {compareEquity.length > 0 && (
                  <Area
                    type="monotone"
                    dataKey="compare"
                    stroke="#f59e0b"
                    strokeWidth={2}
                    fill="url(#compareGrad)"
                    name="Tech + AI"
                  />
                )}
              </AreaChart>
            </ResponsiveContainer>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="card">
              <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
                Trade Statistics
              </h3>
              <div className="space-y-2">
                <StatRow label="Total Trades" value={selectedBt.totalTrades} />
                <StatRow label="Profit Trades" value={selectedBt.profitTrades} color="profit" />
                <StatRow label="Loss Trades" value={selectedBt.lossTrades} color="loss" />
              </div>
            </div>

            <div className="card">
              <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
                History
              </h3>
              <div className="space-y-2">
                {backtests.map(bt => (
                  <div
                    key={bt.id}
                    onClick={() => selectBacktest(bt)}
                    className={`flex items-center justify-between p-2.5 rounded-lg cursor-pointer transition-all ${
                      selectedBt.id === bt.id
                        ? 'bg-[var(--color-accent)]/10 border border-[var(--color-accent)]/30'
                        : 'hover:bg-[var(--color-elevated)]'
                    }`}
                  >
                    <div>
                      <span className="text-xs font-medium text-[var(--color-text-primary)]">{bt.strategyName}</span>
                      <span className="text-xs text-[var(--color-text-muted)] ml-2">{bt.symbol}</span>
                    </div>
                    <span className={`font-data text-xs font-medium ${bt.totalReturn >= 0 ? 'text-profit' : 'text-loss'}`}>
                      {bt.totalReturn >= 0 ? '+' : ''}{bt.totalReturn.toFixed(1)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Trade Details */}
          {trades.length > 0 && (
            <div className="card mt-4">
              <button
                onClick={() => setShowTrades(v => !v)}
                className="flex items-center gap-2 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider border-none bg-transparent cursor-pointer w-full text-left p-0"
              >
                Trade Details ({trades.length})
                {showTrades ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
              </button>
              {showTrades && (
                <div className="mt-3 overflow-x-auto">
                  <table>
                    <thead>
                      <tr>
                        <th>Symbol</th>
                        <th>Side</th>
                        <th>Entry Time</th>
                        <th>Exit Time</th>
                        <th>Entry Price</th>
                        <th>Exit Price</th>
                        <th>Qty</th>
                        <th>PnL</th>
                      </tr>
                    </thead>
                    <tbody>
                      {trades.map((t: any, i: number) => (
                        <tr key={i}>
                          <td className="font-data text-sm">{t.symbol}</td>
                          <td>
                            <span className={`badge ${t.side === 'BUY' ? 'badge-buy' : 'badge-sell'}`}>{t.side}</span>
                          </td>
                          <td className="text-xs">{new Date(t.entryTime / 1000000).toLocaleDateString()}</td>
                          <td className="text-xs">{new Date(t.exitTime / 1000000).toLocaleDateString()}</td>
                          <td className="font-data text-sm">${Number(t.entryPrice).toFixed(2)}</td>
                          <td className="font-data text-sm">${Number(t.exitPrice).toFixed(2)}</td>
                          <td className="font-data text-sm">{Number(t.quantity).toFixed(0)}</td>
                          <td className={`font-data text-sm font-medium ${Number(t.pnl) >= 0 ? 'text-profit' : 'text-loss'}`}>
                            {Number(t.pnl) >= 0 ? '+' : ''}{Number(t.pnl).toFixed(2)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function MetricSmall({ label, value, positive }: { label: string; value: string; positive: boolean }) {
  return (
    <div className="card">
      <div className="text-xs text-[var(--color-text-muted)] mb-1">{label}</div>
      <div className={`font-data text-lg font-semibold ${positive ? 'text-profit' : 'text-loss'}`}>
        {value}
      </div>
    </div>
  )
}

function StatRow({ label, value, color }: { label: string; value: number; color?: string }) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-xs text-[var(--color-text-secondary)]">{label}</span>
      <span className={`font-data text-sm font-medium ${color === 'profit' ? 'text-profit' : color === 'loss' ? 'text-loss' : 'text-[var(--color-text-primary)]'}`}>
        {value}
      </span>
    </div>
  )
}
