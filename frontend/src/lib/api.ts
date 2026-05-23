// Thin fetch wrapper for the Tansiq API.
// Real auth + error handling will be added later.

const BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080').replace(
  /\/+$/,
  '',
)

type Json = Record<string, unknown> | unknown[]

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public body?: unknown,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export async function api<T = unknown>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init.body && !(init.body instanceof FormData)
        ? { 'Content-Type': 'application/json' }
        : {}),
      ...(init.headers ?? {}),
    },
    credentials: 'include',
  })

  const contentType = res.headers.get('content-type') ?? ''
  const data: Json | string = contentType.includes('application/json')
    ? await res.json()
    : await res.text()

  if (!res.ok) {
    throw new ApiError(res.status, `API ${res.status} ${res.statusText}`, data)
  }
  return data as T
}

export const apiBaseUrl = BASE_URL
