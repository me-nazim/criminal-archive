// Public person profile. Shows the bio, location, and every case the
// person is linked to. For anonymous victims, identity fields are hidden
// (the API redacts them before sending).

import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

import { Container } from '../components/ui/Container'
import { Badge } from '../components/ui/Badge'
import { LoadingState, ErrorState, EmptyState } from '../components/ui/States'
import { CaseCard } from '../components/CaseCard'
import { usePerson } from '../hooks/usePersons'
import { useCasesForPerson } from '../hooks/useCases'

export default function PersonProfile() {
  const { slug } = useParams<{ slug: string }>()
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')

  const person = usePerson(slug)
  const cases = useCasesForPerson(slug)

  if (person.isPending) {
    return (
      <Container width="reading">
        <LoadingState />
      </Container>
    )
  }
  if (person.isError || !person.data) {
    return (
      <Container width="reading">
        <ErrorState onRetry={() => person.refetch()} />
      </Container>
    )
  }
  const p = person.data
  const name = p.is_anonymous
    ? t('persons.anonymous')
    : (isBN ? p.full_name_bn : p.full_name_en) || p.full_name_bn || p.full_name_en || '—'
  const bio = (isBN ? p.public_bio_bn : p.public_bio_en) || p.public_bio_bn || p.public_bio_en

  return (
    <Container width="reading" className="py-10">
      <header className="flex flex-col items-start gap-4 sm:flex-row">
        <div className="h-24 w-24 shrink-0 overflow-hidden rounded-full bg-ink-100">
          {!p.is_anonymous && p.photo_url && (
            <img src={p.photo_url} alt="" className="h-full w-full object-cover" />
          )}
        </div>
        <div className="min-w-0">
          <h1 className="font-display text-3xl font-semibold text-ink-900">{name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-ink-600">
            <Badge tone={badgeTone(p.primary_type)}>{t(`person_role.${p.primary_type}`)}</Badge>
            {p.aliases.length > 0 && (
              <span>{t('persons.aliases')}: {p.aliases.join(', ')}</span>
            )}
            {p.occupation && <span>{p.occupation}</span>}
            {p.designation && <span>{p.designation}</span>}
          </div>
        </div>
      </header>

      {bio && (
        <section className="prose prose-sm mt-8 max-w-none whitespace-pre-line text-ink-800">
          <h2 className="font-display text-lg">{t('persons.bio_title')}</h2>
          <p>{bio}</p>
        </section>
      )}

      <section className="mt-10">
        <h2 className="font-display text-xl text-ink-900">{t('persons.linked_cases')}</h2>
        {cases.isPending && <LoadingState />}
        {cases.isError && <ErrorState onRetry={() => cases.refetch()} />}
        {cases.data && cases.data.length === 0 && (
          <EmptyState title={t('persons.no_cases')} />
        )}
        {cases.data && cases.data.length > 0 && (
          <div className="mt-3 grid gap-4 sm:grid-cols-2">
            {cases.data.map((c) => (
              <CaseCard key={c.id} c={c} />
            ))}
          </div>
        )}
      </section>
    </Container>
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
