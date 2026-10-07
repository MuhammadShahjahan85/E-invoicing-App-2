// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Thin fetch wrapper for the /api/v1 REST API.
import type { Issue } from './types'

let csrfToken = ''

export function setCsrf(token: string) {
  csrfToken = token
}

export class ApiError extends Error {
  status: number
  issues?: Issue[]
  data?: unknown
  constructor(status: number, message: string, issues?: Issue[], data?: unknown) {
    super(message)
    this.status = status
    this.issues = issues
    this.data = data
  }
}

type Listener = () => void
const unauthorizedListeners: Listener[] = []
export function onUnauthorized(fn: Listener) {
  unauthorizedListeners.push(fn)
}

async function request<T>(method: string, path: string, body?: unknown, raw?: { contentType: string; data: BodyInit }): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (method !== 'GET' && csrfToken) headers['X-CSRF-Token'] = csrfToken
  let payload: BodyInit | undefined
  if (raw) {
    if (raw.contentType) headers['Content-Type'] = raw.contentType
    payload = raw.data
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch('/api/v1' + path, { method, headers, body: payload, credentials: 'same-origin' })
  const text = await res.text()
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  if (!res.ok) {
    const d = (data ?? {}) as { error?: string; issues?: Issue[] }
    if (res.status === 401 && !path.startsWith('/auth/login')) unauthorizedListeners.forEach((f) => f())
    throw new ApiError(res.status, d.error || res.statusText || 'Request failed', d.issues, data)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body ?? {}),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body ?? {}),
  del: <T>(path: string) => request<T>('DELETE', path),
  upload: <T>(path: string, method: 'PUT' | 'POST', data: BodyInit, contentType: string) => request<T>(method, path, undefined, { contentType, data }),
}

export function qs(params: Record<string, string | number | boolean | undefined | null>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    p.set(k, String(v))
  }
  const s = p.toString()
  return s ? '?' + s : ''
}

export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof Error) return e.message
  return String(e)
}
