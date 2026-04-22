import { useState, useEffect, useCallback, useRef } from 'react'
import { Star, Search, ArrowUpDown, RefreshCw, Clock, Plus, X } from 'lucide-react'
import { mockWatchlist } from '../mock/data'
import { fetchSymbols, fetchQuote } from '../api'
import { getWSClient } from '../ws'
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

const STORAGE_KEY = 'market-lens-groups'

type GroupMap = Record<string, string[]> // group name -> symbol list

function loadGroups(): GroupMap {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch { return {} }
}

function saveGroups(groups: GroupMap) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(groups))
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

  // Groups
  const [groups, setGroups] = useState<GroupMap>(loadGroups)
  const [activeGroup, setActiveGroup] = useState<string>('all')
  const [newGroupName, setNewGroupName] = useState('')
  const [showGroupInput, setShowGroupInput] = useState(false)
  const [managingSymbol, setManagingSymbol] = useState<string | null>(null)

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
    const ws = getWSClient()
    const unsub = ws.on('tick', (data: any) => {
      setWatchlist(prev => prev.map(t =>
        t.symbol === data.symbol
          ? {
              ...t,
              price: Number(data.price) || t.price,
              volume: Number(data.volume) || t.volume,
              high: Number(data.high) || t.high,
              low: Number(data.low) || t.low,
            }
          : t
      ))
    })
    return unsub
  }, [])

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

  const createGroup = () => {
    const name = newGroupName.trim()
    if (!name || groups[name]) return
    const updated = { ...groups, [name]: [] }
    setGroups(updated)
    saveGroups(updated)
    setNewGroupName('')
    setShowGroupInput(false)
    setActiveGroup(name)
  }

  const deleteGroup = (name: string) => {
    const updated = { ...groups }
    delete updated[name]
    setGroups(updated)
    saveGroups(updated)
    if (activeGroup === name) setActiveGroup('all')
  }

  const toggleSymbolGroup = (symbol: string, groupName: string) => {
    const symbols = groups[groupName] || []
    const updated = {
      ...groups,
      [groupName]: symbols.includes(symbol)
        ? symbols.filter(s => s !== symbol)
        : [...symbols, symbol],
    }
    setGroups(updated)
    saveGroups(updated)
  }

  const toggleStar = (symbol: string) => {
    if (Object.keys(groups).length === 0) {
      // Create a default "Favorites" group
      const favs = { Favorites: [symbol] }
      setGroups(favs)
      saveGroups(favs)
      return
    }
    setManagingSymbol(managingSymbol === symbol ? null : symbol)
  }

  const isStarred = (symbol: string) => {
    return Object.values(groups).some(symbols => symbols.includes(symbol))
  }

  const filtered = watchlist.filter(t => {
    const matchesSearch = t.symbol.toLowerCase().includes(search.toLowerCase())
    if (activeGroup === 'all') return matchesSearch
    const groupSymbols = groups[activeGroup] || []
    return matchesSearch && groupSymbols.includes(t.symbol)
  })

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

      {/* Group tabs */}
      <div className="flex items-center gap-2 flex-wrap">
        <button
          onClick={() => setActiveGroup('all')}
          className={`px-3 py-1 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
            activeGroup === 'all' ? 'bg-[var(--color-accent)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
          }`}
        >All</button>
        {Object.keys(groups).map(name => (
          <div key={name} className="flex items-center gap-1">
            <button
              onClick={() => setActiveGroup(name)}
              className={`px-3 py-1 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
                activeGroup === name ? 'bg-[var(--color-accent)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
              }`}
            >{name} ({(groups[name] || []).length})</button>
            <button onClick={() => deleteGroup(name)}
              className="p-0.5 text-[var(--color-text-muted)] hover:text-[var(--color-loss)] border-none bg-transparent cursor-pointer">
              <X size={10} />
            </button>
          </div>
        ))}
        {showGroupInput ? (
          <div className="flex items-center gap-1">
            <input type="text" value={newGroupName} onChange={e => setNewGroupName(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && createGroup()}
              placeholder="Group name" className="input-field text-xs w-28 py-1" autoFocus />
            <button onClick={createGroup} className="btn-ghost text-xs py-1">OK</button>
            <button onClick={() => { setShowGroupInput(false); setNewGroupName('') }}
              className="p-0.5 text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] border-none bg-transparent cursor-pointer">
              <X size={12} />
            </button>
          </div>
        ) : (
          <button onClick={() => setShowGroupInput(true)}
            className="px-2 py-1 rounded-md text-xs border border-dashed border-[var(--color-border-light)] text-[var(--color-text-muted)] bg-transparent cursor-pointer hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] transition-colors flex items-center gap-1">
            <Plus size={10} /> New Group
          </button>
        )}
      </div>

      <div className="card p-0 overflow-hidden relative">
        {managingSymbol && (
          <div className="absolute top-0 right-0 z-10 card-elevated p-3 m-2 min-w-[180px]">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-[var(--color-text-primary)]">Add "{managingSymbol}" to:</span>
              <button onClick={() => setManagingSymbol(null)} className="text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] border-none bg-transparent cursor-pointer p-0">
                <X size={12} />
              </button>
            </div>
            {Object.keys(groups).map(name => (
              <button key={name}
                onClick={() => { toggleSymbolGroup(managingSymbol, name); setManagingSymbol(null) }}
                className={`w-full text-left px-2 py-1.5 text-xs rounded-md border-none cursor-pointer transition-colors ${
                  (groups[name] || []).includes(managingSymbol)
                    ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]'
                    : 'bg-transparent text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)]'
                }`}
              >
                {(groups[name] || []).includes(managingSymbol) ? '✓ ' : ''}{name}
              </button>
            ))}
            {Object.keys(groups).length === 0 && (
              <p className="text-xs text-[var(--color-text-muted)]">Create a group first</p>
            )}
          </div>
        )}
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
                  <Star size={14}
                    onClick={() => toggleStar(t.symbol)}
                    className={`${isStarred(t.symbol) ? 'text-[var(--color-warn)] fill-[var(--color-warn)]' : 'text-[var(--color-text-muted)]'} hover:text-[var(--color-warn)] transition-colors cursor-pointer`}
                  />
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
