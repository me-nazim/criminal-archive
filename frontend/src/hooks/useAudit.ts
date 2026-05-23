import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../lib/api'

export interface AuditRow {
  id: number
  user_id?: string | null
  action: string
  target_type?: string | null
  target_id?: string | null
  metadata?: Record<string, unknown>
  ip_address?: string | null
  user_agent?: string | null
  created_at: string
}

interface AuditEnvelope {
  data: AuditRow[]
  page: { limit: number; next_cursor: number | null }
}

export interface AuditFilters {
  action?: string
  target_type?: string
  user_id?: string
  since?: string
  until?: string
}

export function useAudit(filters: AuditFilters = {}) {
  const q = new URLSearchParams()
  Object.entries(filters).forEach(([k, v]) => {
    if (v) q.set(k, String(v))
  })
  const suffix = q.toString() ? `?${q.toString()}` : ''
  return useQuery<AuditEnvelope>({
    queryKey: ['audit', filters],
    queryFn: () => apiGet<AuditEnvelope>(`/api/v1/admin/audit${suffix}`),
  })
}

export interface DashboardStats {
  cases: Record<string, number>
  persons: Record<string, number>
  users: Record<string, number>
  public_attachments: number
}

export function useDashboardStats() {
  return useQuery<DashboardStats>({
    queryKey: ['admin-stats'],
    queryFn: () => apiGet<DashboardStats>('/api/v1/admin/stats'),
    staleTime: 30_000,
  })
}
