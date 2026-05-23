// Admin user management hooks. Each mutation invalidates the list so
// the table reflects the new state without a manual refetch.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPatch, apiPost } from '../lib/api'

export interface AdminUser {
  id: string
  email: string
  full_name: string
  display_name?: string | null
  role: 'super_admin' | 'admin' | 'moderator' | 'contributor' | 'viewer'
  status: 'pending' | 'approved' | 'suspended' | 'rejected'
  phone?: string | null
  avatar_url?: string | null
  bio?: string | null
  last_login_at?: string | null
  approved_at?: string | null
  created_at: string
  updated_at: string
}

interface ListEnvelope {
  data: AdminUser[]
  page: { limit: number }
}

interface ListParams {
  status?: string
  role?: string
  q?: string
  limit?: number
}

export function useAdminUsers(params: ListParams = {}) {
  const qs = new URLSearchParams()
  if (params.status) qs.set('status', params.status)
  if (params.role) qs.set('role', params.role)
  if (params.q) qs.set('q', params.q)
  if (params.limit) qs.set('limit', String(params.limit))
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return useQuery<AdminUser[]>({
    queryKey: ['admin-users', params],
    queryFn: async () => (await apiGet<ListEnvelope>(`/api/v1/admin/users${suffix}`)).data,
    staleTime: 15_000,
  })
}

function useUserAction(verb: 'approve' | 'reject' | 'suspend' | 'reactivate') {
  const qc = useQueryClient()
  return useMutation<AdminUser, Error, string>({
    mutationFn: (id) => apiPost<AdminUser>(`/api/v1/admin/users/${id}/${verb}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-users'] }),
  })
}

export const useApproveUser = () => useUserAction('approve')
export const useRejectUser = () => useUserAction('reject')
export const useSuspendUser = () => useUserAction('suspend')
export const useReactivateUser = () => useUserAction('reactivate')

export function useSetUserRole() {
  const qc = useQueryClient()
  return useMutation<AdminUser, Error, { id: string; role: string }>({
    mutationFn: ({ id, role }) =>
      apiPatch<AdminUser>(`/api/v1/admin/users/${id}/role`, { role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-users'] }),
  })
}
