// Route guard. Wraps protected routes and either renders the children,
// redirects to /login (preserving the target), or shows a 403 page when
// the user is logged in but their role is insufficient.

import { type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore, type Role, roleAtLeast } from '../lib/auth-store'
import { LoadingState } from './ui/States'
import { Container } from './ui/Container'

interface Props {
  children: ReactNode
  /** Minimum role required, default 'contributor' (any logged-in user). */
  minRole?: Role
}

export function RequireAuth({ children, minRole = 'contributor' }: Props) {
  const location = useLocation()
  const { user, bootstrapped } = useAuthStore()

  if (!bootstrapped) {
    return (
      <Container width="reading">
        <LoadingState />
      </Container>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  if (user.status === 'pending') {
    return <Navigate to="/register/pending" replace />
  }

  if (!roleAtLeast(user, minRole)) {
    return <Navigate to="/forbidden" replace />
  }

  return <>{children}</>
}
