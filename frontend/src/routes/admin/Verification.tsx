// Verifier-facing queue ("My queue") and admin overview ("All").
// Verifiers see their open assignments and can jump straight into the
// admin case editor where the existing verify/reject buttons live.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Card, CardBody, CardHeader } from '../../components/ui/Card'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { LoadingState, EmptyState, ErrorState } from '../../components/ui/States'
import {
  useMyVerificationQueue,
  useAdminVerifications,
  useStartVerification,
  type Assignment,
} from '../../hooks/useVerification'

const TONE: Record<Assignment['status'], 'neutral' | 'info' | 'success' | 'danger' | 'warning'> = {
  unassigned: 'neutral',
  assigned: 'warning',
  in_progress: 'info',
  verified: 'success',
  rejected: 'danger',
}

export default function Verification() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'mine' | 'all'>('mine')

  const mine = useMyVerificationQueue(true)
  const all = useAdminVerifications()
  const start = useStartVerification()

  const list = tab === 'mine' ? mine : all

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.verification.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.verification.subtitle')}</p>
      </header>

      <div role="tablist" className="inline-flex rounded-md border border-ink-200 bg-white text-sm">
        {(['mine', 'all'] as const).map((k) => (
          <button
            key={k}
            type="button"
            role="tab"
            aria-selected={tab === k}
            onClick={() => setTab(k)}
            className={
              tab === k
                ? 'rounded-md bg-ink-900 px-4 py-2 font-medium text-white'
                : 'px-4 py-2 text-ink-700 hover:bg-ink-100'
            }
          >
            {t(`admin.verification.tab_${k}`)}
          </button>
        ))}
      </div>

      <Card>
        <CardHeader>
          <p className="text-sm text-ink-600">
            {tab === 'mine' ? t('admin.verification.mine_help') : t('admin.verification.all_help')}
          </p>
        </CardHeader>

        {list.isPending && <LoadingState />}
        {list.isError && <ErrorState onRetry={() => list.refetch()} />}
        {list.data && list.data.length === 0 && (
          <EmptyState
            title={t('admin.verification.empty_title')}
            message={t('admin.verification.empty_message')}
          />
        )}

        {list.data && list.data.length > 0 && (
          <ul className="divide-y divide-ink-100">
            {list.data.map((a) => (
              <li key={a.id}>
                <CardBody className="flex flex-wrap items-center justify-between gap-3">
                  <div className="min-w-0">
                    <Link
                      to={`/admin/cases/${a.case_id}`}
                      className="font-medium text-ink-900 hover:text-brand-700"
                    >
                      {a.case_title_bn || a.case_title_en || a.case_number}
                    </Link>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-500">
                      <span className="font-mono">{a.case_number}</span>
                      <Badge tone={TONE[a.status]}>{t(`verification_status.${a.status}`)}</Badge>
                      <Badge>{t(`case_status.${a.case_status}`)}</Badge>
                      <span>{t('admin.verification.assigned_at')} {new Date(a.assigned_at).toLocaleString()}</span>
                    </div>
                    {a.notes && (
                      <p className="mt-2 max-w-prose whitespace-pre-line rounded-md bg-ink-50 p-2 text-xs text-ink-600">
                        {a.notes}
                      </p>
                    )}
                  </div>
                  <div className="flex shrink-0 gap-2">
                    {tab === 'mine' && a.status === 'assigned' && (
                      <Button
                        size="sm"
                        variant="secondary"
                        loading={start.isPending && start.variables === a.id}
                        onClick={() => start.mutate(a.id)}
                      >
                        {t('admin.verification.start')}
                      </Button>
                    )}
                    <Link
                      to={`/admin/cases/${a.case_id}`}
                      className="rounded-md bg-ink-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-ink-800"
                    >
                      {t('admin.verification.review')}
                    </Link>
                  </div>
                </CardBody>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}
