import { useTranslation } from 'react-i18next'

export default function Footer() {
  const { t } = useTranslation()
  const year = new Date().getFullYear()

  return (
    <footer className="border-t border-ink-200 bg-white">
      <div className="mx-auto max-w-7xl px-4 py-8 text-sm text-ink-600 sm:px-6 lg:px-8">
        <p>
          © {year} {t('site.name')}. {t('footer.rights')}
        </p>
      </div>
    </footer>
  )
}
