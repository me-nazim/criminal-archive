// Public cases listing with a small filter panel. URL state is the
// source of truth so filtered views are shareable.

import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'

import { Container } from '../components/ui/Container'
import { CaseCard } from '../components/CaseCard'
import { TextField } from '../components/ui/TextField'
import { Select } from '../components/ui/Select'
import { Button } from '../components/ui/Button'
import { LoadingState, EmptyState, ErrorState } from '../components/ui/States'
import { useCrimeTypes } from '../hooks/useReferenceData'
import { usePublicCases, type CaseFilters } from '../hooks/useCases'

export default function Cases() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const [params, setParams] = useSearchParams()

  const filters: CaseFilters = useMemo(
    () => ({
      q: params.get('q') ?? undefined,
      crime_type_id: params.get('crime_type_id') ? Number(params.get('crime_type_id')) : undefined,
      year: params.get('year') ? Number(params.get('year')) : undefined,
      sort: (params.get('sort') as CaseFilters['sort']) ?? undefined,
    }),
    [params],
  )

  const list = usePublicCases(filters)
  const crimes = useCrimeTypes()

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
          {t('cases.title')}
        </h1>
        <p className="mt-2 text-sm text-ink-600">{t('cases.subtitle')}</p>
      </header>

      <section className="mb-6 grid gap-3 rounded-lg border border-ink-200 bg-white p-4 sm:grid-cols-4">
        <TextField
          label={t('cases.search')}
          placeholder={t('cases.search_placeholder') ?? ''}
          defaultValue={filters.q ?? ''}
          onBlur={(e) => setParam('q', e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') setParam('q', (e.target as HTMLInputElement).value)
          }}
        />
        <Select
          label={t('cases.filter_crime_type')}
          placeholder={t('common.any') ?? 'Any'}
          options={[
            { value: '', label: t('common.any') ?? 'Any' },
            ...((crimes.data ?? []).map((c) => ({
              value: c.id,
              label: isBN ? c.name_bn : c.name_en,
            }))),
          ]}
          value={filters.crime_type_id ?? ''}
          onChange={(e) => setParam('crime_type_id', e.target.value)}
        />
        <Select
          label={t('cases.filter_year')}
          placeholder={t('common.any') ?? 'Any'}
          options={[
            { value: '', label: t('common.any') ?? 'Any' },
            ...yearOptions(),
          ]}
          value={filters.year ?? ''}
          onChange={(e) => setParam('year', e.target.value)}
        />
        <Select
          label={t('cases.sort')}
          options={[
            { value: 'published_desc', label: t('cases.sort_published') },
            { value: 'incident_desc', label: t('cases.sort_incident') },
          ]}
          value={filters.sort ?? 'published_desc'}
          onChange={(e) => setParam('sort', e.target.value)}
        />
      </section>

      {list.isPending && <LoadingState />}
      {list.isError && <ErrorState onRetry={() => list.refetch()} retryLabel={t('common.retry')} />}
      {list.data && list.data.length === 0 && (
        <EmptyState
          title={t('cases.empty_title')}
          message={t('cases.empty_message')}
          action={
            <Button variant="secondary" onClick={() => setParams(new URLSearchParams())}>
              {t('cases.clear_filters')}
            </Button>
          }
        />
      )}

      {list.data && list.data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {list.data.map((c) => (
            <CaseCard key={c.id} c={c} />
          ))}
        </div>
      )}
    </Container>
  )
}

function yearOptions() {
  const now = new Date().getFullYear()
  const out = []
  for (let y = now; y >= now - 8; y--) {
    out.push({ value: y, label: String(y) })
  }
  return out
}
