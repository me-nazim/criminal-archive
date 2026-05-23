import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Badge } from '../../components/ui/Badge'
import { Select } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { LoadingState, EmptyState, ErrorState } from '../../components/ui/States'
import { useAdminCases, type CaseFilters } from '../../hooks/useCases'
import { cases as STATUSES } from '../../lib/case-status'

export default function AdminCases() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const [params, setParams] = useSearchParams()

  const filters: CaseFilters & { status?: string } = {
    status: params.get('status') ?? 'pending_review',
    q: params.get('q') ?? undefined,
  }
  const list = useAdminCases(filters)

  const setParam = (k: string, v: string) => {
    const next = new URLSearchParams(params)
    if (v) next.set(k, v)
    else next.delete(k)
    setParams(next, { replace: true })
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.cases.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.cases.subtitle')}</p>
      </header>

      <Card>
        <CardHeader>
          <div className="grid gap-3 sm:grid-cols-3">
            <Select
              label={t('admin.cases.filter_status')}
              options={STATUSES.map((s) => ({ value: s, label: t(`case_status.${s}`) }))}
              value={filters.status}
              onChange={(e) => setParam('status', e.target.value)}
            />
            <TextField
              label={t('admin.cases.search')}
              defaultValue={filters.q ?? ''}
              onBlur={(e) => setParam('q', e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') setParam('q', (e.target as HTMLInputElement).value)
              }}
            />
          </div>
        </CardHeader>

        {list.isPending && <LoadingState />}
        {list.isError && <ErrorState onRetry={() => list.refetch()} />}
        {list.data && list.data.length === 0 && <EmptyState title={t('admin.cases.empty')} />}

        {list.data && list.data.length > 0 && (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-ink-100">
              <thead className="bg-ink-50">
                <tr className="text-left text-xs font-medium uppercase tracking-wide text-ink-500">
                  <th className="px-6 py-3">{t('admin.cases.col_case')}</th>
                  <th className="px-6 py-3">{t('admin.cases.col_status')}</th>
                  <th className="px-6 py-3">{t('admin.cases.col_updated')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-100">
                {list.data.map((c) => (
                  <tr key={c.id} className="text-sm hover:bg-ink-50">
                    <td className="px-6 py-3">
                      <Link
                        to={`/admin/cases/${c.id}`}
                        className="font-medium text-ink-900 hover:text-brand-700"
                      >
                        {(isBN ? c.title_bn : c.title_en) || c.title_bn || c.case_number}
                      </Link>
                      <p className="font-mono text-xs text-ink-500">{c.case_number}</p>
                    </td>
                    <td className="px-6 py-3">
                      <Badge tone="warning">{t(`case_status.${c.status}`)}</Badge>
                    </td>
                    <td className="px-6 py-3 text-xs text-ink-500">
                      {new Date(c.updated_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <CardBody className="text-xs text-ink-500">
          {list.data && list.data.length > 0 && (
            <span>
              {list.data.length} {t('admin.users.rows_suffix')}
            </span>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
