import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  ArrowRight,
  CheckCircle2,
  FileSearch,
  Globe2,
  Lock,
  ScrollText,
  ShieldCheck,
  Users2,
} from 'lucide-react'

import { useBranding } from '../lib/branding'
import { usePublicCases } from '../hooks/useCases'

/**
 * The public landing page. Five-section layout:
 *   1. Hero       — headline, sub-copy, primary + secondary CTAs.
 *   2. Trust bar  — verification, encryption, multilingual signals.
 *   3. Features   — what the portal offers visitors and contributors.
 *   4. Recent     — three most-recently-published cases (lazy + cached).
 *   5. CTA        — final "submit" / "browse" pair, on a tinted card.
 */
export default function Home() {
  const { t, i18n } = useTranslation()
  const { branding } = useBranding()
  const isBN = i18n.language?.startsWith('bn')
  const recent = usePublicCases({ limit: 3, sort: 'published_desc' })

  const siteName = isBN ? branding.site_name_bn : branding.site_name_en
  const tagline = isBN ? branding.tagline_bn : branding.tagline_en

  return (
    <>
      {/* ---------- 1. HERO ---------- */}
      <section className="relative overflow-hidden border-b border-ink-200 bg-white">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-grid-light [background-size:24px_24px] opacity-60"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-hero-fade"
        />
        <div className="container-page relative grid gap-12 py-20 sm:py-24 lg:grid-cols-[1.1fr_1fr] lg:items-center">
          <div className="animate-fade-in">
            <div className="chip mb-5 border-brand-200 bg-brand-50 text-brand-700">
              <ShieldCheck className="h-3.5 w-3.5" aria-hidden />
              {t('home.badge')}
            </div>
            <h1 className="font-display text-4xl font-bold leading-[1.1] tracking-tight text-ink-900 sm:text-5xl lg:text-[3.4rem]">
              {t('home.hero_title')}
            </h1>
            <p className="mt-5 max-w-xl text-lg leading-relaxed text-ink-600">
              {t('home.hero_subtitle')}
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <Link to="/cases" className="btn-primary">
                {t('home.browse_cases')}
                <ArrowRight className="h-4 w-4" aria-hidden />
              </Link>
              <Link to="/me/cases/new" className="btn-ghost">
                {t('home.submit_info')}
              </Link>
            </div>
            <p className="mt-6 text-xs text-ink-500">{tagline} · {siteName}</p>
          </div>

          {/* Hero side card: a stylised case preview to convey the product */}
          <HeroPreviewCard />
        </div>
      </section>

      {/* ---------- 2. TRUST BAR ---------- */}
      <section className="border-b border-ink-200 bg-ink-50/60">
        <div className="container-page grid gap-6 py-8 sm:grid-cols-3">
          <TrustItem icon={ShieldCheck} title={t('home.trust_verify')} body={t('home.trust_verify_body')} />
          <TrustItem icon={Lock} title={t('home.trust_evidence')} body={t('home.trust_evidence_body')} />
          <TrustItem icon={Globe2} title={t('home.trust_languages')} body={t('home.trust_languages_body')} />
        </div>
      </section>

      {/* ---------- 3. FEATURES ---------- */}
      <section className="container-page py-20">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-semibold uppercase tracking-[0.18em] text-brand-600">
            {t('home.features_kicker')}
          </p>
          <h2 className="mt-3 font-display text-3xl font-bold tracking-tight text-ink-900 sm:text-4xl">
            {t('home.features_title')}
          </h2>
          <p className="mt-4 text-ink-600">{t('home.features_subtitle')}</p>
        </div>
        <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <FeatureCard
            icon={FileSearch}
            title={t('home.feat_browse_title')}
            body={t('home.feat_browse_body')}
          />
          <FeatureCard
            icon={ScrollText}
            title={t('home.feat_timeline_title')}
            body={t('home.feat_timeline_body')}
          />
          <FeatureCard
            icon={Users2}
            title={t('home.feat_people_title')}
            body={t('home.feat_people_body')}
          />
          <FeatureCard
            icon={CheckCircle2}
            title={t('home.feat_verify_title')}
            body={t('home.feat_verify_body')}
          />
        </div>
      </section>

      {/* ---------- 4. RECENTLY PUBLISHED ---------- */}
      <section className="border-y border-ink-200 bg-white">
        <div className="container-page py-20">
          <div className="flex items-end justify-between gap-4">
            <div>
              <h2 className="font-display text-3xl font-bold tracking-tight text-ink-900">
                {t('home.recent_title')}
              </h2>
              <p className="mt-2 text-ink-600">{t('home.recent_subtitle')}</p>
            </div>
            <Link to="/cases" className="link-quiet hidden items-center gap-1 text-sm font-medium sm:inline-flex">
              {t('home.see_all')} <ArrowRight className="h-4 w-4" aria-hidden />
            </Link>
          </div>
          <div className="mt-8 grid gap-4 md:grid-cols-3">
            {recent.isLoading && Array.from({ length: 3 }).map((_, i) => <SkeletonCard key={i} />)}
            {!recent.isLoading && recent.data?.length === 0 && (
              <p className="text-sm text-ink-500">{t('home.recent_empty')}</p>
            )}
            {recent.data?.slice(0, 3).map((c) => (
              <Link
                key={c.id}
                to={`/cases/${c.case_number}`}
                className="card-soft group flex flex-col gap-2 p-5 transition hover:-translate-y-0.5 hover:shadow-elevated"
              >
                <span className="text-xs font-semibold uppercase tracking-wide text-brand-600">
                  {c.case_number}
                </span>
                <h3 className="font-display text-lg font-semibold leading-snug text-ink-900 group-hover:text-ink-950">
                  {isBN ? c.title_bn : c.title_en || c.title_bn}
                </h3>
                <p className="line-clamp-3 text-sm text-ink-600">
                  {(isBN ? c.summary_bn : c.summary_en) ?? c.summary_bn ?? ''}
                </p>
                <div className="mt-2 flex items-center gap-2 text-xs text-ink-500">
                  {c.published_at && <span>{new Date(c.published_at).toLocaleDateString()}</span>}
                  {c.tags?.slice(0, 2).map((tag) => (
                    <span key={tag} className="chip">{tag}</span>
                  ))}
                </div>
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* ---------- 5. CTA ---------- */}
      <section className="container-page py-20">
        <div className="card-soft relative overflow-hidden bg-ink-900 p-10 text-white sm:p-14">
          <div
            aria-hidden
            className="pointer-events-none absolute -top-24 -right-24 h-64 w-64 rounded-full opacity-25 blur-3xl"
            style={{ background: 'var(--brand-primary)' }}
          />
          <div className="relative grid gap-6 sm:grid-cols-[1.5fr_1fr] sm:items-end">
            <div>
              <h2 className="font-display text-3xl font-bold tracking-tight sm:text-4xl">
                {t('home.cta_title')}
              </h2>
              <p className="mt-3 max-w-xl text-ink-200">{t('home.cta_subtitle')}</p>
            </div>
            <div className="flex flex-wrap gap-3 sm:justify-end">
              <Link to="/register" className="btn-primary">
                {t('home.cta_register')}
              </Link>
              <Link
                to="/cases"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/30 px-4 py-2 text-sm font-semibold text-white hover:bg-white/10"
              >
                {t('home.cta_browse')}
              </Link>
            </div>
          </div>
        </div>
      </section>
    </>
  )
}

function TrustItem({
  icon: Icon,
  title,
  body,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  body: string
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-ink-200 bg-white text-ink-700">
        <Icon className="h-4 w-4" aria-hidden />
      </div>
      <div>
        <p className="text-sm font-semibold text-ink-900">{title}</p>
        <p className="text-sm text-ink-600">{body}</p>
      </div>
    </div>
  )
}

function FeatureCard({
  icon: Icon,
  title,
  body,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  body: string
}) {
  return (
    <article className="card-soft flex flex-col gap-3 p-5 transition hover:-translate-y-0.5 hover:shadow-elevated">
      <div
        className="inline-flex h-10 w-10 items-center justify-center rounded-lg text-white"
        style={{ background: 'var(--brand-primary)' }}
      >
        <Icon className="h-5 w-5" aria-hidden />
      </div>
      <h3 className="font-display text-lg font-semibold tracking-tight text-ink-900">{title}</h3>
      <p className="text-sm leading-relaxed text-ink-600">{body}</p>
    </article>
  )
}

function HeroPreviewCard() {
  const { t } = useTranslation()
  return (
    <div className="card-soft relative overflow-hidden p-6 sm:p-8 lg:ml-auto lg:max-w-md">
      <div className="flex items-center justify-between">
        <span className="chip border-brand-200 bg-brand-50 text-brand-700">TIP-2026-00045</span>
        <span className="chip">{t('case_status.published')}</span>
      </div>
      <h3 className="mt-4 font-display text-xl font-semibold leading-snug text-ink-900">
        {t('home.preview_title')}
      </h3>
      <p className="mt-2 text-sm text-ink-600">{t('home.preview_summary')}</p>
      <div className="mt-5 grid grid-cols-3 divide-x divide-ink-200 rounded-md border border-ink-200 bg-white text-center text-xs">
        <div className="p-3">
          <div className="font-semibold text-ink-900">12</div>
          <div className="text-ink-500">{t('home.preview_evidence')}</div>
        </div>
        <div className="p-3">
          <div className="font-semibold text-ink-900">7</div>
          <div className="text-ink-500">{t('home.preview_persons')}</div>
        </div>
        <div className="p-3">
          <div className="font-semibold text-ink-900">9</div>
          <div className="text-ink-500">{t('home.preview_sources')}</div>
        </div>
      </div>
      <div className="mt-5 flex items-center gap-2 text-xs text-ink-500">
        <ShieldCheck className="h-3.5 w-3.5 text-brand-600" aria-hidden />
        {t('home.preview_verified')}
      </div>
    </div>
  )
}

function SkeletonCard() {
  return (
    <div className="card-soft p-5">
      <div className="h-3 w-20 animate-pulse rounded bg-ink-200" />
      <div className="mt-3 h-5 w-3/4 animate-pulse rounded bg-ink-200" />
      <div className="mt-2 h-4 w-full animate-pulse rounded bg-ink-100" />
      <div className="mt-2 h-4 w-5/6 animate-pulse rounded bg-ink-100" />
    </div>
  )
}
