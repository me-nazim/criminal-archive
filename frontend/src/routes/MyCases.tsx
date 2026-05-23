// "My submissions" — every case the logged-in user has filed, regardless
// of status. The Status pill makes it obvious where each case sits in
// the verification + publication pipeline.

import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'

import { Container } from '../components/ui/Container'
import { Badge } from '../components/ui/Badge'
import { Card, CardBody } from '../components/ui/Card'
import { LoadingState, EmptyState, ErrorState } from '../components/ui/States'
import { useMyCases } from '../hooks/useCases'

const STATUS_TONE = {
  draft: 'neutral',
  pending_review: 'warning',
  in_verification: 'warning',
  approved: 'info',
  published: 'success',
  rejected: 'danger',
  archived: 'neutral',
} as const

export default function MyCases() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const list = useMyCases()

  return (
    <Container width="wide" className="py-10">
      <header className="mb-6 flex items-end justify-between">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink-900">
            {t('my_cases.title')}
          </h1>
          <p className="text-sm text-ink-600">{t('my_cases.subtitle')}</p>
        </div>
        <Link
          to="/me/cases/new"
          className="inline-flex items-center gap-2 rounded-md bg-ink-900 px-4 py-2 text-sm font-medium text-white hover:bg-ink-800"
        >
          <Plus className="h-4 w-4" aria-hidden />
          {t('my_cases.new')}
        </Link>
      </header>

      {list.isPending && <LoadingState />}
      {list.isError && <ErrorState onRetry={() => list.refetch()} />}
      {list.data && list.data.length === 0 && (
        <EmptyState
          title={t('my_cases.empty_title')}
          message={t('my_cases.empty_message')}
          action={
            <Link
              to="/me/cases/new"
              className="rounded-md bg-ink-900 px-4 py-2 text-sm font-medium text-white hover:bg-ink-800"
            >
              {t('my_cases.new')}
            </Link>
          }
        />
      )}

      {list.data && list.data.length > 0 && (
        <Card>
          <ul className="divide-y divide-ink-100">
            {list.data.map((c) => {
              const title = (isBN ? c.title_bn : c.title_en) || c.title_bn || c.case_number
              const tone = STATUS_TONE[c.status as keyof typeof STATUS_TONE] ?? 'neutral'
              return (
                <li key={c.id}>
                  <CardBody>
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div className="min-w-0">
                        <Link
                          to={`/me/cases/${c.id}/edit`}
                          className="font-display text-base font-semibold text-ink-900 hover:text-brand-700"
                        >
                          {title}
                        </Link>
                        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-500">
                          <span className="font-mono">{c.case_number}</span>
                          <Badge tone={tone}>{t(`case_status.${c.status}`)}</Badge>
                          <span>
                            {t('my_cases.updated')}{' '}
                            {new Date(c.updated_at).toLocaleDateString()}
                          </span>
                        </div>
                      </div>
                      <div className="flex shrink-0 gap-2">
                        <Link
                          to={`/me/cases/${c.id}/edit`}
                          className="rounded-md border border-ink-300 bg-white px-3 py-1.5 text-xs font-medium text-ink-800 hover:bg-ink-100"
                        >
                          {t('my_cases.edit')}
                        </Link>
                        {c.status === 'published' && (
                          <Link
                            to={`/cases/${c.slug || c.case_number}`}
                            className="rounded-md px-3 py-1.5 text-xs font-medium text-ink-700 hover:bg-ink-100"
                          >
                            {t('my_cases.view_public')}
                          </Link>
                        )}
                      </div>
                    </div>
                  </CardBody>
                </li>
              )
            })}
          </ul>
        </Card>
      )}
    </Container>
  )
}
