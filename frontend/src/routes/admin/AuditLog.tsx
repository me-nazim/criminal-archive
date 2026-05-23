import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { TextField } from '../../components/ui/TextField'
import { Badge } from '../../components/ui/Badge'
import { LoadingState, EmptyState, ErrorState } from '../../components/ui/States'
import { useAudit } from '../../hooks/useAudit'

export default function AuditLog() {
  const { t } = useTranslation()
  const [filters, setFilters] = useState({ action: '', target_type: '', user_id: '' })
  const list = useAudit(filters)

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.audit.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.audit.subtitle')}</p>
      </header>

      <Card>
        <CardHeader>
          <div className="grid gap-3 sm:grid-cols-3">
            <TextField
              label={t('admin.audit.filter_action')}
              placeholder="case.publish"
              value={filters.action}
              onChange={(e) => setFilters((f) => ({ ...f, action: e.target.value }))}
            />
            <TextField
              label={t('admin.audit.filter_target')}
              placeholder="case"
              value={filters.target_type}
              onChange={(e) => setFilters((f) => ({ ...f, target_type: e.target.value }))}
            />
            <TextField
              label={t('admin.audit.filter_user')}
              placeholder="uuid"
              value={filters.user_id}
              onChange={(e) => setFilters((f) => ({ ...f, user_id: e.target.value }))}
            />
          </div>
        </CardHeader>

        {list.isPending && <LoadingState />}
        {list.isError && <ErrorState onRetry={() => list.refetch()} />}
        {list.data && list.data.data.length === 0 && (
          <EmptyState title={t('admin.audit.empty')} />
        )}

        {list.data && list.data.data.length > 0 && (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-ink-100">
              <thead className="bg-ink-50">
                <tr className="text-left text-xs font-medium uppercase tracking-wide text-ink-500">
                  <th className="px-4 py-2">{t('admin.audit.col_when')}</th>
                  <th className="px-4 py-2">{t('admin.audit.col_action')}</th>
                  <th className="px-4 py-2">{t('admin.audit.col_target')}</th>
                  <th className="px-4 py-2">{t('admin.audit.col_user')}</th>
                  <th className="px-4 py-2">{t('admin.audit.col_metadata')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-100 text-sm">
                {list.data.data.map((row) => (
                  <tr key={row.id} className="hover:bg-ink-50">
                    <td className="whitespace-nowrap px-4 py-2 text-xs text-ink-500">
                      {new Date(row.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-2">
                      <Badge>{row.action}</Badge>
                    </td>
                    <td className="px-4 py-2 text-xs">
                      <span className="text-ink-600">{row.target_type ?? '—'}</span>
                      {row.target_id && (
                        <span className="ml-1 font-mono text-ink-500">{row.target_id.slice(0, 8)}</span>
                      )}
                    </td>
                    <td className="px-4 py-2 font-mono text-xs text-ink-500">
                      {row.user_id ? row.user_id.slice(0, 8) : '—'}
                    </td>
                    <td className="px-4 py-2 text-xs text-ink-500">
                      {row.metadata ? (
                        <code className="block max-w-md overflow-hidden truncate">
                          {JSON.stringify(row.metadata)}
                        </code>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <CardBody className="text-xs text-ink-500">
          {list.data && (
            <span>
              {list.data.data.length} {t('admin.audit.rows_suffix')}
              {list.data.page.next_cursor != null && (
                <> · {t('admin.audit.more_available')}</>
              )}
            </span>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
