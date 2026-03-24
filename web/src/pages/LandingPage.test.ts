import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import LandingPage from './LandingPage.vue'
import { mockBrowserLanguage } from '../test/setup'

describe('LandingPage locale behavior', () => {
  test('renders Simplified Chinese copy when browser language is Chinese', () => {
    mockBrowserLanguage('zh-CN', ['zh-CN', 'en-US'])

    const wrapper = mount(LandingPage, {
      global: {
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('终端原生')
    expect(wrapper.text()).toContain('进入控制台')
  })

  test('prefers a persisted Chinese override over an English browser locale', () => {
    localStorage.setItem('golem-web-ui-locale', 'zh-CN')
    mockBrowserLanguage('en-US', ['en-US'])

    const wrapper = mount(LandingPage, {
      global: {
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('Geo 垂直化与自动进化')
  })
})
