import { useEffect, useState } from 'react'
import { Play, Square, Settings2, Plus, Brain, X } from 'lucide-react'
import { mockStrategies } from '../mock/data'
import { fetchStrategies, createStrategy, updateStrategy } from '../api'
import type { StrategyConfig } from '../types'

export default function Strategy() {
  const [strategies, setStrategies] = useState<StrategyConfig[]>(mockStrategies)
  const [selectedStrategy, setSelectedStrategy] = useState<StrategyConfig | null>(mockStrategies[0])
  const [showNewDialog, setShowNewDialog] = useState(false)
  const [showConfigDialog, setShowConfigDialog] = useState(false)
  const [toggling, setToggling] = useState<string | null>(null)

  const [newStrategy, setNewStrategy] = useState({
    name: 'ma_crossover',
    displayName: '',
    symbols: 'AAPL',
    shortPeriod: 5,
    longPeriod: 20,
  })

  useEffect(() => {
    fetchStrategies().then(data => {
      if (data && data.length > 0) {
        setStrategies(data)
        setSelectedStrategy(data[0])
      }
    }).catch(() => {})
  }, [])

  const handleToggle = async (s: StrategyConfig) => {
    if (toggling) return
    setToggling(s.id)
    try {
      await updateStrategy(s.id, {
        name: s.name,
        displayName: s.displayName,
        enabled: !s.enabled,
        symbols: s.symbols,
        params: Object.fromEntries(Object.entries(s.params).map(([k, v]) => [k, String(v)])),
        ai: { enabled: s.aiEnabled, signalWeight: s.aiWeight },
      })
      const updated = strategies.map(st =>
        st.id === s.id
          ? { ...st, enabled: !st.enabled, status: (!st.enabled ? 'RUNNING' : 'STOPPED') as StrategyConfig['status'] }
          : st
      )
      setStrategies(updated)
      if (selectedStrategy?.id === s.id) {
        setSelectedStrategy({ ...s, enabled: !s.enabled, status: (!s.enabled ? 'RUNNING' : 'STOPPED') as StrategyConfig['status'] })
      }
    } catch (err) {
      console.error('Failed to toggle strategy:', err)
    }
    setToggling(null)
  }

  const [createError, setCreateError] = useState('')

  const handleCreate = async () => {
    if (!newStrategy.displayName.trim()) {
      setCreateError('Display name is required')
      return
    }
    setCreateError('')
    try {
      const created = await createStrategy({
        name: newStrategy.name,
        displayName: newStrategy.displayName,
        symbols: newStrategy.symbols.split(',').map(s => s.trim()).filter(Boolean),
        params: newStrategy.name === 'ma_crossover'
          ? { shortPeriod: String(newStrategy.shortPeriod), longPeriod: String(newStrategy.longPeriod) }
          : { period: '14', oversold: '30', overbought: '70' },
        enabled: false,
        ai: { enabled: false, signalWeight: 0 },
      })
      const mapped: StrategyConfig = {
        id: created.id,
        name: created.name || newStrategy.name,
        displayName: created.displayName || newStrategy.displayName,
        enabled: false,
        status: 'STOPPED',
        symbols: newStrategy.symbols.split(',').map(s => s.trim()).filter(Boolean),
        params: newStrategy.name === 'ma_crossover'
          ? { shortPeriod: newStrategy.shortPeriod, longPeriod: newStrategy.longPeriod }
          : { period: 14, oversold: 30, overbought: 70 },
        aiEnabled: false,
        aiWeight: 0,
        pnl: 0,
        winRate: 0,
        createdAt: new Date().toISOString().slice(0, 10),
      }
      setStrategies(prev => [...prev, mapped])
      setSelectedStrategy(mapped)
      setShowNewDialog(false)
      setNewStrategy({ name: 'ma_crossover', displayName: '', symbols: 'AAPL', shortPeriod: 5, longPeriod: 20 })
    } catch (err) {
      console.error('Failed to create strategy:', err)
    }
  }

  const handleSaveConfig = async () => {
    if (!selectedStrategy) return
    try {
      await updateStrategy(selectedStrategy.id, {
        name: selectedStrategy.name,
        displayName: selectedStrategy.displayName,
        enabled: selectedStrategy.enabled,
        symbols: selectedStrategy.symbols,
        params: Object.fromEntries(Object.entries(selectedStrategy.params).map(([k, v]) => [k, String(v)])),
        ai: { enabled: selectedStrategy.aiEnabled, signalWeight: selectedStrategy.aiWeight },
      })
      const updated = { ...selectedStrategy }
      setStrategies(prev => prev.map(s => s.id === updated.id ? updated : s))
      setShowConfigDialog(false)
    } catch (err) {
      console.error('Failed to save config:', err)
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Strategy Management</h1>
        <button onClick={() => setShowNewDialog(true)} className="btn-primary flex items-center gap-2">
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
                  <button
                    onClick={() => handleToggle(selectedStrategy)}
                    disabled={!!toggling}
                    className="btn-ghost flex items-center gap-2 text-loss"
                  >
                    <Square size={14} /> {toggling === selectedStrategy.id ? 'Stopping...' : 'Stop'}
                  </button>
                ) : (
                  <button
                    onClick={() => handleToggle(selectedStrategy)}
                    disabled={!!toggling}
                    className="btn-primary flex items-center gap-2"
                  >
                    <Play size={14} /> {toggling === selectedStrategy.id ? 'Starting...' : 'Start'}
                  </button>
                )}
                <button
                  onClick={() => setShowConfigDialog(true)}
                  className="btn-ghost flex items-center gap-2"
                >
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
                  <button
                    onClick={() => {
                      const updated = { ...selectedStrategy, aiEnabled: !selectedStrategy.aiEnabled }
                      setSelectedStrategy(updated)
                      setStrategies(prev => prev.map(s => s.id === updated.id ? updated : s))
                    }}
                    className={`w-8 h-4 rounded-full transition-colors ${selectedStrategy.aiEnabled ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border-light)]'} relative cursor-pointer border-none`}
                  >
                    <div className={`absolute top-0.5 w-3 h-3 rounded-full bg-white transition-transform ${selectedStrategy.aiEnabled ? 'left-[18px]' : 'left-0.5'}`} />
                  </button>
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

      {/* New Strategy Dialog */}
      {showNewDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowNewDialog(false)}>
          <div className="card w-full max-w-md space-y-4" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="font-heading text-lg font-bold text-[var(--color-text-primary)]">New Strategy</h2>
              <button onClick={() => setShowNewDialog(false)} className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] bg-transparent border-none cursor-pointer">
                <X size={16} />
              </button>
            </div>
            <div>
              <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Strategy Type</label>
              <select
                value={newStrategy.name}
                onChange={e => setNewStrategy(prev => ({ ...prev, name: e.target.value }))}
                className="select-field w-full"
              >
                <option value="ma_crossover">MA Crossover</option>
                <option value="rsi_reversion">RSI Mean Reversion</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Display Name</label>
              <input
                type="text"
                value={newStrategy.displayName}
                onChange={e => { setNewStrategy(prev => ({ ...prev, displayName: e.target.value })); setCreateError('') }}
                placeholder="e.g. My MA Strategy"
                className={`input-field w-full ${createError ? 'border-[var(--color-loss)]' : ''}`}
              />
              {createError && <p className="text-xs text-loss mt-1">{createError}</p>}
            </div>
            <div>
              <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Symbols (comma separated)</label>
              <input
                type="text"
                value={newStrategy.symbols}
                onChange={e => setNewStrategy(prev => ({ ...prev, symbols: e.target.value }))}
                placeholder="AAPL, GOOGL, MSFT"
                className="input-field w-full"
              />
            </div>
            {newStrategy.name === 'ma_crossover' && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Short Period</label>
                  <input type="number" value={newStrategy.shortPeriod} onChange={e => setNewStrategy(prev => ({ ...prev, shortPeriod: Number(e.target.value) }))} className="input-field w-full" />
                </div>
                <div>
                  <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Long Period</label>
                  <input type="number" value={newStrategy.longPeriod} onChange={e => setNewStrategy(prev => ({ ...prev, longPeriod: Number(e.target.value) }))} className="input-field w-full" />
                </div>
              </div>
            )}
            <button onClick={handleCreate} className="btn-primary w-full">Create Strategy</button>
          </div>
        </div>
      )}

      {/* Config Dialog */}
      {showConfigDialog && selectedStrategy && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={() => setShowConfigDialog(false)}>
          <div className="card w-full max-w-md space-y-4" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="font-heading text-lg font-bold text-[var(--color-text-primary)]">Configure {selectedStrategy.displayName}</h2>
              <button onClick={() => setShowConfigDialog(false)} className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] bg-transparent border-none cursor-pointer">
                <X size={16} />
              </button>
            </div>
            <div>
              <label className="block text-xs text-[var(--color-text-muted)] mb-1.5">Symbols (comma separated)</label>
              <input
                type="text"
                value={selectedStrategy.symbols.join(', ')}
                onChange={e => setSelectedStrategy({
                  ...selectedStrategy,
                  symbols: e.target.value.split(',').map(s => s.trim()).filter(Boolean),
                })}
                className="input-field w-full"
              />
            </div>
            <div>
              <h3 className="text-xs text-[var(--color-text-muted)] mb-2">Parameters</h3>
              <div className="space-y-2">
                {Object.entries(selectedStrategy.params).map(([key, value]) => (
                  <div key={key} className="flex items-center gap-3">
                    <span className="text-xs text-[var(--color-text-secondary)] w-28">{key}</span>
                    <input
                      type="number"
                      value={value}
                      onChange={e => setSelectedStrategy({
                        ...selectedStrategy,
                        params: { ...selectedStrategy.params, [key]: Number(e.target.value) },
                      })}
                      className="input-field flex-1"
                    />
                  </div>
                ))}
              </div>
            </div>
            <div className="flex gap-3">
              <button onClick={() => setShowConfigDialog(false)} className="btn-ghost flex-1">Cancel</button>
              <button onClick={handleSaveConfig} className="btn-primary flex-1">Save</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
