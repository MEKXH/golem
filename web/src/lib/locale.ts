import { computed, readonly, ref } from 'vue'
import { localeStorageKey, messages, type Locale } from './messages'

const activeLocale = ref<Locale>('en')
let initialized = false

function normalizeLocale(value: string | null | undefined): Locale | null {
  const normalized = (value ?? '').trim().toLowerCase()
  if (normalized === '') {
    return null
  }
  if (normalized.startsWith('zh')) {
    return 'zh-CN'
  }
  if (normalized.startsWith('en')) {
    return 'en'
  }
  return null
}

function readStoredLocale(): Locale | null {
  if (typeof window === 'undefined') {
    return null
  }
  return normalizeLocale(window.localStorage.getItem(localeStorageKey))
}

function detectBrowserLocale(): Locale {
  if (typeof navigator === 'undefined') {
    return 'en'
  }
  const candidates = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const candidate of candidates) {
    const locale = normalizeLocale(candidate)
    if (locale !== null) {
      return locale
    }
  }
  return 'en'
}

export function resolvePreferredLocale(): Locale {
  return readStoredLocale() ?? detectBrowserLocale()
}

function ensureInitialized() {
  if (initialized) {
    return
  }
  activeLocale.value = resolvePreferredLocale()
  initialized = true
}

export function setLocale(nextLocale: Locale) {
  activeLocale.value = nextLocale
  initialized = true
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(localeStorageKey, nextLocale)
  }
}

export function useLocale() {
  ensureInitialized()

  const copy = computed(() => messages[activeLocale.value])

  return {
    locale: readonly(activeLocale),
    copy,
    setLocale,
  }
}

export function resetLocaleForTests() {
  initialized = false
  activeLocale.value = 'en'
}
