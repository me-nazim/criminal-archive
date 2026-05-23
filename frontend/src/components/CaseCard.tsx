// CaseCard renders a compact case summary used in lists. The locale
// switcher decides which title field to surface; the case_number is
// always shown in monospace as a stable identifier.

import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Badge } from './ui/Badge'
import type { CaseRow } from '../hooks/useCases'

const STATUS_TONE = {
  draft: 'neutral',
  pending_review: 'warning',
  in_verification: 'warning',
  approved: 'info',
  published: 'success',
  rejected: 'danger',
  archived: 'neutral',
} as const

export function CaseCard({ c }: { c: CaseRow }) {
  const { i18n, t } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const title = (isBN ? c.title_bn : c.title_en) || c.title_bn || c.title_en || c.case_number
  const summary = (isBN ? c.summary_bn : c.summary_en) ?? c.summary_bn ?? c.summary_en
  const tone = STATUS_TONE[c.status as keyof typeof STATUS_TONE] ?? 'neutral'

  return (
    <Link
      to={`/cases/${c.slug || c.case_number}`}
      className="group flex h-full flex-col rounded-lg border border-ink-200 bg-white p-4 shadow-sm transition-colors hover:border-ink-300"
    >
      {c.cover_image_url ? (
        <div className="aspect-[16/9] overflow-hidden rounded-md bg-ink-100">
          <img
            src={c.cover_image_url}
            alt=""
            className="h-full w-full object-cover transition-transform group-hover:scale-[1.02]"
            loading="lazy"
          />
        </div>
      ) : (
        <div className="aspect-[16/9] rounded-md bg-gradient-to-br from-ink-100 to-ink-200" />
      )}
      <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-ink-500">
        <span className="font-mono">{c.case_number}</span>
        <Badge tone={tone}>{t(`case_status.${c.status}`)}</Badge>
        {c.incident_date && (
          <span>{new Date(c.incident_date).toLocaleDateString()}</span>
        )}
      </div>
      <h3 className="mt-2 font-display text-base font-semibold text-ink-900 group-hover:text-brand-700">
        {title}
      </h3>
      {summary && <p className="mt-1 line-clamp-3 text-sm text-ink-600">{summary}</p>}
      <div className="mt-3 flex flex-wrap gap-1">
        {(c.tags ?? []).slice(0, 4).map((tag) => (
          <span key={tag} className="rounded bg-ink-100 px-2 py-0.5 text-xs text-ink-600">
            #{tag}
          </span>
        ))}
      </div>
    </Link>
  )
}
