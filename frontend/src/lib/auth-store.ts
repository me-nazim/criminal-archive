// Auth store: holds the current access token + user. The refresh token
// lives in an httpOnly cookie and is never accessible to JavaScript.
//
// We persist a thin session marker (just the user id) so that on page
// reload we know whether to attempt a /auth/refresh on boot.

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Role = 'super_admin' | 'admin' | 'moderator' | 'contributor' | 'viewer'

export interface AuthUser {
  id: string
  email: string
  full_name: string
  display_name?: string | null
  role: Role
  status: 'pending' | 'approved' | 'suspended' | 'rejected'
  phone?: string | null
  avatar_url?: string | null
  bio?: string | null
  created_at: string
  last_login_at?: string | null
}

interface AuthState {
  /** The current access JWT, or null if logged out. Not persisted. */
  accessToken: string | null
  /** Currently authenticated user, or null. Persisted (without the token). */
  user: AuthUser | null
  /** True until the initial refresh attempt finishes on app boot. */
  bootstrapped: boolean

  setSession: (accessToken: string, user: AuthUser) => void
  setBootstrapped: (b: boolean) => void
  clear: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      user: null,
      bootstrapped: false,
      setSession: (accessToken, user) => set({ accessToken, user }),
      setBootstrapped: (b) => set({ bootstrapped: b }),
      clear: () => set({ accessToken: null, user: null }),
    }),
    {
      name: 'tip-auth',
      // Only persist the user; the access token is short-lived and we
      // re-acquire it via /auth/refresh on every reload.
      partialize: (s) => ({ user: s.user }) as Partial<AuthState>,
    },
  ),
)

// Helpers --------------------------------------------------------------------

const ROLE_RANK: Record<Role, number> = {
  viewer: 1,
  contributor: 2,
  moderator: 3,
  admin: 4,
  super_admin: 5,
}

/**
 * roleAtLeast returns true when the user has a role rank ≥ minimum.
 * It returns false for unauthenticated users.
 */
export function roleAtLeast(user: AuthUser | null, min: Role): boolean {
  if (!user) return false
  return ROLE_RANK[user.role] >= ROLE_RANK[min]
}
