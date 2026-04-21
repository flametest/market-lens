import { useEffect, useRef, useState } from 'react'
import { createChart, type IChartApi, ColorType, CandlestickSeries, LineSeries, HistogramSeries } from 'lightweight-charts'
import { mockCandles, mockWatchlist } from '../mock/data'
import { fetchCandles, fetchQuote } from '../api'
import type { Interval, Candle, Tick } from '../types'

const intervals: Interval[] = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w']
const indicators = ['MA(5)', 'MA(20)', 'EMA(12)', 'RSI', 'MACD', 'Bollinger']

type UTCTimestamp = import('lightweight-charts').UTCTimestamp

export default function ChartAnalysis() {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const [activeSymbol] = useState('AAPL')
  const [activeInterval, setActiveInterval] = useState<Interval>('1d')
  const [activeIndicators, setActiveIndicators] = useState<string[]>(['MA(5)', 'MA(20)'])
  const [candles, setCandles] = useState<Candle[]>(mockCandles)
  const [currentTick, setCurrentTick] = useState<Tick | undefined>(mockWatchlist.find(t => t.symbol === 'AAPL'))

  useEffect(() => {
    fetchCandles(activeSymbol, activeInterval).then(data => {
      if (data && data.length > 0) {
        setCandles(data)
      } else {
        setCandles(generateMockCandles(activeInterval))
      }
    }).catch(() => {
      setCandles(generateMockCandles(activeInterval))
    })
  }, [activeSymbol, activeInterval])

  useEffect(() => {
    fetchQuote(activeSymbol).then(tick => {
      if (tick) setCurrentTick(tick)
    }).catch(() => {})
  }, [activeSymbol])

  useEffect(() => {
    if (!chartContainerRef.current || candles.length === 0) return

    const showRSI = activeIndicators.includes('RSI')
    const showMACD = activeIndicators.includes('MACD')

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: '#0f1219' },
        textColor: '#7a8299',
        fontSize: 11,
        panes: {
          separatorColor: '#1e2433',
          separatorHoverColor: '#2a3042',
          enableResize: true,
        },
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
      rightPriceScale: { borderColor: '#1e2433' },
      timeScale: { borderColor: '#1e2433', timeVisible: true },
    })

    // Pane 0: Main candlestick chart
    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#10b981',
      downColor: '#ef4444',
      borderUpColor: '#10b981',
      borderDownColor: '#ef4444',
      wickUpColor: '#10b981',
      wickDownColor: '#ef4444',
    })
    candleSeries.setData(candles.map(c => ({
      time: c.time as number as UTCTimestamp,
      open: c.open, high: c.high, low: c.low, close: c.close,
    })))

    if (activeIndicators.includes('MA(5)')) {
      const ma5 = chart.addSeries(LineSeries, {
        color: '#6366f1', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      })
      ma5.setData(computeMA(candles, 5))
    }

    if (activeIndicators.includes('MA(20)')) {
      const ma20 = chart.addSeries(LineSeries, {
        color: '#f59e0b', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      })
      ma20.setData(computeMA(candles, 20))
    }

    if (activeIndicators.includes('EMA(12)')) {
      const ema12 = chart.addSeries(LineSeries, {
        color: '#ec4899', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      })
      ema12.setData(computeEMA(candles, 12))
    }

    if (activeIndicators.includes('Bollinger')) {
      const closes = candles.map(c => c.close)
      const bb = computeBollinger(closes, 20, 2)
      const upperData: { time: UTCTimestamp; value: number }[] = []
      const lowerData: { time: UTCTimestamp; value: number }[] = []
      const middleData: { time: UTCTimestamp; value: number }[] = []
      for (let i = 0; i < bb.upper.length; i++) {
        const t = candles[i].time as number as UTCTimestamp
        if (bb.upper[i] !== null) upperData.push({ time: t, value: bb.upper[i]! })
        if (bb.lower[i] !== null) lowerData.push({ time: t, value: bb.lower[i]! })
        if (bb.middle[i] !== null) middleData.push({ time: t, value: bb.middle[i]! })
      }
      const upperLine = chart.addSeries(LineSeries, {
        color: '#8b5cf6', lineWidth: 1, priceLineVisible: false, lastValueVisible: false, lineStyle: 2,
      })
      const lowerLine = chart.addSeries(LineSeries, {
        color: '#8b5cf6', lineWidth: 1, priceLineVisible: false, lastValueVisible: false, lineStyle: 2,
      })
      const middleLine = chart.addSeries(LineSeries, {
        color: '#8b5cf680', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      })
      upperLine.setData(upperData)
      lowerLine.setData(lowerData)
      middleLine.setData(middleData)
    }

    // Pane 1: RSI
    if (showRSI) {
      const rsiData = computeRSI(candles, 14)
      const rsiValid = rsiData
        .map((v, i) => ({ time: candles[i].time as number as UTCTimestamp, value: v }))
        .filter(d => d.value !== null) as { time: UTCTimestamp; value: number }[]

      const rsiLine = chart.addSeries(LineSeries, {
        color: '#a78bfa', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      }, 1)
      rsiLine.setData(rsiValid)

      // Reference lines at 70/30
      const rsiRefData = candles.map(c => ({ time: c.time as number as UTCTimestamp, value: 70 as number }))
      const ob = chart.addSeries(LineSeries, {
        color: '#ef444480', lineWidth: 1, lineStyle: 2, priceLineVisible: false, lastValueVisible: false, crosshairMarkerVisible: false,
      }, 1)
      ob.setData(rsiRefData)

      const osData = candles.map(c => ({ time: c.time as number as UTCTimestamp, value: 30 }))
      const os = chart.addSeries(LineSeries, {
        color: '#10b98180', lineWidth: 1, lineStyle: 2, priceLineVisible: false, lastValueVisible: false, crosshairMarkerVisible: false,
      }, 1)
      os.setData(osData)

      // Set RSI pane height
      const panes = chart.panes()
      if (panes.length > 1) panes[1].setHeight(140)
    }

    // Pane 2: MACD
    if (showMACD) {
      const macdData = computeMACD(candles, 12, 26, 9)
      const macdSeriesData: { time: UTCTimestamp; value: number }[] = []
      const signalSeriesData: { time: UTCTimestamp; value: number }[] = []
      const histSeriesData: { time: UTCTimestamp; value: number; color: string }[] = []

      for (let i = 0; i < macdData.macd.length; i++) {
        const t = candles[i].time as number as UTCTimestamp
        if (macdData.macd[i] !== null) macdSeriesData.push({ time: t, value: macdData.macd[i]! })
        if (macdData.signal[i] !== null) signalSeriesData.push({ time: t, value: macdData.signal[i]! })
        if (macdData.histogram[i] !== null) {
          histSeriesData.push({
            time: t,
            value: macdData.histogram[i]!,
            color: macdData.histogram[i]! >= 0 ? '#10b98180' : '#ef444480',
          })
        }
      }

      const macdPaneIndex = showRSI ? 2 : 1

      const macdLine = chart.addSeries(LineSeries, {
        color: '#6366f1', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      }, macdPaneIndex)
      macdLine.setData(macdSeriesData)

      const signalLine = chart.addSeries(LineSeries, {
        color: '#f59e0b', lineWidth: 1, priceLineVisible: false, lastValueVisible: false,
      }, macdPaneIndex)
      signalLine.setData(signalSeriesData)

      const histogram = chart.addSeries(HistogramSeries, {
        priceLineVisible: false, lastValueVisible: false,
      }, macdPaneIndex)
      histogram.setData(histSeriesData)

      // Set MACD pane height
      const panes = chart.panes()
      if (panes.length > macdPaneIndex) panes[macdPaneIndex].setHeight(140)
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
      chartRef.current = null
    }
  }, [activeInterval, activeIndicators, candles])

  const toggleIndicator = (ind: string) => {
    setActiveIndicators(prev =>
      prev.includes(ind) ? prev.filter(i => i !== ind) : [...prev, ind]
    )
  }

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
        <div ref={chartContainerRef} style={{ height: activeIndicators.includes('RSI') || activeIndicators.includes('MACD') ? 700 : 520 }} />
      </div>
    </div>
  )
}

