// Admin settings hooks. The list endpoint returns *masked* values for
// secret fields ({ "__masked": true }) so the UI knows when to render a
// placeholder vs. the actual stored value.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost, api } from '../lib/api'

export interface SettingsRow {
  key: string
  value: unknown
  description?: string | null
  updated_at: string
}

export interface SettingsListResp {
  data: SettingsRow[]
  cipher_configured: boolean
}

export function useSettingsList() {
  return useQuery<SettingsListResp>({
    queryKey: ['admin-settings'],
    queryFn: () => apiGet<SettingsListResp>('/api/v1/admin/settings'),
    staleTime: 30_000,
  })
}

export function useSetting<T>(key: string) {
  return useQuery<{ key: string; value: T }>({
    queryKey: ['admin-settings', key],
    queryFn: () => apiGet<{ key: string; value: T }>(`/api/v1/admin/settings/${key}`),
    staleTime: 30_000,
  })
}

export function useUpdateSetting<T>(key: string) {
  const qc = useQueryClient()
  return useMutation<{ key: string; value: T }, Error, T>({
    mutationFn: (value) =>
      api<{ key: string; value: T }>(`/api/v1/admin/settings/${key}`, {
        method: 'PUT',
        body: { value },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-settings'] })
      qc.invalidateQueries({ queryKey: ['admin-settings', key] })
      qc.invalidateQueries({ queryKey: ['branding'] })
    },
  })
}

export function useTestEmail() {
  return useMutation<{ ok: boolean }, Error, { to: string; config: unknown }>({
    mutationFn: (payload) => apiPost<{ ok: boolean }>('/api/v1/admin/settings/email/test', payload),
  })
}

export function useTestStorage() {
  return useMutation<{ ok: boolean }, Error, unknown>({
    mutationFn: (cfg) => apiPost<{ ok: boolean }>('/api/v1/admin/settings/storage/test', cfg),
  })
}

/** Replace a `__masked` value with the empty string so submitting it
 *  preserves the existing secret on the server. */
export function unmask<T>(v: T): T {
  if (typeof v !== 'object' || v === null) return v
  const cloned: Record<string, unknown> = { ...(v as object as Record<string, unknown>) }
  for (const [k, val] of Object.entries(cloned)) {
    if (val && typeof val === 'object' && (val as Record<string, unknown>).__masked === true) {
      cloned[k] = ''
    } else if (val && typeof val === 'object' && !Array.isArray(val)) {
      cloned[k] = unmask(val)
    }
  }
  return cloned as T
}

/** True when the stored field is masked (secret already on the server). */
export function isMasked(v: unknown): boolean {
  return !!v && typeof v === 'object' && (v as Record<string, unknown>).__masked === true
}
