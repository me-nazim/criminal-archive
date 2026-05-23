import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LogOut, ShieldCheck, UserCircle2 } from 'lucide-react'

import LanguageSwitcher from './LanguageSwitcher'
import { useAuthStore, roleAtLeast } from '../../lib/auth-store'
import { useLogoutMutation } from '../../hooks/useAuth'
import { Button } from '../ui/Button'
import { cn } from '../../lib/cn'

export default function Header() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const logout = useLogoutMutation()
  const nav = useNavigate()

  const navItem = ({ isActive }: { isActive: boolean }) =>
    cn(
      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
      isActive
        ? 'bg-ink-900 text-white'
        : 'text-ink-700 hover:bg-ink-100 hover:text-ink-900',
    )

  return (
    <header className="sticky top-0 z-30 border-b border-ink-200 bg-white/80 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <Link to="/" className="flex items-center gap-2">
          <span className="inline-block h-8 w-8 rounded bg-brand-500" aria-hidden />
          <span className="font-display text-lg font-semibold text-ink-900">
            {t('site.short')}
          </span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          <NavLink to="/" end className={navItem}>
            {t('nav.home')}
          </NavLink>
          <NavLink to="/cases" className={navItem}>
            {t('nav.cases')}
          </NavLink>
          <NavLink to="/persons" className={navItem}>
            {t('nav.persons')}
          </NavLink>
          {user && (
            <NavLink to="/me/cases/new" className={navItem}>
              {t('nav.submit')}
            </NavLink>
          )}
          {roleAtLeast(user, 'admin') && (
            <NavLink to="/admin" className={navItem}>
              <span className="inline-flex items-center gap-1">
                <ShieldCheck className="h-4 w-4" aria-hidden />
                {t('nav.admin')}
              </span>
            </NavLink>
          )}
        </nav>

        <div className="flex items-center gap-2">
          <LanguageSwitcher />
          {user ? (
            <div className="flex items-center gap-1">
              <Link
                to="/me"
                className="hidden items-center gap-1 rounded-md border border-ink-300 px-3 py-1.5 text-sm font-medium text-ink-800 hover:bg-ink-100 sm:inline-flex"
              >
                <UserCircle2 className="h-4 w-4" aria-hidden />
                {user.full_name.split(' ')[0]}
              </Link>
              <Button
                size="sm"
                variant="ghost"
                aria-label={t('nav.logout') ?? 'Log out'}
                onClick={() => {
                  logout.mutate(undefined, {
                    onSettled: () => nav('/', { replace: true }),
                  })
                }}
                loading={logout.isPending}
              >
                <LogOut className="h-4 w-4" aria-hidden />
              </Button>
            </div>
          ) : (
            <>
              <Link
                to="/login"
                className="hidden rounded-md border border-ink-300 px-3 py-1.5 text-sm font-medium text-ink-800 hover:bg-ink-100 sm:inline-block"
              >
                {t('nav.login')}
              </Link>
              <Link
                to="/register"
                className="rounded-md bg-ink-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-ink-800"
              >
                {t('nav.register')}
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
