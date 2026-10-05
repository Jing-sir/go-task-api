import { createI18n } from 'vue-i18n'
import { messages } from './messages'

export const LOCALE_KEY = 'go-task-web.locale'

function readLocale() {
  if (typeof window === 'undefined') {
    return 'zh-CN'
  }

  const stored = window.localStorage.getItem(LOCALE_KEY)
  if (stored === 'en-US' || stored === 'zh-CN') {
    return stored
  }

  return 'zh-CN'
}

export const i18n = createI18n({
  legacy: false,
  locale: readLocale(),
  fallbackLocale: 'zh-CN',
  messages,
})

