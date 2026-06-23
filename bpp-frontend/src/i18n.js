import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import id from './locales/id.json'

const savedLang = localStorage.getItem('bpp_lang') || 'id'

i18n.use(initReactI18next).init({
  resources: { en: { translation: en }, id: { translation: id } },
  lng: savedLang,
  fallbackLng: 'en',
  interpolation: { escapeValue: false }
})

export default i18n
