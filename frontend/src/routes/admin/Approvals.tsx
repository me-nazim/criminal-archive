// Pending registrations queue. One-click approve/reject, with optimistic
// invalidation handled by the mutation hooks.

import { useTranslation } from 'react-i18next'
import { Check, X } from 'lucide-react'
import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Button } from '../../components/ui/Button'
import { Badge } from '../../components/ui/Badge'
import { EmptyState, ErrorState, LoadingState } from '../../components/ui/States'
import {
  useAdminUsers,
  useApproveUser,
  useRejectUser,
} from '../../hooks/useAdminUsers'

export default function Approvals() {
  const { t } = useTranslation()
  const list = useAdminUsers({ status: 'pending', limit: 100 })
  const approve = useApproveUser()
  const reject = useRejectUser()

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.approvals.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.approvals.subtitle')}</p>
      </header>

      {list.isPending && <LoadingState />}
      {list.isError && <ErrorState onRetry={() => list.refetch()} />}
      {list.data && list.data.length === 0 && (
        <EmptyState
          title={t('admin.approvals.empty_title')}
          message={t('admin.approvals.empty_message')}
        />
      )}

      {list.data && list.data.length > 0 && (
        <Card>
          <CardHeader>
            <p className="text-sm font-medium text-ink-800">
              {list.data.length} {t('admin.approvals.pending_count_suffix')}
            </p>
          </CardHeader>
          <ul role="list" className="divide-y divide-ink-100">
            {list.data.map((u) => (
              <li key={u.id}>
                <CardBody className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-ink-900">{u.full_name}</p>
                    <p className="truncate text-xs text-ink-500">
                      {u.email}
                      {u.phone ? ` · ${u.phone}` : ''}
                    </p>
                    <div className="mt-1 flex flex-wrap items-center gap-2">
                      <Badge tone="warning">{t('user_status.pending')}</Badge>
                      <Badge>{t(`role.${u.role}`)}</Badge>
                      <span className="text-xs text-ink-500">
                        {new Date(u.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button
                      size="sm"
                      variant="secondary"
                      leftIcon={<X className="h-4 w-4" aria-hidden />}
                      loading={reject.isPending && reject.variables === u.id}
                      onClick={() => reject.mutate(u.id)}
                    >
                      {t('admin.approvals.reject')}
                    </Button>
                    <Button
                      size="sm"
                      leftIcon={<Check className="h-4 w-4" aria-hidden />}
                      loading={approve.isPending && approve.variables === u.id}
                      onClick={() => approve.mutate(u.id)}
                    >
                      {t('admin.approvals.approve')}
                    </Button>
                  </div>
                </CardBody>
              </li>
            ))}
          </ul>
        </Card>
      )}
    </div>
  )
}
