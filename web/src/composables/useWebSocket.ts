import { onBeforeUnmount, shallowRef } from 'vue'

export type SocketStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'reconnecting'

export interface UseWebSocketOptions {
  url: string
  autoReconnect?: boolean
  onMessage?: (event: MessageEvent<string>) => void
  onOpen?: (event: Event) => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
}

export function useWebSocket(options: UseWebSocketOptions) {
  const socket = shallowRef<WebSocket | null>(null)
  const status = shallowRef<SocketStatus>('idle')

  let reconnectTimer: number | null = null
  let reconnectCount = 0
  let manualClose = false

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function close() {
    manualClose = true
    clearReconnectTimer()

    if (socket.value && socket.value.readyState !== WebSocket.CLOSED) {
      socket.value.close()
    }

    socket.value = null
    status.value = 'closed'
  }

  function scheduleReconnect() {
    if (options.autoReconnect === false || manualClose) {
      return
    }

    clearReconnectTimer()
    status.value = 'reconnecting'

    const delay = Math.min(1000 * 2 ** reconnectCount, 5000)
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      reconnectCount += 1
      open()
    }, delay)
  }

  function open() {
    manualClose = false
    clearReconnectTimer()

    if (socket.value && socket.value.readyState === WebSocket.OPEN) {
      return
    }

    status.value = 'connecting'
    const nextSocket = new WebSocket(options.url)
    socket.value = nextSocket

    nextSocket.onopen = (event) => {
      reconnectCount = 0
      status.value = 'open'
      options.onOpen?.(event)
    }

    nextSocket.onmessage = (event) => {
      options.onMessage?.(event as MessageEvent<string>)
    }

    nextSocket.onerror = (event) => {
      options.onError?.(event)
    }

    nextSocket.onclose = (event) => {
      socket.value = null
      status.value = 'closed'
      options.onClose?.(event)

      if (!manualClose) {
        scheduleReconnect()
      }
    }
  }

  onBeforeUnmount(close)

  return {
    close,
    open,
    socket,
    status,
  }
}