// --- Indicator computations ---

function computeMA(candles: { time: number; close: number }[], period: number) {
  return candles
    .map((c, i, arr) => {
      if (i < period - 1) return null
      const sum = arr.slice(i - period + 1, i + 1).reduce((s, x) => s + x.close, 0)
      return { time: c.time as number as UTCTimestamp, value: sum / period }
    })
    .filter(Boolean) as { time: UTCTimestamp; value: number }[]
}

function computeEMA(candles: { time: number; close: number }[], period: number) {
  const k = 2 / (period + 1)
  const result: { time: UTCTimestamp; value: number }[] = []
  let ema: number | null = null

  for (let i = 0; i < candles.length; i++) {
    if (i < period - 1) continue
    if (ema === null) {
      let sum = 0
      for (let j = i - period + 1; j <= i; j++) sum += candles[j].close
      ema = sum / period
    } else {
      ema = candles[i].close * k + ema * (1 - k)
    }
    result.push({ time: candles[i].time as number as UTCTimestamp, value: ema })
  }
  return result
}

function computeRSI(candles: { close: number }[], period: number): (number | null)[] {
  const result: (number | null)[] = new Array(candles.length).fill(null)
  if (candles.length < period + 1) return result

  let avgGain = 0
  let avgLoss = 0

  for (let i = 1; i <= period; i++) {
    const change = candles[i].close - candles[i - 1].close
    if (change > 0) avgGain += change
    else avgLoss += Math.abs(change)
  }
  avgGain /= period
  avgLoss /= period

  result[period] = avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss)

  for (let i = period + 1; i < candles.length; i++) {
    const change = candles[i].close - candles[i - 1].close
    const gain = change > 0 ? change : 0
    const loss = change < 0 ? Math.abs(change) : 0
    avgGain = (avgGain * (period - 1) + gain) / period
    avgLoss = (avgLoss * (period - 1) + loss) / period
    result[i] = avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss)
  }
  return result
}

