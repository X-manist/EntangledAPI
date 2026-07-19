import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

import PlatformTypeBadge from '../PlatformTypeBadge.vue'

describe('PlatformTypeBadge', () => {
  it.each([
    ['glm_coding_plan', 'GLM Coding Plan'],
    ['kimi_coding_plan', 'Kimi Coding Plan'],
    ['github_copilot', 'GitHub Copilot']
  ] as const)('shows the %s brand instead of OpenAI', (upstreamProvider, label) => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'apikey',
        upstreamProvider
      },
      global: {
        stubs: {
          PlatformIcon: true,
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain(label)
    expect(wrapper.text()).not.toContain('OpenAI')
  })
})
