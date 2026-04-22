import { useEffect, useRef, useCallback } from 'react'
import { getWSClient } from '../ws'

type TickData = {
  symbol: string
  price: number
  volume: number
  change: number
  changePercent: number
  high: number
  low: number
  open: number
  timestamp: number
  source: string
}

type SignalData = {
  id: string
  symbol: string
  source: string
  type: string
  strength: number
  price: number
  timestamp: number
}

export function useTick(symbol: string): { latest: TickData | null } {
  const ref = useRef<TickData | null>(null)

  useEffect(() => {
    const ws = getWSClient()
    const handler = (data: TickData) => {
      if (data.symbol === symbol) {
        ref.current = data
      }
    }
    const unsub = ws.on('tick', handler)
    return unsub
  }, [symbol])

  return { latest: ref.current }
}

export function useTickStream(onTick: (data: TickData) => void) {
  useEffect(() => {
    const ws = getWSClient()
    const unsub = ws.on('tick', onTick)
    return unsub
  }, [onTick])
}

export function useSignalStream(onSignal: (data: SignalData) => void) {
  useEffect(() => {
    const ws = getWSClient()
    const unsub = ws.on('signal', onSignal)
    return unsub
  }, [onSignal])
}

export function useWSMessage(type: string, handler: (data: any) => void) {
  useEffect(() => {
    const ws = getWSClient()
    const unsub = ws.on(type, handler)
    return unsub
  }, [type, handler])
}
