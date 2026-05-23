// Thin fetch wrapper for the Tansiq API.
//
// Responsibilities:
//  - Inject the Bearer access token from the auth store.
//  - Send + receive cookies (refresh token cookie).
//  - On a 401 with code "token_expired", call /auth/refresh once and retry.
//  - Surface a typed ApiError that carries the API error envelope.
//
// Anything that needs richer behaviour (caching, retries, optimistic
// updates) goes on top of this through TanStack Query.

import { useAuthStore, type AuthUser } from './auth-store'

const RAW_BASE = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080').replace(
  /\/+$/,
  '',
)

export const apiBaseUrl = RAW_BASE

export interface ApiErrorBody {
  error?: {
    code?: string
    message?: string
    fields?: Record<string, string>
    request_id?: string
  }
}

export class ApiError extends Error {
  status: number
  code: string
  fields?: Record<string, string>
  requestId?: string

  constructor(status: number, body: ApiErrorBody | null) {
    const e = body?.error ?? {}
    super(e.message ?? `API ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.code = e.code ?? 'unknown_error'
    this.fields = e.fields
    this.requestId = e.request_id
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  /** JSON-serialisable body. Set body: undefined for GET. */
  body?: unknown
  /** Skip auto-refresh on 401 (used by the refresh call itself). */
  noRefresh?: boolean
}

let refreshInFlight: Promise<boolean> | null = null

async function refreshOnce(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight
  refreshInFlight = (async () => {
    try {
      const res = await fetch(`${RAW_BASE}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!res.ok) {
        useAuthStore.getState().clear()
        return false
      }
      const data = (await res.json()) as { access_token: string; user: AuthUser }
      useAuthStore.getState().setSession(data.access_token, data.user)
      return true
    } catch {
      useAuthStore.getState().clear()
      return false
    } finally {
      refreshInFlight = null
    }
  })()
  return refreshInFlight
}

async function rawRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const url = path.startsWith('http') ? path : `${RAW_BASE}${path}`
  const headers = new Headers(opts.headers ?? {})
  headers.set('Accept', 'application/json')

  const token = useAuthStore.getState().accessToken
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  let body: BodyInit | undefined
  if (opts.body !== undefined && opts.body !== null) {
    if (opts.body instanceof FormData) {
      body = opts.body
    } else {
      body = JSON.stringify(opts.body)
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json')
      }
    }
  }

  const res = await fetch(url, {
    ...opts,
    headers,
    body,
    credentials: 'include',
  })

  if (res.status === 204) {
    return undefined as T
  }

  const contentType = res.headers.get('content-type') ?? ''
  const data = contentType.includes('application/json') ? await res.json() : await res.text()

  if (!res.ok) {
    throw new ApiError(res.status, typeof data === 'object' ? (data as ApiErrorBody) : null)
  }
  return data as T
}

/**
 * api() is the canonical entry point for all API calls. It transparently
 * refreshes an expired access token once and retries the original request.
 */
export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  try {
    return await rawRequest<T>(path, opts)
  } catch (err) {
    if (
      err instanceof ApiError &&
      err.status === 401 &&
      err.code === 'token_expired' &&
      !opts.noRefresh
    ) {
      const ok = await refreshOnce()
      if (ok) {
        return rawRequest<T>(path, { ...opts, noRefresh: true })
      }
    }
    throw err
  }
}

// Convenience helpers ------------------------------------------------------

export const apiGet = <T>(path: string, opts: RequestOptions = {}): Promise<T> =>
  api<T>(path, { ...opts, method: 'GET' })

export const apiPost = <T>(path: string, body?: unknown, opts: RequestOptions = {}): Promise<T> =>
  api<T>(path, { ...opts, method: 'POST', body })

export const apiPatch = <T>(path: string, body?: unknown, opts: RequestOptions = {}): Promise<T> =>
  api<T>(path, { ...opts, method: 'PATCH', body })

export const apiDelete = <T>(path: string, opts: RequestOptions = {}): Promise<T> =>
  api<T>(path, { ...opts, method: 'DELETE' })
