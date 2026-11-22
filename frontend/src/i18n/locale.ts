export const LOCALES = { en: 'English', ru: 'Русский' } as const
export type Locale = keyof typeof LOCALES

export function detectLocale(): Locale {
  const lang = navigator.language.slice(0, 2)
  return lang in LOCALES ? (lang as Locale) : 'en'
}
