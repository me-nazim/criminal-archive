// Mutation hooks around the auth endpoints. Each one updates the
// auth-store on success so consumers don't have to remember to.

import { useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost } from '../lib/api'
import { useAuthStore, type AuthUser } from '../lib/auth-store'

interface RegisterPayload {
  email: string
  password: string
  full_name: string
  phone?: string | null
}

interface LoginPayload {
  email: string
  password: string
}

interface LoginResponse {
  access_token: string
  expires_in: number
  user: AuthUser
}

interface ChangePasswordPayload {
  old_password: string
  new_password: string
}

export function useRegisterMutation() {
  return useMutation<AuthUser, Error, RegisterPayload>({
    mutationFn: (payload) => apiPost<AuthUser>('/api/v1/auth/register', payload),
  })
}

export function useLoginMutation() {
  const setSession = useAuthStore((s) => s.setSession)
  return useMutation<LoginResponse, Error, LoginPayload>({
    mutationFn: (payload) => apiPost<LoginResponse>('/api/v1/auth/login', payload),
    onSuccess: (data) => setSession(data.access_token, data.user),
  })
}

export function useLogoutMutation() {
  const clear = useAuthStore((s) => s.clear)
  const qc = useQueryClient()
  return useMutation<void, Error, void>({
    mutationFn: async () => {
      try {
        await apiPost<void>('/api/v1/auth/logout')
      } catch {
        // Logout is best-effort.
      }
    },
    onSuccess: () => {
      clear()
      qc.clear()
    },
  })
}

export function useChangePasswordMutation() {
  return useMutation<void, Error, ChangePasswordPayload>({
    mutationFn: (payload) => apiPost<void>('/api/v1/auth/password/change', payload),
  })
}

export function useRefreshMe() {
  const setSession = useAuthStore((s) => s.setSession)
  const accessToken = useAuthStore((s) => s.accessToken)
  return useMutation<AuthUser, Error, void>({
    mutationFn: async () => apiGet<AuthUser>('/api/v1/auth/me'),
    onSuccess: (user) => {
      // The token is unchanged; just refresh the user record.
      if (accessToken) setSession(accessToken, user)
    },
  })
}
