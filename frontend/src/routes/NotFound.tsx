import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function NotFound() {
  const { t } = useTranslation()
  return (
    <section className="mx-auto max-w-xl px-4 py-24 text-center">
      <p className="font-display text-6xl font-bold text-ink-900">404</p>
      <h1 className="mt-2 text-xl font-semibold text-ink-800">{t('notfound.title')}</h1>
      <Link
        to="/"
        className="mt-6 inline-block rounded-md bg-ink-900 px-4 py-2 text-sm font-semibold text-white hover:bg-ink-800"
      >
        {t('notfound.back_home')}
      </Link>
    </section>
  )
}
