import { mount } from '@vue/test-utils'
import { describe, expect, test } from 'vitest'
import ConsoleTopbar from './ConsoleTopbar.vue'

describe('ConsoleTopbar locale controls', () => {
  test('renders language toggle controls alongside the desktop action rail', () => {
    const wrapper = mount(ConsoleTopbar, {
      props: {
        healthStatus: 'ok',
        versionState: { version: 'v0.7.1', requestId: 'req-1' },
        isChecking: false,
      },
      global: {
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('EN')
    expect(wrapper.text()).toContain('中文')
    expect(wrapper.find('.console-topbar-actions').exists()).toBe(true)
  })
})
