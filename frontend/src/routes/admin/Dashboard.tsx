import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useAdminUsers } from '../../hooks/useAdminUsers'
import { Card, CardBody } from '../../components/ui/Card'

export default function Dashboard() {
  const { t } = useTranslation()
  const pending = useAdminUsers({ status: 'pending', limit: 100 })

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('admin.dashboard.title')}
        </h1>
        <p className="text-sm text-ink-600">{t('admin.dashboard.subtitle')}</p>
      </header>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <MetricCard
          label={t('admin.dashboard.pending_users')}
          value={pending.isPending ? '…' : String(pending.data?.length ?? 0)}
          href="/admin/approvals"
        />
        <MetricCard label={t('admin.dashboard.cases_pending')} value="—" />
        <MetricCard label={t('admin.dashboard.cases_published')} value="—" />
      </div>
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
    <Link to={href} className="block focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-500">
      {inner}
    </Link>
  )
}
