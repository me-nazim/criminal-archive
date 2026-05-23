import { Link, NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import LanguageSwitcher from './LanguageSwitcher'

export default function Header() {
  const { t } = useTranslation()

  const navItem = ({ isActive }: { isActive: boolean }) =>
    [
      'rounded-md px-3 py-2 text-sm font-medium transition-colors',
      isActive
        ? 'bg-ink-900 text-white'
        : 'text-ink-700 hover:bg-ink-100 hover:text-ink-900',
    ].join(' ')

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
          <NavLink to="/submit" className={navItem}>
            {t('nav.submit')}
          </NavLink>
        </nav>

        <div className="flex items-center gap-2">
          <LanguageSwitcher />
          <Link
            to="/login"
            className="hidden rounded-md border border-ink-300 px-3 py-1.5 text-sm font-medium text-ink-800 hover:bg-ink-100 sm:inline-block"
          >
            {t('nav.login')}
          </Link>
        </div>
      </div>
    </header>
  )
}
