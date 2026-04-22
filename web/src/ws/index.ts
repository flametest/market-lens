type MessageHandler = (data: any) => void

class WSClient {
  private url: string
  private ws: WebSocket | null = null
  private handlers: Map<string, Set<MessageHandler>> = new Map()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private backoff = 1000
  private maxBackoff = 30000
  private stopped = false

  constructor(url: string) {
    this.url = url
  }

  connect() {
    this.stopped = false
    this.doConnect()
  }

  private doConnect() {
    if (this.stopped) return

    try {
      this.ws = new WebSocket(this.url)
    } catch {
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.backoff = 1000
    }

    this.ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type && msg.data !== undefined) {
          this.dispatch(msg.type, msg.data)
        }
      } catch {}
    }

    this.ws.onclose = () => {
      this.ws = null
      this.scheduleReconnect()
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  private scheduleReconnect() {
    if (this.stopped) return
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.reconnectTimer = setTimeout(() => {
      this.doConnect()
    }, this.backoff)
    this.backoff = Math.min(this.backoff * 2, this.maxBackoff)
  }

  on(type: string, handler: MessageHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set())
    }
    this.handlers.get(type)!.add(handler)
    return () => this.off(type, handler)
  }

  off(type: string, handler: MessageHandler) {
    this.handlers.get(type)?.delete(handler)
  }

  private dispatch(type: string, data: any) {
    // Wildcard handlers
    this.handlers.get('*')?.forEach(h => h({ type, data }))
    // Type-specific handlers
    this.handlers.get(type)?.forEach(h => h(data))
  }

  close() {
    this.stopped = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }
}

let client: WSClient | null = null

export function getWSClient(): WSClient {
  if (!client) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    const port = '8081'
    const url = `${proto}//${host}:${port}/ws`
    client = new WSClient(url)
    client.connect()
  }
  return client
}

export { WSClient }
