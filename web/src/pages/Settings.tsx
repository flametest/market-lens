import { useEffect, useState } from 'react'
import { Database, Shield, Bell, Server } from 'lucide-react'

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

type DataSource = {
  name: string
  market: string
  type: string
  status: string
  enabled: boolean
  switching?: boolean
}

export default function Settings() {
  const [dataSources, setDataSources] = useState<DataSource[]>([])
  const [switching, setSwitching] = useState<string | null>(null)

  const [systemMetrics] = useState([
    { label: 'CPU Usage', value: '12.3%', color: 'profit' },
    { label: 'Memory', value: '1.2 GB / 2 GB', color: 'profit' },
    { label: 'Goroutines', value: '156', color: 'text-[var(--color-text-primary)]' },
    { label: 'Event Bus', value: 'Connected', color: 'profit' },
    { label: 'ClickHouse', value: 'Healthy', color: 'profit' },
    { label: 'Redis', value: 'Connected', color: 'profit' },
  ])

  const [riskRules, setRiskRules] = useState([
    { name: 'Max Position Size', value: '1000 shares', enabled: true },
    { name: 'Max Drawdown', value: '15%', enabled: true },
    { name: 'Daily Trade Limit', value: '50', enabled: true },
    { name: 'Max Order Amount', value: '$50,000', enabled: true },
    { name: 'Min Trade Interval', value: '30s', enabled: true },
  ])

  const [notifications, setNotifications] = useState([
    { label: 'Trade Signals', desc: 'Buy/Sell signal alerts', enabled: true },
    { label: 'Order Updates', desc: 'Fill and status changes', enabled: true },
    { label: 'AI Risk Warnings', desc: 'Panic and overheat alerts', enabled: true },
    { label: 'System Alerts', desc: 'Connection and health events', enabled: false },
    { label: 'Backtest Complete', desc: 'Notify when backtest finishes', enabled: true },
  ])

  useEffect(() => {
    loadProvider()
  }, [])

  const loadProvider = async () => {
    try {
      const data = await request<{ active: string; providers: string[] }>('/settings/provider')
      const providerMeta: Record<string, { market: string; type: string }> = {
        twelvedata: { market: 'US Stocks', type: 'WebSocket + REST' },
        finnhub: { market: 'US Stocks', type: 'WebSocket + REST' },
      }
      const sources: DataSource[] = (data.providers || []).map(name => ({
        name,
        market: providerMeta[name]?.market || 'Global',
        type: providerMeta[name]?.type || 'REST',
        status: name === data.active ? 'connected' : 'standby',
        enabled: name === data.active,
      }))
      setDataSources(sources)
    } catch {
      setDataSources([
        { name: 'twelvedata', market: 'US Stocks', type: 'WebSocket + REST', status: 'connected', enabled: true },
      ])
    }
  }

  const toggleDataSource = async (name: string) => {
    const ds = dataSources.find(d => d.name === name)
    if (!ds || ds.enabled || switching) return

    setSwitching(name)
    setDataSources(prev => prev.map(d => ({ ...d, switching: d.name === name })))
    try {
      await request('/settings/provider', {
        method: 'PUT',
        body: JSON.stringify({ name }),
      })
      await loadProvider()
    } catch (err) {
      console.error('Failed to switch provider:', err)
    }
    setSwitching(null)
  }

  const toggleRiskRule = (index: number) => {
    setRiskRules(prev => prev.map((r, i) =>
      i === index ? { ...r, enabled: !r.enabled } : r
    ))
  }

  const toggleNotification = (index: number) => {
    setNotifications(prev => prev.map((n, i) =>
      i === index ? { ...n, enabled: !n.enabled } : n
    ))
  }

  const Toggle = ({ on, onToggle, loading }: { on: boolean; onToggle: () => void; loading?: boolean }) => (
    <button
      onClick={onToggle}
      disabled={loading}
      className={`w-8 h-4 rounded-full transition-colors ${on ? 'bg-[var(--color-accent)]' : 'bg-[var(--color-border-light)]'} relative cursor-pointer border-none ${loading ? 'opacity-50' : ''}`}
    >
      <div className={`absolute top-0.5 w-3 h-3 rounded-full bg-white transition-transform ${on ? 'left-[18px]' : 'left-0.5'}`} />
    </button>
  )

  return (
    <div className="space-y-6 animate-fade-in">
      <h1 className="font-heading text-2xl font-bold tracking-tight">Settings</h1>

      <div className="grid grid-cols-2 gap-4">
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Database size={14} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)]">
              Data Sources
            </h2>
          </div>
          <div className="space-y-3">
            {dataSources.map((ds) => (
              <div key={ds.name} className="card-elevated">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${ds.status === 'connected' ? 'bg-[var(--color-profit)] animate-pulse-soft' : 'bg-[var(--color-text-muted)]'}`} />
                    <span className="text-sm font-medium text-[var(--color-text-primary)]">{ds.name}</span>
                  </div>
                  <Toggle on={ds.enabled} onToggle={() => toggleDataSource(ds.name)} loading={ds.switching || switching === ds.name} />
                </div>
                <div className="flex items-center gap-4 text-xs text-[var(--color-text-muted)]">
                  <span>{ds.market}</span>
                  <span>{ds.type}</span>
                  <span className={`capitalize ${ds.status === 'connected' ? 'text-profit' : ''}`}>
                    {ds.switching ? 'switching...' : ds.status}
                  </span>
                </div>
              </div>
            ))}
            {dataSources.length === 0 && (
              <div className="text-xs text-[var(--color-text-muted)]">No data providers configured. Set API keys in .env file.</div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Shield size={14} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)]">
              Risk Controls
            </h2>
          </div>
          <div className="space-y-2">
            {riskRules.map((r, i) => (
              <div key={r.name} className="card-elevated flex items-center justify-between">
                <div>
                  <span className="text-sm text-[var(--color-text-primary)]">{r.name}</span>
                  <div className="font-data text-xs text-[var(--color-text-muted)]">{r.value}</div>
                </div>
                <Toggle on={r.enabled} onToggle={() => toggleRiskRule(i)} />
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Server size={14} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)]">
              System Status
            </h2>
          </div>
          <div className="grid grid-cols-2 gap-3">
            {systemMetrics.map(m => (
              <div key={m.label} className="card-elevated">
                <div className="text-xs text-[var(--color-text-muted)] mb-1">{m.label}</div>
                <div className={`font-data text-sm font-semibold text-${m.color}`}>
                  {m.value}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-2 mb-4">
            <Bell size={14} className="text-[var(--color-accent-glow)]" />
            <h2 className="font-heading text-sm font-semibold text-[var(--color-text-primary)]">
              Notifications
            </h2>
          </div>
          <div className="space-y-3">
            {notifications.map((n, i) => (
              <div key={n.label} className="flex items-center justify-between">
                <div>
                  <span className="text-sm text-[var(--color-text-primary)]">{n.label}</span>
                  <div className="text-xs text-[var(--color-text-muted)]">{n.desc}</div>
                </div>
                <Toggle on={n.enabled} onToggle={() => toggleNotification(i)} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
