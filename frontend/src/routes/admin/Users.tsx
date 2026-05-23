// Full user management table with filters + per-row role / status
// actions. Super-admin role changes are guarded server-side too.

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Select } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState, ErrorState, LoadingState } from '../../components/ui/States'
import {
  useAdminUsers,
  useApproveUser,
  useRejectUser,
  useReactivateUser,
  useSetUserRole,
  useSuspendUser,
  type AdminUser,
} from '../../hooks/useAdminUsers'
import { useAuthStore } from '../../lib/auth-store'

const STATUS_TONE: Record<AdminUser['status'], 'success' | 'warning' | 'danger' | 'neutral'> = {
  approved: 'success',
  pending: 'warning',
  suspended: 'danger',
  rejected: 'neutral',
}

const ROLES = ['viewer', 'contributor', 'moderator', 'admin', 'super_admin'] as const

export default function Users() {
  const { t } = useTranslation()
  const me = useAuthStore((s) => s.user)
  const [filters, setFilters] = useState({ status: '', role: '', q: '' })

  const list = useAdminUsers(filters)
  const approve = useApproveUser()
  const reject = useRejectUser()
  const suspend = useSuspendUser()
  const reactivate = useReactivateUser()
  const setRole = useSetUserRole()

  const isMe = (u: AdminUser) => me?.id === u.id

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.users.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.users.subtitle')}</p>
      </header>

      <Card>
        <CardHeader>
          <div className="grid gap-3 sm:grid-cols-3">
            <TextField
              label={t('admin.users.search')}
              placeholder="email or name"
              value={filters.q}
              onChange={(e) => setFilters((f) => ({ ...f, q: e.target.value }))}
            />
            <Select
              label={t('admin.users.filter_status')}
              placeholder={t('admin.users.any') ?? ''}
              options={[
                { value: '', label: t('admin.users.any') ?? 'Any' },
                { value: 'pending', label: t('user_status.pending') },
                { value: 'approved', label: t('user_status.approved') },
                { value: 'suspended', label: t('user_status.suspended') },
                { value: 'rejected', label: t('user_status.rejected') },
              ]}
              value={filters.status}
              onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
            />
            <Select
              label={t('admin.users.filter_role')}
              placeholder={t('admin.users.any') ?? ''}
              options={[
                { value: '', label: t('admin.users.any') ?? 'Any' },
                ...ROLES.map((r) => ({ value: r, label: t(`role.${r}`) })),
              ]}
              value={filters.role}
              onChange={(e) => setFilters((f) => ({ ...f, role: e.target.value }))}
            />
          </div>
        </CardHeader>

        {list.isPending && <LoadingState />}
        {list.isError && <ErrorState onRetry={() => list.refetch()} />}
        {list.data && list.data.length === 0 && (
          <EmptyState title={t('admin.users.empty')} />
        )}

        {list.data && list.data.length > 0 && (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-ink-100">
              <thead className="bg-ink-50">
                <tr className="text-left text-xs font-medium uppercase tracking-wide text-ink-500">
                  <th className="px-6 py-3">{t('admin.users.col_user')}</th>
                  <th className="px-6 py-3">{t('admin.users.col_role')}</th>
                  <th className="px-6 py-3">{t('admin.users.col_status')}</th>
                  <th className="px-6 py-3 text-right">{t('admin.users.col_actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-100">
                {list.data.map((u) => (
                  <tr key={u.id} className="text-sm">
                    <td className="px-6 py-3">
                      <p className="font-medium text-ink-900">{u.full_name}</p>
                      <p className="text-xs text-ink-500">{u.email}</p>
                    </td>
                    <td className="px-6 py-3">
                      <Select
                        aria-label="role"
                        className="h-8 px-2 py-1 text-xs"
                        options={ROLES.map((r) => ({ value: r, label: t(`role.${r}`) }))}
                        value={u.role}
                        disabled={isMe(u) || setRole.isPending}
                        onChange={(e) => setRole.mutate({ id: u.id, role: e.target.value })}
                      />
                    </td>
                    <td className="px-6 py-3">
                      <Badge tone={STATUS_TONE[u.status]}>{t(`user_status.${u.status}`)}</Badge>
                    </td>
                    <td className="px-6 py-3 text-right">
                      {!isMe(u) && (
                        <div className="flex justify-end gap-2">
                          {u.status === 'pending' && (
                            <>
                              <Button size="sm" variant="secondary" onClick={() => reject.mutate(u.id)}>
                                {t('admin.approvals.reject')}
                              </Button>
                              <Button size="sm" onClick={() => approve.mutate(u.id)}>
                                {t('admin.approvals.approve')}
                              </Button>
                            </>
                          )}
                          {u.status === 'approved' && (
                            <Button size="sm" variant="secondary" onClick={() => suspend.mutate(u.id)}>
                              {t('admin.users.suspend')}
                            </Button>
                          )}
                          {u.status === 'suspended' && (
                            <Button size="sm" onClick={() => reactivate.mutate(u.id)}>
                              {t('admin.users.reactivate')}
                            </Button>
                          )}
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <CardBody className="text-xs text-ink-500">
          {list.data && list.data.length > 0 && (
            <span>{list.data.length} {t('admin.users.rows_suffix')}</span>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
