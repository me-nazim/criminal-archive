// In-app notifications. Polled every 60s while the user is logged in.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost } from '../lib/api'
import { useAuthStore } from '../lib/auth-store'

export interface Notification {
  id: string
  kind: string
  title: string
  body?: string
  link?: string
  metadata?: Record<string, unknown>
  read_at?: string | null
  created_at: string
}

interface ListResp {
  data: Notification[]
  unread: number
}

export function useNotifications() {
  const isAuthed = useAuthStore((s) => !!s.user)
  return useQuery<ListResp>({
    queryKey: ['notifications'],
    queryFn: () => apiGet<ListResp>('/api/v1/notifications?limit=15'),
    enabled: isAuthed,
    refetchInterval: 60_000,
    staleTime: 15_000,
  })
}

export function useMarkNotificationRead() {
  const qc = useQueryClient()
  return useMutation<void, Error, string>({
    mutationFn: (id) => apiPost<void>(`/api/v1/notifications/${id}/read`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })
}

export function useMarkAllNotificationsRead() {
  const qc = useQueryClient()
  return useMutation<void, Error, void>({
    mutationFn: () => apiPost<void>('/api/v1/notifications/read-all'),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })
}
