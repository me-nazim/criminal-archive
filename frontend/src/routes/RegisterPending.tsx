import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CheckCircle2 } from 'lucide-react'
import { Card, CardBody } from '../components/ui/Card'
import { Container } from '../components/ui/Container'

export default function RegisterPending() {
  const { t } = useTranslation()
  return (
    <Container width="narrow" className="py-16">
      <Card>
        <CardBody className="flex flex-col items-center gap-4 py-12 text-center">
          <CheckCircle2 className="h-10 w-10 text-green-600" aria-hidden />
          <h1 className="font-display text-xl font-semibold text-ink-900">
            {t('register_pending.title')}
          </h1>
          <p className="max-w-prose text-sm text-ink-600">{t('register_pending.message')}</p>
          <Link
            to="/"
            className="mt-2 rounded-md bg-ink-900 px-4 py-2 text-sm font-medium text-white hover:bg-ink-800"
          >
            {t('register_pending.home_link')}
          </Link>
        </CardBody>
      </Card>
    </Container>
  )
}
