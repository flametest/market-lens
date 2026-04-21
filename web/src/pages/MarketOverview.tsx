import { useState, useEffect, useCallback, useRef } from 'react'
import { Star, Search, ArrowUpDown, RefreshCw, Clock } from 'lucide-react'
import { mockWatchlist } from '../mock/data'
import { fetchSymbols, fetchQuote } from '../api'
import type { Tick } from '../types'

type SortKey = keyof Tick
type SortDir = 'asc' | 'desc'

const REFRESH_OPTIONS = [
  { label: 'Off', ms: 0 },
  { label: '1s', ms: 1000 },
  { label: '5s', ms: 5000 },
  { label: '10s', ms: 10000 },
  { label: '1m', ms: 60000 },
  { label: '5m', ms: 300000 },
  { label: '30m', ms: 1800000 },
  { label: '1h', ms: 3600000 },
] as const

const symbolNames: Record<string, string> = {
  AAPL: 'Apple Inc.', GOOGL: 'Alphabet Inc.', MSFT: 'Microsoft Corp.',
  AMZN: 'Amazon.com Inc.', NVDA: 'NVIDIA Corp.', META: 'Meta Platforms',
  TSLA: 'Tesla Inc.', JPM: 'JPMorgan Chase',
}

export default function MarketOverview() {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('symbol')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [watchlist, setWatchlist] = useState<Tick[]>(mockWatchlist)
  const [loading, setLoading] = useState(true)
  const [refreshMs, setRefreshMs] = useState(10000)
  const [showRefreshMenu, setShowRefreshMenu] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const loadData = useCallback(async (q?: string) => {
    setLoading(true)
    try {
      const raw: any[] = await fetchSymbols(q || undefined)
      if (!raw || raw.length === 0) { setLoading(false); return }

      const ticks = await Promise.all(
        raw.map(async (s: any) => {
          const tick = await fetchQuote(s.code)
          if (tick && tick.price > 0) return tick
          return {
            symbol: s.code, price: 0, volume: 0, change: 0, changePercent: 0,
            high: 0, low: 0, open: 0, timestamp: Date.now(),
          }
        })
      )
      const valid = ticks.filter(t => t.price > 0)
      if (valid.length > 0) setWatchlist(valid)
    } catch { /* keep mock */ }
    setLoading(false)
  }, [])

  useEffect(() => { loadData() }, [loadData])

  useEffect(() => {
    const timer = setTimeout(() => loadData(search), 500)
    return () => clearTimeout(timer)
  }, [search, loadData])

  useEffect(() => {
    if (refreshMs === 0) return
    const id = setInterval(() => loadData(search), refreshMs)
    return () => clearInterval(id)
  }, [refreshMs, loadData, search])

  useEffect(() => {
    if (!showRefreshMenu) return
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowRefreshMenu(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [showRefreshMenu])

  const activeRefreshLabel = REFRESH_OPTIONS.find(o => o.ms === refreshMs)?.label || 'Off'

  const filtered = watchlist.filter(t =>
    t.symbol.toLowerCase().includes(search.toLowerCase())
  )

  const sorted = [...filtered].sort((a, b) => {
    const aVal = a[sortKey]
    const bVal = b[sortKey]
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return sortDir === 'asc' ? aVal - bVal : bVal - aVal
    }
    return sortDir === 'asc'
      ? String(aVal).localeCompare(String(bVal))
      : String(bVal).localeCompare(String(aVal))
  })

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    else { setSortKey(key); setSortDir('desc') }
  }

  const SortHeader = ({ label, field }: { label: string; field: SortKey }) => (
    <th
      className="cursor-pointer select-none hover:text-[var(--color-text-secondary)] transition-colors"
      onClick={() => toggleSort(field)}
    >
      <span className="flex items-center gap-1">
        {label}
        <ArrowUpDown size={10} className={sortKey === field ? 'text-[var(--color-accent)]' : ''} />
      </span>
    </th>
  )

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Market Overview</h1>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)]" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search symbol..."
              className="input-field pl-9 w-56 text-xs"
            />
          </div>
          <div className="relative" ref={menuRef}>
            <button
              onClick={() => setShowRefreshMenu(v => !v)}
              className={`btn-ghost flex items-center gap-1.5 ${refreshMs > 0 ? 'text-[var(--color-accent)]' : ''}`}
            >
              <Clock size={12} />
              <span className="text-xs">{refreshMs > 0 ? `Auto ${activeRefreshLabel}` : 'Auto Refresh'}</span>
              {refreshMs > 0 && <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-accent)] animate-pulse-soft" />}
            </button>
            {showRefreshMenu && (
              <div className="absolute right-0 top-full mt-1 card-elevated py-1 min-w-[100px] z-50">
                {REFRESH_OPTIONS.map(opt => (
                  <button
                    key={opt.label}
                    onClick={() => { setRefreshMs(opt.ms); setShowRefreshMenu(false) }}
                    className={`w-full text-left px-3 py-1.5 text-xs hover:bg-[var(--color-bg-hover)] transition-colors border-none bg-transparent cursor-pointer ${
                      refreshMs === opt.ms ? 'text-[var(--color-accent)] font-semibold' : 'text-[var(--color-text-secondary)]'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
          <button onClick={() => loadData(search)} className="btn-ghost flex items-center gap-1.5" disabled={loading}>
            <RefreshCw size={12} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      <div className="card p-0 overflow-hidden">
        <table>
          <thead>
            <tr>
              <th className="w-8"></th>
              <SortHeader label="Symbol" field="symbol" />
              <SortHeader label="Price" field="price" />
              <SortHeader label="Change" field="changePercent" />
              <SortHeader label="High" field="high" />
              <SortHeader label="Low" field="low" />
              <SortHeader label="Volume" field="volume" />
            </tr>
          </thead>
          <tbody>
            {sorted.map((t) => (
              <tr key={t.symbol} className="group cursor-pointer">
                <td className="w-8">
                  <Star size={14} className="text-[var(--color-text-muted)] hover:text-[var(--color-warn)] transition-colors cursor-pointer" />
                </td>
                <td>
                  <div className="flex items-center gap-2">
                    <span className="font-data text-sm font-semibold text-[var(--color-text-primary)]">{t.symbol}</span>
                    <span className="text-xs text-[var(--color-text-muted)]">{symbolNames[t.symbol] || ''}</span>
                  </div>
                </td>
                <td className="font-data text-sm font-medium text-[var(--color-text-primary)]">
                  ${t.price.toFixed(2)}
                </td>
                <td>
                  <span className={`font-data text-sm font-medium ${t.changePercent >= 0 ? 'text-profit' : 'text-loss'}`}>
                    {t.changePercent >= 0 ? '+' : ''}{t.changePercent.toFixed(2)}%
                  </span>
                </td>
                <td className="font-data text-xs">${t.high.toFixed(2)}</td>
                <td className="font-data text-xs">${t.low.toFixed(2)}</td>
                <td className="font-data text-xs">{(t.volume / 1e6).toFixed(1)}M</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
