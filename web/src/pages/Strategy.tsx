import { useEffect, useState } from 'react'
import { Play, Square, Settings2, Plus, Brain } from 'lucide-react'
import { mockStrategies } from '../mock/data'
import { fetchStrategies } from '../api'
import type { StrategyConfig } from '../types'

export default function Strategy() {
  const [strategies, setStrategies] = useState<StrategyConfig[]>(mockStrategies)
  const [selectedStrategy, setSelectedStrategy] = useState<StrategyConfig | null>(mockStrategies[0])

  useEffect(() => {
    fetchStrategies().then(data => {
      if (data && data.length > 0) {
        setStrategies(data)
        setSelectedStrategy(data[0])
      }
    }).catch(() => {})
  }, [])

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Strategy Management</h1>
        <button className="btn-primary flex items-center gap-2">
          <Plus size={14} /> New Strategy
        </button>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-1 space-y-3">
          {strategies.map(s => (
            <div
              key={s.id}
              onClick={() => setSelectedStrategy(s)}
              className={`card cursor-pointer transition-all ${
                selectedStrategy?.id === s.id
                  ? 'border-[var(--color-accent)] bg-[var(--color-elevated)]'
                  : 'hover:border-[var(--color-border-light)]'
              }`}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <div className={`w-2 h-2 rounded-full ${s.status === 'RUNNING' ? 'bg-[var(--color-profit)] animate-pulse-soft' : 'bg-[var(--color-text-muted)]'}`} />
                  <span className="text-sm font-semibold text-[var(--color-text-primary)]">{s.displayName}</span>
                </div>
                <span className={`badge ${s.status === 'RUNNING' ? 'badge-running' : 'badge-stopped'}`}>
                  {s.status}
                </span>
              </div>
              <div className="flex items-center justify-between text-xs text-[var(--color-text-muted)]">
                <span>{s.symbols.join(', ')}</span>
                <span className={`font-data ${s.pnl >= 0 ? 'text-profit' : 'text-loss'}`}>
                  {s.pnl >= 0 ? '+' : ''}{s.pnl.toLocaleString()}
                </span>
              </div>
            </div>
          ))}
        </div>

        {selectedStrategy && (
          <div className="col-span-2 card animate-slide-up">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="font-heading text-lg font-bold text-[var(--color-text-primary)]">
                  {selectedStrategy.displayName}
                </h2>
                <span className="text-xs text-[var(--color-text-muted)]">{selectedStrategy.name}</span>
              </div>
              <div className="flex items-center gap-2">
                {selectedStrategy.status === 'RUNNING' ? (
                  <button className="btn-ghost flex items-center gap-2 text-loss">
                    <Square size={14} /> Stop
                  </button>
                ) : (
                  <button className="btn-primary flex items-center gap-2">
                    <Play size={14} /> Start
                  </button>
                )}
                <button className="btn-ghost flex items-center gap-2">
                  <Settings2 size={14} /> Configure
                </button>
              </div>
            </div>

            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="card-elevated">
                <div className="text-xs text-[var(--color-text-muted)] mb-1">Win Rate</div>
                <div className="font-data text-lg font-semibold text-[var(--color-text-primary)]">
                  {(selectedStrategy.winRate * 100).toFixed(0)}%
                </div>
              </div>
              <div className="card-elevated">
                <div className="text-xs text-[var(--color-text-muted)] mb-1">Total PnL</div>
                <div className={`font-data text-lg font-semibold ${selectedStrategy.pnl >= 0 ? 'text-profit' : 'text-loss'}`}>
                  {selectedStrategy.pnl >= 0 ? '+' : ''}${selectedStrategy.pnl.toLocaleString()}
                </div>
              </div>
              <div className="card-elevated">
                <div className="text-xs text-[var(--color-text-muted)] mb-1">Symbols</div>
                <div className="font-data text-lg font-semibold text-[var(--color-text-primary)]">
                  {selectedStrategy.symbols.length}
                </div>
              </div>
            </div>

            <div className="mb-6">
              <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
                Strategy Parameters
              </h3>
              <div className="grid grid-cols-2 gap-3">
                {Object.entries(selectedStrategy.params).map(([key, value]) => (
                  <div key={key} className="card-elevated flex items-center justify-between">
                    <span className="text-xs text-[var(--color-text-secondary)]">{key}</span>
                    <span className="font-data text-sm font-medium text-[var(--color-text-primary)]">{value}</span>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 mb-3">
                <Brain size={14} className="text-[var(--color-accent-glow)]" />
                <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
                  AI Configuration
                </h3>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="card-elevated flex items-center justify-between">
                  <span className="text-xs text-[var(--color-text-secondary)]">AI Enabled</span>
                  <div className={`w-8 h-4 rounded-full transition-colors ${selectedStrategy.aiEnabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border-light)]'} relative cursor-pointer`}>
                    <div className={`absolute top-0.5 w-3 h-3 rounded-full bg-white transition-transform ${selectedStrategy.aiEnabled ? 'left-[18px]' : 'left-0.5'}`} />
                  </div>
                </div>
                <div className="card-elevated flex items-center justify-between">
                  <span className="text-xs text-[var(--color-text-secondary)]">AI Weight</span>
                  <span className="font-data text-sm font-medium text-[var(--color-accent-glow)]">
                    {(selectedStrategy.aiWeight * 100).toFixed(0)}%
                  </span>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
