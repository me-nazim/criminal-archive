import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Github, Mail } from 'lucide-react'

import { Logo } from '../brand/Logo'
import { useBranding } from '../../lib/branding'

export default function Footer() {
  const { t, i18n } = useTranslation()
  const { branding } = useBranding()
  const year = new Date().getFullYear()
  const isBengali = i18n.language?.startsWith('bn')

  const tagline = isBengali ? branding.tagline_bn : branding.tagline_en
  const siteName = isBengali ? branding.site_name_bn : branding.site_name_en

  return (
    <footer className="mt-16 border-t border-ink-200 bg-white">
      <div className="container-page py-12">
        <div className="grid gap-10 md:grid-cols-[1.4fr_1fr_1fr_1fr]">
          <div>
            <Logo customSrc={branding.logo_url} bengali={isBengali} />
            <p className="mt-4 max-w-md text-sm leading-relaxed text-ink-600">{tagline}</p>
            <div className="mt-4 flex items-center gap-2">
              {branding.support_email && (
                <a
                  href={`mailto:${branding.support_email}`}
                  className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-ink-200 text-ink-700 hover:bg-ink-100"
                  aria-label="Email support"
                >
                  <Mail className="h-4 w-4" aria-hidden />
                </a>
              )}
              {branding.social?.github && (
                <a
                  href={branding.social.github}
                  rel="noreferrer noopener"
                  target="_blank"
                  className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-ink-200 text-ink-700 hover:bg-ink-100"
                  aria-label="GitHub"
                >
                  <Github className="h-4 w-4" aria-hidden />
                </a>
              )}
            </div>
          </div>

          <FooterColumn title={t('footer.explore')}>
            <FooterLink to="/cases">{t('nav.cases')}</FooterLink>
            <FooterLink to="/persons">{t('nav.persons')}</FooterLink>
            <FooterLink to="/search">{t('nav.search')}</FooterLink>
          </FooterColumn>

          <FooterColumn title={t('footer.contribute')}>
            <FooterLink to="/register">{t('nav.register')}</FooterLink>
            <FooterLink to="/login">{t('nav.login')}</FooterLink>
            <FooterLink to="/me/cases/new">{t('home.submit_info')}</FooterLink>
          </FooterColumn>

          <FooterColumn title={t('footer.about')}>
            <FooterLink to="/about" anchor>{t('nav.about')}</FooterLink>
            <FooterLink to="/privacy" anchor>{t('footer.privacy')}</FooterLink>
            <FooterLink to="/terms" anchor>{t('footer.terms')}</FooterLink>
          </FooterColumn>
        </div>

        <div className="mt-10 flex flex-col items-start justify-between gap-3 border-t border-ink-200 pt-6 text-xs text-ink-500 sm:flex-row sm:items-center">
          <p>
            © {year} {siteName}. {t('footer.rights')}
          </p>
          <p>
            {t('footer.built_with')}{' '}
            <span className="font-medium text-ink-700">React · Tailwind · Go</span>
          </p>
        </div>
      </div>
    </footer>
  )
}

function FooterColumn({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h4 className="mb-3 text-xs font-semibold uppercase tracking-[0.14em] text-ink-500">{title}</h4>
      <ul className="space-y-2 text-sm">{children}</ul>
    </div>
  )
}

function FooterLink({
  to,
  children,
  anchor,
}: {
  to: string
  children: React.ReactNode
  anchor?: boolean
}) {
  if (anchor) {
    return (
      <li>
        <a href={to} className="link-quiet">
          {children}
        </a>
      </li>
    )
  }
  return (
    <li>
      <Link to={to} className="link-quiet">
        {children}
      </Link>
    </li>
  )
}
