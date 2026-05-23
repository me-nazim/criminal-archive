import { NavLink, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Database, Mail, Palette, ToggleLeft } from 'lucide-react'

import { cn } from '../../../lib/cn'

/**
 * The /admin/settings shell. Two-column layout: a sub-nav that mirrors
 * the four `app_settings` rows, with the active page rendered on the
 * right via <Outlet />.
 */
export default function SettingsLayout() {
  const { t } = useTranslation()

  const itemCls = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
      isActive
        ? 'bg-ink-900 text-white'
        : 'text-ink-700 hover:bg-ink-100 hover:text-ink-900',
    )

  return (
    <div>
      <header className="mb-6">
        <h1 className="font-display text-2xl font-bold text-ink-900">
          {t('admin.settings.title')}
        </h1>
        <p className="mt-1 text-sm text-ink-600">{t('admin.settings.subtitle')}</p>
      </header>
      <div className="grid gap-6 lg:grid-cols-[14rem_1fr]">
        <nav className="flex flex-col gap-1">
          <NavLink to="/admin/settings/branding" className={itemCls}>
            <Palette className="h-4 w-4" aria-hidden />
            {t('admin.settings.nav_branding')}
          </NavLink>
          <NavLink to="/admin/settings/email" className={itemCls}>
            <Mail className="h-4 w-4" aria-hidden />
            {t('admin.settings.nav_email')}
          </NavLink>
          <NavLink to="/admin/settings/storage" className={itemCls}>
            <Database className="h-4 w-4" aria-hidden />
            {t('admin.settings.nav_storage')}
          </NavLink>
          <NavLink to="/admin/settings/features" className={itemCls}>
            <ToggleLeft className="h-4 w-4" aria-hidden />
            {t('admin.settings.nav_features')}
          </NavLink>
        </nav>
        <main className="min-w-0">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
