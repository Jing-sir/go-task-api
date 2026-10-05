import type { ApiEnvelope } from './types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1/ws'

export interface ApiRequestOptions extends RequestInit {
  token?: string | null
}

export function getWsUrl(token: string | null) {
  const url = new URL(WS_URL)
  if (token) {
    url.searchParams.set('token', token)
  }
  return url.toString()
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const url = path.startsWith('http') ? path : new URL(path, API_BASE_URL).toString()
  const headers = new Headers(options.headers)

  if (options.token) {
    headers.set('Authorization', `Bearer ${options.token}`)
  }

  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  const payload = (await response.json()) as ApiEnvelope<T>

  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || `请求失败 (${response.status})`)
  }

  return payload.data
}

