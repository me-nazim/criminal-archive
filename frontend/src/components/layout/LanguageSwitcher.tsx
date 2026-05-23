import { useTranslation } from 'react-i18next'

const LANGS = [
  { code: 'bn', label: 'বাংলা' },
  { code: 'en', label: 'EN' },
] as const

export default function LanguageSwitcher() {
  const { i18n } = useTranslation()
  const current = i18n.resolvedLanguage ?? 'bn'

  return (
    <div className="inline-flex overflow-hidden rounded-md border border-ink-200 text-xs">
      {LANGS.map((l) => {
        const active = current.startsWith(l.code)
        return (
          <button
            key={l.code}
            type="button"
            onClick={() => i18n.changeLanguage(l.code)}
            className={[
              'px-2.5 py-1 font-medium transition-colors',
              active ? 'bg-ink-900 text-white' : 'text-ink-700 hover:bg-ink-100',
            ].join(' ')}
            aria-pressed={active}
          >
            {l.label}
          </button>
        )
      })}
    </div>
  )
}
