import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'

import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Button } from '../../components/ui/Button'
import { Select } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { Badge } from '../../components/ui/Badge'
import { LoadingState, EmptyState, ErrorState } from '../../components/ui/States'
import { useAdminPersons, useApprovePerson } from '../../hooks/usePersons'

const STATUSES = [
  'pending_review',
  'in_verification',
  'approved',
  'published',
  'rejected',
  'archived',
]

export default function AdminPersons() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const [params, setParams] = useSearchParams()
  const filters = {
    status: params.get('status') ?? 'pending_review',
    q: params.get('q') ?? undefined,
  }
  const list = useAdminPersons(filters)
  const approve = useApprovePerson()

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
          {t('admin.persons.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.persons.subtitle')}</p>
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
        {list.data && list.data.length === 0 && <EmptyState title={t('admin.persons.empty')} />}

        {list.data && list.data.length > 0 && (
          <ul className="divide-y divide-ink-100">
            {list.data.map((p) => (
              <li key={p.id}>
                <CardBody className="flex flex-wrap items-center justify-between gap-3">
                  <div className="min-w-0">
                    <Link to={`/persons/${p.slug}`} className="text-sm font-medium text-ink-900 hover:text-brand-700">
                      {p.is_anonymous
                        ? t('persons.anonymous')
                        : (isBN ? p.full_name_bn : p.full_name_en) ||
                          p.full_name_bn ||
                          p.full_name_en ||
                          p.slug}
                    </Link>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-500">
                      <Badge>{t(`person_role.${p.primary_type}`)}</Badge>
                      <Badge tone="warning">{t(`case_status.${p.status}`)}</Badge>
                      <span>
                        {t('persons.case_count', { count: p.case_count ?? 0 })}
                      </span>
                    </div>
                  </div>
                  {p.status !== 'published' && (
                    <Button
                      size="sm"
                      onClick={() => approve.mutate(p.id)}
                      loading={approve.isPending && approve.variables === p.id}
                    >
                      {t('admin.persons.publish')}
                    </Button>
                  )}
                </CardBody>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}
