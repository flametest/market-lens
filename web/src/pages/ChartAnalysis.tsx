import { useEffect, useRef, useState } from 'react'
import { createChart, type IChartApi, ColorType, CandlestickSeries, LineSeries } from 'lightweight-charts'
import { mockCandles, mockWatchlist } from '../mock/data'
import type { Interval } from '../types'

const intervals: Interval[] = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w']
const indicators = ['MA(5)', 'MA(20)', 'EMA(12)', 'RSI', 'MACD', 'Bollinger']

export default function ChartAnalysis() {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const [activeSymbol] = useState('AAPL')
  const [activeInterval, setActiveInterval] = useState<Interval>('1d')
  const [activeIndicators, setActiveIndicators] = useState<string[]>(['MA(5)', 'MA(20)'])

  useEffect(() => {
    if (!chartContainerRef.current) return

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: '#0f1219' },
        textColor: '#7a8299',
        fontSize: 11,
      },
      grid: {
        vertLines: { color: '#1e2433' },
        horzLines: { color: '#1e2433' },
      },
      crosshair: {
        mode: 0,
        vertLine: { color: '#2a3042', width: 1, style: 2 },
        horzLine: { color: '#2a3042', width: 1, style: 2 },
      },
      rightPriceScale: {
        borderColor: '#1e2433',
      },
      timeScale: {
        borderColor: '#1e2433',
        timeVisible: true,
      },
    })

    const series = chart.addSeries(CandlestickSeries, {
      upColor: '#10b981',
      downColor: '#ef4444',
      borderUpColor: '#10b981',
      borderDownColor: '#ef4444',
      wickUpColor: '#10b981',
      wickDownColor: '#ef4444',
    })

    series.setData(mockCandles.map(c => ({
      time: c.time as number as import('lightweight-charts').UTCTimestamp,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
    })))

    if (activeIndicators.includes('MA(5)')) {
      const ma5 = chart.addSeries(LineSeries, {
        color: '#6366f1',
        lineWidth: 1,
        priceLineVisible: false,
        lastValueVisible: false,
      })
      ma5.setData(computeMA(mockCandles, 5))
    }

    if (activeIndicators.includes('MA(20)')) {
      const ma20 = chart.addSeries(LineSeries, {
        color: '#f59e0b',
        lineWidth: 1,
        priceLineVisible: false,
        lastValueVisible: false,
      })
      ma20.setData(computeMA(mockCandles, 20))
    }

    chart.timeScale().fitContent()
    chartRef.current = chart

    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth })
      }
    }
    window.addEventListener('resize', handleResize)
    handleResize()

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
    }
  }, [activeInterval, activeIndicators])

  const toggleIndicator = (ind: string) => {
    setActiveIndicators(prev =>
      prev.includes(ind) ? prev.filter(i => i !== ind) : [...prev, ind]
    )
  }

  const currentTick = mockWatchlist.find(t => t.symbol === activeSymbol)

  return (
    <div className="space-y-4 animate-fade-in">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="font-heading text-2xl font-bold tracking-tight">
            {activeSymbol}
          </h1>
          {currentTick && (
            <div className="flex items-center gap-3">
              <span className="font-data text-xl font-semibold text-[var(--color-text-primary)]">
                ${currentTick.price.toFixed(2)}
              </span>
              <span className={`font-data text-sm font-medium ${currentTick.changePercent >= 0 ? 'text-profit' : 'text-loss'}`}>
                {currentTick.changePercent >= 0 ? '+' : ''}{currentTick.changePercent.toFixed(2)}%
              </span>
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        {intervals.map(i => (
          <button
            key={i}
            onClick={() => setActiveInterval(i)}
            className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer border-none ${
              activeInterval === i
                ? 'bg-[var(--color-accent)] text-white'
                : 'bg-[var(--color-elevated)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            {i}
          </button>
        ))}
        <div className="w-px h-5 bg-[var(--color-border)] mx-2" />
        {indicators.map(ind => (
          <button
            key={ind}
            onClick={() => toggleIndicator(ind)}
            className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer border-none ${
              activeIndicators.includes(ind)
                ? 'bg-[var(--color-accent)]/15 text-[var(--color-accent-glow)]'
                : 'bg-[var(--color-elevated)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
            }`}
          >
            {ind}
          </button>
        ))}
      </div>

      <div className="card p-0 overflow-hidden">
        <div ref={chartContainerRef} className="h-[520px]" />
      </div>
    </div>
  )
}

function computeMA(candles: { time: number; close: number }[], period: number) {
  return candles
    .map((c, i, arr) => {
      if (i < period - 1) return null
      const sum = arr.slice(i - period + 1, i + 1).reduce((s, x) => s + x.close, 0)
      return { time: c.time as number as import('lightweight-charts').UTCTimestamp, value: sum / period }
    })
    .filter(Boolean) as { time: import('lightweight-charts').UTCTimestamp; value: number }[]
}
