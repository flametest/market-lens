import { useState } from 'react'
import { Play, Download, BarChart3 } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'
import { mockBacktests, equityCurveData } from '../mock/data'
import type { AIMode } from '../types'

const aiModeLabels: Record<AIMode, string> = {
  off: 'Technical Only',
  on: 'Tech + AI',
  compare: 'Compare',
}

export default function Backtest() {
  const [selectedBt, setSelectedBt] = useState(mockBacktests[0])

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
            <select className="select-field w-full">
              <option>AAPL</option><option>GOOGL</option><option>MSFT</option><option>NVDA</option><option>TSLA</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Strategy</label>
            <select className="select-field w-full">
              <option>MA Crossover</option><option>RSI Mean Reversion</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Date Range</label>
            <div className="flex gap-2">
              <input type="date" className="input-field text-xs" defaultValue="2025-01-01" />
              <input type="date" className="input-field text-xs" defaultValue="2026-01-01" />
            </div>
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">AI Mode</label>
            <select className="select-field w-full">
              <option value="off">Technical Only</option>
              <option value="on">Tech + AI</option>
              <option value="compare">Compare</option>
            </select>
          </div>
        </div>
        <div className="grid grid-cols-4 gap-4 mb-4">
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Initial Cash</label>
            <input type="text" className="input-field" defaultValue="100000" />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Commission</label>
            <input type="text" className="input-field" defaultValue="0.001" />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Slippage</label>
            <input type="text" className="input-field" defaultValue="0.001" />
          </div>
          <div className="flex items-end">
            <button className="btn-primary flex items-center gap-2 w-full justify-center">
              <Play size={14} /> Run Backtest
            </button>
          </div>
        </div>
      </div>

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
              <button className="btn-ghost text-xs flex items-center gap-1.5">
                <Download size={12} /> Export
              </button>
            </div>
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={equityCurveData}>
                <defs>
                  <linearGradient id="equityGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#6366f1" stopOpacity={0.2} />
                    <stop offset="100%" stopColor="#6366f1" stopOpacity={0} />
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
                  itemStyle={{ color: '#6366f1' }}
                  formatter={(v) => [`$${Number(v).toLocaleString()}`, 'Equity']}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#6366f1"
                  strokeWidth={2}
                  fill="url(#equityGrad)"
                />
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
                {mockBacktests.map(bt => (
                  <div
                    key={bt.id}
                    onClick={() => setSelectedBt(bt)}
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
