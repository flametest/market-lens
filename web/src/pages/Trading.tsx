import { useEffect, useState } from 'react'
import { ArrowUpRight, ArrowDownRight, Wallet } from 'lucide-react'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Cell } from 'recharts'
import { mockAccount, mockPositions, mockOrders, dailyPnlData } from '../mock/data'
import { fetchAccount, fetchPositions, fetchOrders, submitOrder } from '../api'
import { getWSClient } from '../ws'
import type { Position, Order } from '../types'

type Tab = 'positions' | 'orders' | 'history'

export default function Trading() {
  const [activeTab, setActiveTab] = useState<Tab>('positions')
  const [account, setAccount] = useState(mockAccount)
  const [positions, setPositions] = useState<Position[]>(mockPositions)
  const [orders, setOrders] = useState<Order[]>(mockOrders)
  const [pnlData] = useState(dailyPnlData)
  const [orderSymbol, setOrderSymbol] = useState('AAPL')
  const [orderSide, setOrderSide] = useState<'BUY' | 'SELL'>('BUY')
  const [orderType, setOrderType] = useState<'MARKET' | 'LIMIT'>('MARKET')
  const [orderQty, setOrderQty] = useState('100')
  const [orderPrice, setOrderPrice] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [orderError, setOrderError] = useState('')

  const handleSubmitOrder = async () => {
    setSubmitting(true)
    setOrderError('')
    try {
      const qty = Number(orderQty)
      const price = orderType === 'LIMIT' ? Number(orderPrice) : 0
      if (!orderSymbol || qty <= 0) {
        setOrderError('Symbol and quantity are required')
        setSubmitting(false)
        return
      }
      if (orderType === 'LIMIT' && price <= 0) {
        setOrderError('Limit price is required')
        setSubmitting(false)
        return
      }
      await submitOrder({ symbol: orderSymbol, side: orderSide, type: orderType, quantity: qty, price })
      // Refresh all data
      refreshAll()
    } catch (err: any) {
      setOrderError(err?.message || 'Order failed')
    }
    setSubmitting(false)
  }

  const refreshAll = async () => {
    const [raw, posData, ordData] = await Promise.all([
      fetchAccount().catch(() => null as any),
      fetchPositions().catch(() => [] as Position[]),
      fetchOrders().catch(() => [] as Order[]),
    ])
    if (raw) {
      const cash = Number(raw.cash) || 0
      const totalValue = Number(raw.totalValue) || 0
      setAccount({ ...mockAccount, ...raw, totalValue, cash, unrealizedPnl: totalValue - 1000000 })
    }
    setPositions(posData && posData.length > 0 ? posData : [])
    setOrders(ordData && ordData.length > 0 ? ordData : [])
  }

  useEffect(() => { refreshAll() }, [])

  // Refresh when switching tabs
  useEffect(() => { refreshAll() }, [activeTab])

  useEffect(() => {
    const ws = getWSClient()
    const unsubFill = ws.on('fill', () => {
      fetchPositions().then(data => { if (data && data.length > 0) setPositions(data) }).catch(() => {})
      fetchOrders().then(data => { if (data && data.length > 0) setOrders(data) }).catch(() => {})
      fetchAccount().then((raw: any) => {
        if (raw && raw.totalValue > 0) {
          setAccount({ ...mockAccount, ...raw, totalValue: Number(raw.totalValue), cash: Number(raw.cash) })
        }
      }).catch(() => {})
    })
    return unsubFill
  }, [])

  return (
    <div className="space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-bold tracking-tight">Paper Trading</h1>
        <div className="flex items-center gap-2 text-xs text-[var(--color-text-muted)]">
          <Wallet size={14} />
          <span>Simulated Account</span>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="card">
          <div className="text-xs text-[var(--color-text-muted)] mb-1">Total Value</div>
          <div className="font-data text-xl font-semibold text-[var(--color-text-primary)]">
            ${account.totalValue.toLocaleString()}
          </div>
          <div className="font-data text-xs text-profit mt-1">
            +{(account.totalValue / 100000 * 100 - 100).toFixed(1)}%
          </div>
        </div>
        <div className="card">
          <div className="text-xs text-[var(--color-text-muted)] mb-1">Available Cash</div>
          <div className="font-data text-xl font-semibold text-[var(--color-text-primary)]">
            ${account.cash.toLocaleString()}
          </div>
        </div>
        <div className="card">
          <div className="text-xs text-[var(--color-text-muted)] mb-1">Unrealized PnL</div>
          <div className={`font-data text-xl font-semibold ${account.unrealizedPnl >= 0 ? 'text-profit' : 'text-loss'}`}>
            {account.unrealizedPnl >= 0 ? '+' : ''}${account.unrealizedPnl.toLocaleString()}
          </div>
        </div>
        <div className="card">
          <div className="text-xs text-[var(--color-text-muted)] mb-1">Daily PnL</div>
          <div className={`font-data text-xl font-semibold ${account.dailyPnl >= 0 ? 'text-profit' : 'text-loss'}`}>
            {account.dailyPnl >= 0 ? '+' : ''}${account.dailyPnl.toLocaleString()}
          </div>
        </div>
      </div>

      <div className="card">
        <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
          New Order
        </h3>
        <div className="flex items-end gap-3">
          <div className="flex-1">
            <label className="block text-xs text-[var(--color-text-muted)] mb-1">Symbol</label>
            <input type="text" value={orderSymbol} onChange={e => setOrderSymbol(e.target.value.toUpperCase())}
              placeholder="AAPL" className="input-field w-full text-xs" />
          </div>
          <div className="flex-1">
            <label className="block text-xs text-[var(--color-text-muted)] mb-1">Side</label>
            <div className="flex gap-1">
              <button onClick={() => setOrderSide('BUY')}
                className={`flex-1 py-1.5 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
                  orderSide === 'BUY' ? 'bg-[var(--color-profit)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)]'
                }`}>Buy</button>
              <button onClick={() => setOrderSide('SELL')}
                className={`flex-1 py-1.5 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
                  orderSide === 'SELL' ? 'bg-[var(--color-loss)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)]'
                }`}>Sell</button>
            </div>
          </div>
          <div className="flex-1">
            <label className="block text-xs text-[var(--color-text-muted)] mb-1">Type</label>
            <div className="flex gap-1">
              <button onClick={() => setOrderType('MARKET')}
                className={`flex-1 py-1.5 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
                  orderType === 'MARKET' ? 'bg-[var(--color-accent)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)]'
                }`}>Market</button>
              <button onClick={() => setOrderType('LIMIT')}
                className={`flex-1 py-1.5 rounded-md text-xs font-medium border-none cursor-pointer transition-colors ${
                  orderType === 'LIMIT' ? 'bg-[var(--color-accent)] text-white' : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)]'
                }`}>Limit</button>
            </div>
          </div>
          <div className="w-24">
            <label className="block text-xs text-[var(--color-text-muted)] mb-1">Quantity</label>
            <input type="number" value={orderQty} onChange={e => setOrderQty(e.target.value)}
              className="input-field w-full text-xs" min="1" />
          </div>
          {orderType === 'LIMIT' && (
            <div className="w-28">
              <label className="block text-xs text-[var(--color-text-muted)] mb-1">Price ($)</label>
              <input type="number" value={orderPrice} onChange={e => setOrderPrice(e.target.value)}
                placeholder="0.00" className="input-field w-full text-xs" min="0" step="0.01" />
            </div>
          )}
          <button onClick={handleSubmitOrder} disabled={submitting}
            className={`px-5 py-1.5 rounded-md text-xs font-semibold text-white border-none cursor-pointer transition-colors ${
              submitting ? 'opacity-50' : ''
            } ${orderSide === 'BUY' ? 'bg-[var(--color-profit)] hover:opacity-90' : 'bg-[var(--color-loss)] hover:opacity-90'}`}>
            {submitting ? 'Submitting...' : `${orderSide} ${orderSymbol}`}
          </button>
        </div>
        {orderError && <p className="text-xs text-loss mt-2">{orderError}</p>}
      </div>

      <div className="card">
        <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-4">
          Daily PnL
        </h3>
        <ResponsiveContainer width="100%" height={180}>
          <BarChart data={pnlData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#1e2433" vertical={false} />
            <XAxis
              dataKey="date"
              tick={{ fontSize: 10, fill: '#4a5068' }}
              tickLine={false}
              axisLine={false}
              tickFormatter={(v: string) => v.slice(8)}
            />
            <YAxis
              tick={{ fontSize: 10, fill: '#4a5068' }}
              tickLine={false}
              axisLine={false}
              tickFormatter={(v: number) => `$${(v / 1000).toFixed(0)}k`}
            />
            <Tooltip
              contentStyle={{ background: '#161a25', border: '1px solid #1e2433', borderRadius: '8px', fontSize: '12px' }}
              formatter={(v) => [`$${Number(v).toLocaleString()}`, 'PnL']}
            />
            <Bar dataKey="pnl" radius={[3, 3, 0, 0]}>
              {pnlData.map((entry, index) => (
                <Cell key={index} fill={entry.pnl >= 0 ? '#10b981' : '#ef4444'} fillOpacity={0.7} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="card p-0 overflow-hidden">
        <div className="flex border-b border-[var(--color-border)]">
          {(['positions', 'orders', 'history'] as Tab[]).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-5 py-3 text-xs font-medium capitalize transition-colors cursor-pointer bg-transparent border-none ${
                activeTab === tab
                  ? 'text-[var(--color-accent-glow)] border-b-2 border-[var(--color-accent)]'
                  : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {activeTab === 'positions' && (
          <table>
            <thead>
              <tr>
                <th>Symbol</th>
                <th>Side</th>
                <th>Qty</th>
                <th>Avg Price</th>
                <th>Current</th>
                <th>PnL</th>
                <th>PnL %</th>
              </tr>
            </thead>
            <tbody>
              {positions.map(p => (
                <tr key={p.symbol}>
                  <td>
                    <div className="flex items-center gap-2">
                      <span className="font-data text-sm font-semibold text-[var(--color-text-primary)]">{p.symbol}</span>
                      <span className="text-xs text-[var(--color-text-muted)]">{p.name}</span>
                    </div>
                  </td>
                  <td>
                    <span className={`badge ${p.side === 'BUY' ? 'badge-buy' : 'badge-sell'}`}>
                      {p.side === 'BUY' ? <ArrowUpRight size={10} /> : <ArrowDownRight size={10} />}
                      {p.side}
                    </span>
                  </td>
                  <td className="font-data text-sm">{p.quantity}</td>
                  <td className="font-data text-sm">${p.avgPrice.toFixed(2)}</td>
                  <td className="font-data text-sm text-[var(--color-text-primary)]">${p.currentPrice.toFixed(2)}</td>
                  <td className={`font-data text-sm font-medium ${p.unrealizedPnl >= 0 ? 'text-profit' : 'text-loss'}`}>
                    {p.unrealizedPnl >= 0 ? '+' : ''}${p.unrealizedPnl.toFixed(2)}
                  </td>
                  <td className={`font-data text-sm font-medium ${p.unrealizedPnlPercent >= 0 ? 'text-profit' : 'text-loss'}`}>
                    {p.unrealizedPnlPercent >= 0 ? '+' : ''}{p.unrealizedPnlPercent.toFixed(2)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {activeTab === 'orders' && (
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Symbol</th>
                <th>Side</th>
                <th>Type</th>
                <th>Qty</th>
                <th>Price</th>
                <th>Filled</th>
                <th>Status</th>
                <th>Strategy</th>
              </tr>
            </thead>
            <tbody>
              {orders.map(o => (
                <tr key={o.id}>
                  <td className="text-xs">{new Date(o.createdAt).toLocaleString()}</td>
                  <td className="font-data text-sm font-medium text-[var(--color-text-primary)]">{o.symbol}</td>
                  <td>
                    <span className={`badge ${o.side === 'BUY' ? 'badge-buy' : 'badge-sell'}`}>{o.side}</span>
                  </td>
                  <td className="text-xs">{o.type}</td>
                  <td className="font-data text-sm">{o.quantity}</td>
                  <td className="font-data text-sm">${o.price.toFixed(2)}</td>
                  <td className="font-data text-sm">{o.filledQty}/{o.quantity}</td>
                  <td>
                    <span className={`badge ${o.status === 'FILLED' ? 'badge-buy' : o.status === 'NEW' ? 'badge-running' : 'badge-neutral'}`}>
                      {o.status}
                    </span>
                  </td>
                  <td className="text-xs">{o.strategy}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
