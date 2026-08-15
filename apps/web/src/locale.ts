export type InterfaceLocale = 'zh-CN' | 'en'
export const INTERFACE_LOCALE_KEY = 'vrc-harbor-locale'

export function readInterfaceLocale(storage: Pick<Storage, 'getItem'> = localStorage): InterfaceLocale {
  return storage.getItem(INTERFACE_LOCALE_KEY) === 'en' ? 'en' : 'zh-CN'
}

export function applyInterfaceLocale(locale: InterfaceLocale, persist = true, storage: Pick<Storage, 'setItem'> = localStorage) {
  document.documentElement.lang = locale
  document.documentElement.dataset.locale = locale
  if (persist) storage.setItem(INTERFACE_LOCALE_KEY, locale)
  return locale
}

export function interfaceText(locale: InterfaceLocale, chinese: string, english: string) {
  return locale === 'en' ? english : chinese
}
