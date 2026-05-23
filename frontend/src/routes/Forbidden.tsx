import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '../components/ui/Container'

export default function Forbidden() {
  const { t } = useTranslation()
  return (
    <Container width="narrow" className="py-24 text-center">
      <p className="font-display text-6xl font-bold text-ink-900">403</p>
      <h1 className="mt-2 text-xl font-semibold text-ink-800">{t('forbidden.title')}</h1>
      <p className="mt-2 text-sm text-ink-600">{t('forbidden.message')}</p>
      <Link
        to="/"
        className="mt-6 inline-block rounded-md bg-ink-900 px-4 py-2 text-sm font-semibold text-white hover:bg-ink-800"
      >
        {t('notfound.back_home')}
      </Link>
    </Container>
  )
}
