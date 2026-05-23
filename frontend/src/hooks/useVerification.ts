import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost } from '../lib/api'

export interface Assignment {
  id: string
  case_id: string
  case_number: string
  case_slug: string
  case_title_bn: string
  case_title_en?: string | null
  case_status: string
  assigned_to?: string | null
  assigned_by?: string | null
  status: 'unassigned' | 'assigned' | 'in_progress' | 'verified' | 'rejected'
  notes?: string | null
  assigned_at: string
  completed_at?: string | null
}

interface ListEnvelope<T> {
  data: T[]
}

export function useMyVerificationQueue(openOnly = true) {
  return useQuery<Assignment[]>({
    queryKey: ['verification', 'mine', openOnly],
    queryFn: async () =>
      (await apiGet<ListEnvelope<Assignment>>(
        `/api/v1/verification/queue${openOnly ? '' : '?open=false'}`,
      )).data,
  })
}

export function useAdminVerifications(status?: string) {
  return useQuery<Assignment[]>({
    queryKey: ['verification', 'admin', status],
    queryFn: async () => {
      const qs = status ? `?status=${encodeURIComponent(status)}` : ''
      return (await apiGet<ListEnvelope<Assignment>>(`/api/v1/admin/verification${qs}`)).data
    },
  })
}

export function useStartVerification() {
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (id) => apiPost<void>(`/api/v1/verification/${id}/start`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['verification'] }),
  })
}

export function useAppendVerificationNote() {
  const qc = useQueryClient()
  return useMutation<void, Error, { id: string; note: string }>({
    mutationFn: ({ id, note }) => apiPost<void>(`/api/v1/verification/${id}/notes`, { note }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['verification'] }),
  })
}
