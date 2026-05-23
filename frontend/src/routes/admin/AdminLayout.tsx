// Admin shell: persistent left navigation, RBAC-gated. Wrapping in
// RequireAuth ensures the shell never renders for an anonymous user.

import { NavLink, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { FileText, LayoutDashboard, ScrollText, ShieldAlert, ShieldCheck, Users, UserSquare } from 'lucide-react'
import { cn } from '../../lib/cn'

export default function AdminLayout() {
  const { t } = useTranslation()

  const itemCls = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
      isActive
        ? 'bg-ink-900 text-white'
        : 'text-ink-700 hover:bg-ink-100 hover:text-ink-900',
    )

  return (
    <div className="grid min-h-[calc(100vh-9rem)] grid-cols-1 gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[16rem_1fr] lg:px-8">
      <aside className="flex flex-col gap-1 lg:border-r lg:border-ink-200 lg:pr-4">
        <p className="px-3 py-2 text-xs font-semibold uppercase tracking-wide text-ink-500">
          {t('admin.nav_title')}
        </p>
        <NavLink to="/admin" end className={itemCls}>
          <LayoutDashboard className="h-4 w-4" aria-hidden />
          {t('admin.nav_dashboard')}
        </NavLink>
        <NavLink to="/admin/approvals" className={itemCls}>
          <ShieldAlert className="h-4 w-4" aria-hidden />
          {t('admin.nav_approvals')}
        </NavLink>
        <NavLink to="/admin/users" className={itemCls}>
          <Users className="h-4 w-4" aria-hidden />
          {t('admin.nav_users')}
        </NavLink>
        <NavLink to="/admin/cases" className={itemCls}>
          <FileText className="h-4 w-4" aria-hidden />
          {t('admin.nav_cases')}
        </NavLink>
        <NavLink to="/admin/persons" className={itemCls}>
          <UserSquare className="h-4 w-4" aria-hidden />
          {t('admin.nav_persons')}
        </NavLink>
        <NavLink to="/admin/verification" className={itemCls}>
          <ShieldCheck className="h-4 w-4" aria-hidden />
          {t('admin.nav_verification')}
        </NavLink>
        <NavLink to="/admin/audit-log" className={itemCls}>
          <ScrollText className="h-4 w-4" aria-hidden />
          {t('admin.nav_audit')}
        </NavLink>
      </aside>
      <main className="min-w-0">
        <Outlet />
      </main>
    </div>
  )
}
