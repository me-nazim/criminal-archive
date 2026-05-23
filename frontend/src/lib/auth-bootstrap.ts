// Auth bootstrap: at app boot, if we believe the user *might* be logged
// in (we have a persisted user record), we POST to /auth/refresh to
// exchange the httpOnly refresh cookie for a fresh access token.
//
// On success we update the auth store; on failure we clear it.

import { apiBaseUrl } from './api'
import { useAuthStore, type AuthUser } from './auth-store'

interface RefreshResponse {
  access_token: string
  expires_in: number
  user: AuthUser
}

export async function bootstrapAuth(): Promise<void> {
  const { user, setSession, clear, setBootstrapped } = useAuthStore.getState()

  // No persisted user → nothing to do (still mark bootstrapped so guards
  // stop showing a loading spinner).
  if (!user) {
    setBootstrapped(true)
    return
  }

  try {
    const res = await fetch(`${apiBaseUrl}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) {
      clear()
      return
    }
    const data = (await res.json()) as RefreshResponse
    setSession(data.access_token, data.user)
  } catch {
    clear()
  } finally {
    setBootstrapped(true)
  }
}
