import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'

import { Container } from '../components/ui/Container'
import { TextField } from '../components/ui/TextField'
import { Badge } from '../components/ui/Badge'
import { LoadingState, EmptyState } from '../components/ui/States'
import { CaseCard } from '../components/CaseCard'
import { apiGet } from '../lib/api'
import type { CaseRow } from '../hooks/useCases'
import type { Person } from '../hooks/usePersons'

interface SearchResult {
  q: string
  cases: CaseRow[]
  persons: Person[]
}

export default function Search() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''

  const search = useQuery<SearchResult>({
    queryKey: ['search', q],
    queryFn: () => apiGet<SearchResult>(`/api/v1/search?q=${encodeURIComponent(q)}`),
    enabled: q.length > 0,
  })

  return (
    <Container width="wide" className="py-10">
      <header className="mb-6">
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('search.title')}
        </h1>
        <TextField
          className="mt-3 max-w-xl"
          aria-label={t('search.input_label') ?? 'Search'}
          placeholder={t('search.placeholder') ?? ''}
          defaultValue={q}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              const v = (e.target as HTMLInputElement).value
              const next = new URLSearchParams(params)
              if (v) next.set('q', v)
              else next.delete('q')
              setParams(next, { replace: true })
            }
          }}
        />
      </header>

      {!q && (
        <EmptyState title={t('search.idle_title')} message={t('search.idle_message')} />
      )}
      {q && search.isPending && <LoadingState />}
      {q && search.data && search.data.cases.length === 0 && search.data.persons.length === 0 && (
        <EmptyState title={t('search.empty_title')} />
      )}

      {search.data && search.data.cases.length > 0 && (
        <section className="mb-10">
          <h2 className="mb-3 font-display text-lg text-ink-900">{t('nav.cases')}</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {search.data.cases.map((c) => (
              <CaseCard key={c.id} c={c} />
            ))}
          </div>
        </section>
      )}

      {search.data && search.data.persons.length > 0 && (
        <section>
          <h2 className="mb-3 font-display text-lg text-ink-900">{t('nav.persons')}</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {search.data.persons.map((p) => {
              const name = p.is_anonymous
                ? t('persons.anonymous')
                : (isBN ? p.full_name_bn : p.full_name_en) ||
                  p.full_name_bn ||
                  p.full_name_en ||
                  p.slug
              return (
                <Link
                  key={p.id}
                  to={`/persons/${p.slug}`}
                  className="flex items-start gap-4 rounded-lg border border-ink-200 bg-white p-4 hover:border-ink-300"
                >
                  <div className="h-12 w-12 shrink-0 overflow-hidden rounded-full bg-ink-100">
                    {!p.is_anonymous && p.photo_url && (
                      <img src={p.photo_url} alt="" className="h-full w-full object-cover" />
                    )}
                  </div>
                  <div className="min-w-0">
                    <p className="truncate font-medium text-ink-900">{name}</p>
                    <Badge>{t(`person_role.${p.primary_type}`)}</Badge>
                  </div>
                </Link>
              )
            })}
          </div>
        </section>
      )}
    </Container>
  )
}
