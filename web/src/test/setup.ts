import { afterEach, beforeEach, vi } from 'vitest'

const originalNavigator = globalThis.navigator

function defineNavigator(language: string, languages: string[]) {
  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: {
      ...originalNavigator,
      language,
      languages,
    },
  })
}

beforeEach(() => {
  localStorage.clear()
  defineNavigator('en-US', ['en-US'])
})

afterEach(async () => {
  localStorage.clear()
  vi.restoreAllMocks()
  Object.defineProperty(globalThis, 'navigator', {
    configurable: true,
    value: originalNavigator,
  })
  const localeModule = await import('../lib/locale')
  localeModule.resetLocaleForTests()
})

export function mockBrowserLanguage(language: string, languages: string[] = [language]) {
  defineNavigator(language, languages)
}
