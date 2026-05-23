import { useState } from 'react'
import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  LogOut,
  Menu,
  Search as SearchIcon,
  ShieldCheck,
  UserCircle2,
  X,
} from 'lucide-react'

import LanguageSwitcher from './LanguageSwitcher'
import { Logo } from '../brand/Logo'
import NotificationsBell from './NotificationsBell'
import { useAuthStore, roleAtLeast } from '../../lib/auth-store'
import { useLogoutMutation } from '../../hooks/useAuth'
import { useBranding } from '../../lib/branding'
import { Button } from '../ui/Button'
import { cn } from '../../lib/cn'

export default function Header() {
  const { t, i18n } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const logout = useLogoutMutation()
  const nav = useNavigate()
  const { branding } = useBranding()
  const [mobileOpen, setMobileOpen] = useState(false)

  const isBengali = i18n.language?.startsWith('bn')

  const navItem = ({ isActive }: { isActive: boolean }) =>
    cn(
      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
      isActive
        ? 'bg-ink-900 text-white shadow-soft'
        : 'text-ink-700 hover:bg-ink-100 hover:text-ink-900',
    )

  const closeMobile = () => setMobileOpen(false)

  const navLinks = [
    { to: '/', label: t('nav.home'), end: true },
    { to: '/cases', label: t('nav.cases') },
    { to: '/persons', label: t('nav.persons') },
    { to: '/search', label: t('nav.search'), icon: SearchIcon },
  ]

  return (
    <header className="sticky top-0 z-30 border-b border-ink-200 bg-white/85 backdrop-blur supports-[backdrop-filter]:bg-white/70">
      <div className="container-page flex h-16 items-center justify-between gap-4">
        <Link to="/" className="flex items-center" aria-label={branding.site_name_en}>
          <Logo customSrc={branding.logo_url} bengali={isBengali} label={isBengali ? branding.site_name_bn : branding.site_name_en} />
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {navLinks.map((l) => (
            <NavLink key={l.to} to={l.to} end={l.end} className={navItem}>
              <span className="inline-flex items-center gap-1.5">
                {l.icon ? <l.icon className="h-4 w-4" aria-hidden /> : null}
                {l.label}
              </span>
            </NavLink>
          ))}
          {user && (
            <NavLink to="/me/cases" className={navItem}>
              {t('nav.submit')}
            </NavLink>
          )}
          {roleAtLeast(user, 'admin') && (
            <NavLink to="/admin" className={navItem}>
              <span className="inline-flex items-center gap-1.5">
                <ShieldCheck className="h-4 w-4" aria-hidden />
                {t('nav.admin')}
              </span>
            </NavLink>
          )}
        </nav>

        <div className="flex items-center gap-2">
          {user && <NotificationsBell />}
          <LanguageSwitcher />
          {user ? (
            <div className="flex items-center gap-1">
              <Link
                to="/me"
                className="hidden items-center gap-1.5 rounded-md border border-ink-300 px-3 py-1.5 text-sm font-medium text-ink-800 hover:bg-ink-100 sm:inline-flex"
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
              <Link to="/register" className="btn-primary text-sm">
                {t('nav.register')}
              </Link>
            </>
          )}
          <button
            type="button"
            onClick={() => setMobileOpen((o) => !o)}
            className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-ink-300 text-ink-800 hover:bg-ink-100 md:hidden"
            aria-label="Toggle menu"
            aria-expanded={mobileOpen}
          >
            {mobileOpen ? <X className="h-4 w-4" aria-hidden /> : <Menu className="h-4 w-4" aria-hidden />}
          </button>
        </div>
      </div>

      {mobileOpen && (
        <nav className="border-t border-ink-200 bg-white md:hidden">
          <div className="container-page flex flex-col gap-1 py-3">
            {navLinks.map((l) => (
              <NavLink key={l.to} to={l.to} end={l.end} className={navItem} onClick={closeMobile}>
                <span className="inline-flex items-center gap-2">
                  {l.icon ? <l.icon className="h-4 w-4" aria-hidden /> : null}
                  {l.label}
                </span>
              </NavLink>
            ))}
            {user && (
              <NavLink to="/me/cases" className={navItem} onClick={closeMobile}>
                {t('nav.submit')}
              </NavLink>
            )}
            {roleAtLeast(user, 'admin') && (
              <NavLink to="/admin" className={navItem} onClick={closeMobile}>
                {t('nav.admin')}
              </NavLink>
            )}
            {!user && (
              <NavLink to="/login" className={navItem} onClick={closeMobile}>
                {t('nav.login')}
              </NavLink>
            )}
          </div>
        </nav>
      )}
    </header>
  )
}
