import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '../components/ui/Container'
import { TextField } from '../components/ui/TextField'
import { Select } from '../components/ui/Select'
import { Badge } from '../components/ui/Badge'
import { LoadingState, EmptyState, ErrorState } from '../components/ui/States'
import { usePublicPersons, type Person } from '../hooks/usePersons'

export default function Persons() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const [params, setParams] = useSearchParams()
  const filters = {
    primary_type: params.get('primary_type') ?? '',
    q: params.get('q') ?? '',
  }
  const list = usePublicPersons({
    primary_type: filters.primary_type || undefined,
    q: filters.q || undefined,
  })

  const setParam = (k: string, v: string) => {
    const next = new URLSearchParams(params)
    if (v) next.set(k, v)
    else next.delete(k)
    setParams(next, { replace: true })
  }

  return (
    <Container width="wide" className="py-10">
      <header className="mb-8">
        <h1 className="font-display text-3xl font-semibold text-ink-900">
          {t('persons.title')}
        </h1>
        <p className="mt-2 text-sm text-ink-600">{t('persons.subtitle')}</p>
      </header>

      <section className="mb-6 grid gap-3 rounded-lg border border-ink-200 bg-white p-4 sm:grid-cols-2">
        <TextField
          label={t('persons.search')}
          placeholder={t('persons.search_placeholder') ?? ''}
          defaultValue={filters.q}
          onBlur={(e) => setParam('q', e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') setParam('q', (e.target as HTMLInputElement).value)
          }}
        />
        <Select
          label={t('persons.filter_role')}
          options={[
            { value: '', label: t('common.any') ?? 'Any' },
            { value: 'victim', label: t('person_role.victim') },
            { value: 'accused', label: t('person_role.accused') },
            { value: 'witness', label: t('person_role.witness') },
            { value: 'other', label: t('person_role.other') },
          ]}
          value={filters.primary_type}
          onChange={(e) => setParam('primary_type', e.target.value)}
        />
      </section>

      {list.isPending && <LoadingState />}
      {list.isError && <ErrorState onRetry={() => list.refetch()} />}
      {list.data && list.data.length === 0 && (
        <EmptyState title={t('persons.empty_title')} message={t('persons.empty_message')} />
      )}

      {list.data && list.data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {list.data.map((p) => (
            <PersonCard key={p.id} p={p} isBN={isBN} />
          ))}
        </div>
      )}
    </Container>
  )
}

function PersonCard({ p, isBN }: { p: Person; isBN: boolean }) {
  const { t } = useTranslation()
  const name = p.is_anonymous
    ? t('persons.anonymous')
    : (isBN ? p.full_name_bn : p.full_name_en) || p.full_name_bn || p.full_name_en || '—'
  return (
    <Link
      to={`/persons/${p.slug}`}
      className="group flex items-start gap-4 rounded-lg border border-ink-200 bg-white p-4 hover:border-ink-300"
    >
      <div className="h-14 w-14 shrink-0 overflow-hidden rounded-full bg-ink-100">
        {!p.is_anonymous && p.photo_url && (
          <img src={p.photo_url} alt="" className="h-full w-full object-cover" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate font-display text-base font-semibold text-ink-900 group-hover:text-brand-700">
          {name}
        </p>
        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-ink-500">
          <Badge tone={badgeTone(p.primary_type)}>{t(`person_role.${p.primary_type}`)}</Badge>
          {p.case_count !== undefined && p.case_count > 0 && (
            <span>{t('persons.case_count', { count: p.case_count })}</span>
          )}
        </div>
        {p.occupation && (
          <p className="mt-1 truncate text-xs text-ink-600">{p.occupation}</p>
        )}
      </div>
    </Link>
  )
}

function badgeTone(t: string): 'success' | 'danger' | 'warning' | 'neutral' {
  switch (t) {
    case 'victim':
      return 'warning'
    case 'accused':
      return 'danger'
    case 'witness':
      return 'neutral'
    default:
      return 'neutral'
  }
}