function computeBollinger(closes: number[], period: number, mult: number) {
  const upper: (number | null)[] = new Array(closes.length).fill(null)
  const middle: (number | null)[] = new Array(closes.length).fill(null)
  const lower: (number | null)[] = new Array(closes.length).fill(null)

  for (let i = period - 1; i < closes.length; i++) {
    let sum = 0
    for (let j = i - period + 1; j <= i; j++) sum += closes[j]
    const avg = sum / period
    let sqSum = 0
    for (let j = i - period + 1; j <= i; j++) sqSum += (closes[j] - avg) ** 2
    const std = Math.sqrt(sqSum / period)
    middle[i] = avg
    upper[i] = avg + mult * std
    lower[i] = avg - mult * std
  }
  return { upper, middle, lower }
}

function computeMACD(candles: { close: number }[], fast: number, slow: number, signal: number) {
  const macd: (number | null)[] = new Array(candles.length).fill(null)
  const signalLine: (number | null)[] = new Array(candles.length).fill(null)
  const histogram: (number | null)[] = new Array(candles.length).fill(null)

  const emaFast = computeEMAValues(candles.map(c => c.close), fast)
  const emaSlow = computeEMAValues(candles.map(c => c.close), slow)

  const macdValues: (number | null)[] = []
  for (let i = 0; i < candles.length; i++) {
    if (emaFast[i] !== null && emaSlow[i] !== null) {
      macdValues.push(emaFast[i]! - emaSlow[i]!)
    } else {
      macdValues.push(null)
    }
  }

  const sigEma = computeEMAValues(macdValues, signal)

  for (let i = 0; i < candles.length; i++) {
    if (macdValues[i] !== null) macd[i] = macdValues[i]
    if (sigEma[i] !== null) {
      signalLine[i] = sigEma[i]
      if (macdValues[i] !== null) histogram[i] = macdValues[i]! - sigEma[i]!
    }
  }

  return { macd, signal: signalLine, histogram }
}

function computeEMAValues(values: (number | null)[], period: number): (number | null)[] {
  const result: (number | null)[] = new Array(values.length).fill(null)
  const k = 2 / (period + 1)
  let ema: number | null = null
  let firstValid = -1

  for (let i = period - 1; i < values.length; i++) {
    let valid = true
    let sum = 0
    for (let j = i - period + 1; j <= i; j++) {
      if (values[j] === null) { valid = false; break }
      sum += values[j]!
    }
    if (!valid) continue
    if (ema === null) {
      ema = sum / period
      result[i] = ema
      firstValid = i
    }
  }

  if (firstValid < 0) return result

  for (let i = firstValid + 1; i < values.length; i++) {
    if (values[i] === null || ema === null) continue
    ema = values[i]! * k + ema * (1 - k)
    result[i] = ema
  }
  return result
}

const intervalStepMs: Record<string, number> = {
  '1m': 60_000, '5m': 300_000, '15m': 900_000, '30m': 1_800_000,
  '1h': 3_600_000, '4h': 14_400_000, '1d': 86_400_000, '1w': 604_800_000, '1M': 2_592_000_000,
}

function generateMockCandles(interval: string): Candle[] {
  const step = intervalStepMs[interval] || 86_400_000
  const count = 120
  const now = Date.now()
  const result: Candle[] = []
  let price = 180 + Math.random() * 20

  for (let i = count; i >= 0; i--) {
    const open = price
    const volatility = interval === '1m' || interval === '5m' ? 0.5 : interval === '1w' || interval === '1M' ? 8 : 3
    const change = (Math.random() - 0.48) * volatility
    const close = open + change
    const high = Math.max(open, close) + Math.random() * volatility * 0.6
    const low = Math.min(open, close) - Math.random() * volatility * 0.6
    const volume = Math.floor(Math.random() * 50_000_000) + 10_000_000
    result.push({
      time: Math.floor((now - i * step) / 1000),
      open: Math.round(open * 100) / 100,
      high: Math.round(high * 100) / 100,
      low: Math.round(low * 100) / 100,
      close: Math.round(close * 100) / 100,
      volume,
    })
    price = close
  }
  return result
}
