// Public case detail page. Renders the full record — title, description,
// linked persons, timeline, evidence, news sources — pulling Bangla
// fields preferentially with English fallback.

import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ExternalLink } from 'lucide-react'

import { Container } from '../components/ui/Container'
import { Badge } from '../components/ui/Badge'
import { Card, CardBody } from '../components/ui/Card'
import { LoadingState, ErrorState } from '../components/ui/States'
import { usePublicCase } from '../hooks/useCases'
import { useCaseAttachments } from '../hooks/useAttachments'

export default function CaseDetail() {
  const { key } = useParams<{ key: string }>()
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')

  const detail = usePublicCase(key)
  const attachments = useCaseAttachments(detail.data?.case.id)

  if (detail.isPending) {
    return (
      <Container width="reading">
        <LoadingState />
      </Container>
    )
  }
  if (detail.isError || !detail.data) {
    return (
      <Container width="reading">
        <ErrorState
          title={t('cases.detail_error_title')}
          message={t('cases.detail_error_message')}
          onRetry={() => detail.refetch()}
        />
      </Container>
    )
  }

  const c = detail.data.case
  const persons = detail.data.persons
  const timeline = detail.data.timeline
  const news = detail.data.news_sources
  const title = (isBN ? c.title_bn : c.title_en) || c.title_bn || c.title_en
  const desc = (isBN ? c.description_bn : c.description_en) || c.description_bn || c.description_en
  const summary = (isBN ? c.summary_bn : c.summary_en) || c.summary_bn || c.summary_en

  return (
    <Container width="reading" className="py-10">
      <nav aria-label="breadcrumb" className="mb-4 text-xs text-ink-500">
        <Link to="/cases" className="hover:text-ink-800">
          {t('nav.cases')}
        </Link>
        <span className="mx-2 text-ink-400">/</span>
        <span className="font-mono">{c.case_number}</span>
      </nav>

      <header className="space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="success">{t(`case_status.${c.status}`)}</Badge>
          {c.case_status && <Badge tone="info">{c.case_status}</Badge>}
          {c.incident_date && (
            <span className="text-xs text-ink-500">
              {new Date(c.incident_date).toLocaleDateString()}
            </span>
          )}
        </div>
        <h1 className="font-display text-3xl font-semibold text-ink-900 sm:text-4xl">
          {title || c.case_number}
        </h1>
        {summary && <p className="text-lg text-ink-700">{summary}</p>}
      </header>

      {c.cover_image_url && (
        <figure className="my-8 overflow-hidden rounded-lg border border-ink-200">
          <img src={c.cover_image_url} alt="" className="w-full" />
        </figure>
      )}

      {desc && (
        <section className="prose prose-sm mt-8 max-w-none whitespace-pre-line text-ink-800">
          <h2 className="font-display text-xl">{t('cases.description_title')}</h2>
          <p>{desc}</p>
        </section>
      )}

      {persons.length > 0 && (
        <section className="mt-10">
          <h2 className="font-display text-xl text-ink-900">{t('cases.persons_title')}</h2>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            {persons.map((p) => (
              <Link
                key={`${p.person_id}-${p.role}`}
                to={`/persons/${p.person_slug}`}
                className="flex items-center gap-3 rounded-md border border-ink-200 bg-white p-3 hover:border-ink-300"
              >
                <div className="h-12 w-12 shrink-0 overflow-hidden rounded-full bg-ink-100">
                  {!p.is_anonymous && p.photo_url && (
                    <img src={p.photo_url} alt="" className="h-full w-full object-cover" />
                  )}
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-ink-900">
                    {p.is_anonymous
                      ? t('persons.anonymous')
                      : (isBN ? p.name_bn : p.name_en) || p.name_bn || p.name_en}
                  </p>
                  <p className="text-xs uppercase tracking-wide text-ink-500">
                    {t(`person_role.${p.role}`)}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}

      {timeline.length > 0 && (
        <section className="mt-10">
          <h2 className="font-display text-xl text-ink-900">{t('cases.timeline_title')}</h2>
          <ol className="mt-3 space-y-3 border-l border-ink-200 pl-4">
            {timeline.map((e) => (
              <li key={e.id} className="relative">
                <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-brand-500" />
                <p className="text-xs text-ink-500">
                  {new Date(e.event_date).toLocaleDateString()}
                  {e.event_time ? ` ${e.event_time}` : ''}
                </p>
                <p className="text-sm font-medium text-ink-900">
                  {(isBN ? e.title_bn : e.title_en) || e.title_bn}
                </p>
                {(e.description_bn || e.description_en) && (
                  <p className="mt-1 text-sm text-ink-700 whitespace-pre-line">
                    {(isBN ? e.description_bn : e.description_en) ||
                      e.description_bn ||
                      e.description_en}
                  </p>
                )}
                {e.source_url && (
                  <a
                    href={e.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-1 inline-flex items-center gap-1 text-xs text-brand-600 hover:underline"
                  >
                    <ExternalLink className="h-3 w-3" aria-hidden /> {t('cases.timeline_source')}
                  </a>
                )}
              </li>
            ))}
          </ol>
        </section>
      )}

      {attachments.data && attachments.data.length > 0 && (
        <section className="mt-10">
          <h2 className="font-display text-xl text-ink-900">{t('cases.evidence_title')}</h2>
          <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {attachments.data.map((a) => (
              <a
                key={a.id}
                href={a.public_url ?? '#'}
                target="_blank"
                rel="noopener noreferrer"
                className="block overflow-hidden rounded-md border border-ink-200 bg-white"
              >
                {a.mime_type.startsWith('image/') && a.public_url ? (
                  <img src={a.public_url} alt={a.original_filename} className="aspect-[4/3] w-full object-cover" />
                ) : (
                  <div className="flex aspect-[4/3] items-center justify-center bg-ink-50 text-xs text-ink-500">
                    {a.mime_type}
                  </div>
                )}
                <div className="p-2 text-xs text-ink-600">
                  <p className="truncate font-mono">{a.stored_filename}</p>
                  <p className="text-ink-500">{formatSize(a.size_bytes)}</p>
                </div>
              </a>
            ))}
          </div>
        </section>
      )}

      {news.length > 0 && (
        <section className="mt-10">
          <h2 className="font-display text-xl text-ink-900">{t('cases.news_title')}</h2>
          <ul className="mt-3 space-y-2">
            {news.map((n) => (
              <li key={n.id}>
                <a
                  href={n.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-start gap-2 text-sm text-brand-700 hover:underline"
                >
                  <ExternalLink className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
                  <span>
                    <span className="font-medium">{n.title || n.url}</span>
                    {n.source_name && <span className="ml-1 text-ink-500">— {n.source_name}</span>}
                  </span>
                </a>
              </li>
            ))}
          </ul>
        </section>
      )}

      <footer className="mt-12 border-t border-ink-200 pt-4 text-xs text-ink-500">
        <Card>
          <CardBody className="text-xs text-ink-500">
            {t('cases.footer_note', { count: c.view_count })}
          </CardBody>
        </Card>
      </footer>
    </Container>
  )
}

function formatSize(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
