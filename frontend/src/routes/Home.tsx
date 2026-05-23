import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function Home() {
  const { t } = useTranslation()

  return (
    <section className="mx-auto max-w-5xl px-4 py-20 text-center sm:px-6 lg:px-8">
      <h1 className="font-display text-4xl font-bold leading-tight text-ink-900 sm:text-5xl">
        {t('home.hero_title')}
      </h1>
      <p className="mx-auto mt-6 max-w-2xl text-lg text-ink-600">
        {t('home.hero_subtitle')}
      </p>
      <div className="mt-10 flex flex-wrap items-center justify-center gap-3">
        <Link
          to="/cases"
          className="rounded-md bg-ink-900 px-5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-ink-800"
        >
          {t('home.browse_cases')}
        </Link>
        <Link
          to="/submit"
          className="rounded-md border border-ink-300 bg-white px-5 py-2.5 text-sm font-semibold text-ink-800 hover:bg-ink-100"
        >
          {t('home.submit_info')}
        </Link>
      </div>
    </section>
  )
}
