import { useState } from 'react'
import { Star, Search, ArrowUpDown } from 'lucide-react'
import { mockWatchlist } from '../mock/data'
import type { Tick } from '../types'

type SortKey = keyof Tick
type SortDir = 'asc' | 'desc'

export default function MarketOverview() {
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('symbol')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const filtered = mockWatchlist.filter(t =>
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
                    <span className="text-xs text-[var(--color-text-muted)]">{t.symbol === 'AAPL' ? 'Apple Inc.' : t.symbol === 'GOOGL' ? 'Alphabet Inc.' : t.symbol === 'MSFT' ? 'Microsoft Corp.' : t.symbol === 'AMZN' ? 'Amazon.com Inc.' : t.symbol === 'NVDA' ? 'NVIDIA Corp.' : t.symbol === 'META' ? 'Meta Platforms' : t.symbol === 'TSLA' ? 'Tesla Inc.' : 'JPMorgan Chase'}</span>
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
