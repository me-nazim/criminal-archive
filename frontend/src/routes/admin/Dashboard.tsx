// Admin dashboard with live counts. The same admin /stats endpoint is
// used everywhere instead of issuing N separate count queries from the
// frontend.

import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Card, CardBody } from '../../components/ui/Card'
import { LoadingState, ErrorState } from '../../components/ui/States'
import { useDashboardStats } from '../../hooks/useAudit'
import { useAdminUsers } from '../../hooks/useAdminUsers'

export default function Dashboard() {
  const { t } = useTranslation()
  const stats = useDashboardStats()
  const pendingUsers = useAdminUsers({ status: 'pending', limit: 100 })

  if (stats.isPending) return <LoadingState />
  if (stats.isError) return <ErrorState onRetry={() => stats.refetch()} />

  const caseCounts = stats.data?.cases ?? {}
  const personCounts = stats.data?.persons ?? {}
  const userCounts = stats.data?.users ?? {}

  const sumExcept = (m: Record<string, number>, exclude: string[]) =>
    Object.entries(m)
      .filter(([k]) => !exclude.includes(k))
      .reduce((sum, [, v]) => sum + v, 0)

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.dashboard.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.dashboard.subtitle')}</p>
      </header>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label={t('admin.dashboard.pending_users')}
          value={String(pendingUsers.data?.length ?? userCounts['pending'] ?? 0)}
          href="/admin/approvals"
        />
        <MetricCard
          label={t('admin.dashboard.cases_pending')}
          value={String((caseCounts['pending_review'] ?? 0) + (caseCounts['in_verification'] ?? 0))}
          href="/admin/cases?status=pending_review"
        />
        <MetricCard
          label={t('admin.dashboard.cases_published')}
          value={String(caseCounts['published'] ?? 0)}
          href="/admin/cases?status=published"
        />
        <MetricCard
          label={t('admin.dashboard.persons_pending')}
          value={String(sumExcept(personCounts, ['published']))}
          href="/admin/persons"
        />
      </section>

      <section className="grid gap-4 sm:grid-cols-2">
        <BreakdownCard title={t('admin.dashboard.cases_by_status')} counts={caseCounts} prefix="case_status" />
        <BreakdownCard title={t('admin.dashboard.users_by_status')} counts={userCounts} prefix="user_status" />
      </section>
    </div>
  )
}

function MetricCard({
  label,
  value,
  href,
}: {
  label: string
  value: string
  href?: string
}) {
  const inner = (
    <Card className="hover:border-ink-300">
      <CardBody>
        <p className="text-xs uppercase tracking-wide text-ink-500">{label}</p>
        <p className="mt-2 font-display text-3xl font-semibold text-ink-900">{value}</p>
      </CardBody>
    </Card>
  )
  if (!href) return inner
  return (
    <Link
      to={href}
      className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-500"
    >
      {inner}
    </Link>
  )
}

function BreakdownCard({
  title,
  counts,
  prefix,
}: {
  title: string
  counts: Record<string, number>
  prefix: string
}) {
  const { t } = useTranslation()
  const entries = Object.entries(counts).sort((a, b) => b[1] - a[1])
  return (
    <Card>
      <CardBody>
        <p className="text-xs uppercase tracking-wide text-ink-500">{title}</p>
        {entries.length === 0 ? (
          <p className="mt-2 text-sm text-ink-500">—</p>
        ) : (
          <ul className="mt-3 space-y-1.5">
            {entries.map(([k, v]) => (
              <li key={k} className="flex items-center justify-between text-sm">
                <span className="text-ink-700">{t(`${prefix}.${k}`)}</span>
                <span className="font-mono text-ink-500">{v}</span>
              </li>
            ))}
          </ul>
        )}
      </CardBody>
    </Card>
  )
}
