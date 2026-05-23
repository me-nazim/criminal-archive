import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import bn from '../locales/bn.json'
import en from '../locales/en.json'

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      bn: { translation: bn },
      en: { translation: en },
    },
    fallbackLng: 'bn',
    supportedLngs: ['bn', 'en'],
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'htmlTag', 'navigator'],
      caches: ['localStorage'],
    },
  })

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

export default i18n
