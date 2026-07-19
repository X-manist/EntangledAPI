import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  createAccountMock,
  importCodexSessionMock,
  createOpenAICodexPATMock,
  getGitHubCopilotOAuthCapabilitiesMock,
  startGitHubCopilotOAuthDeviceMock,
  pollGitHubCopilotOAuthDeviceMock,
} = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  importCodexSessionMock: vi.fn(),
  createOpenAICodexPATMock: vi.fn(),
  getGitHubCopilotOAuthCapabilitiesMock: vi.fn(),
  startGitHubCopilotOAuthDeviceMock: vi.fn(),
  pollGitHubCopilotOAuthDeviceMock: vi.fn(),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isSimpleMode: true }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false }),
      importCodexSession: importCodexSessionMock,
      createOpenAICodexPAT: createOpenAICodexPATMock,
      getGitHubCopilotOAuthCapabilities: getGitHubCopilotOAuthCapabilitiesMock,
      startGitHubCopilotOAuthDevice: startGitHubCopilotOAuthDeviceMock,
      pollGitHubCopilotOAuthDevice: pollGitHubCopilotOAuthDeviceMock,
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }),
      getSettings: vi.fn().mockResolvedValue({}),
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([]),
    },
  },
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn().mockResolvedValue([]),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  emits: ['import-codex-session', 'import-codex-pat'],
  template: `
    <div>
      <button data-testid="import-codex-session" @click="$emit('import-codex-session', 'session-json')">session</button>
      <button data-testid="import-codex-pat" @click="$emit('import-codex-pat', 'pat-token')">pat</button>
    </div>
  `,
})

function mountModal() {
  return mount(CreateAccountModal, {
    props: { show: true, proxies: [], groups: [] },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub,
        ConfirmDialog: true,
        Select: true,
        Icon: true,
        PlatformIcon: true,
        ProxySelector: true,
        ProxyAdBanner: true,
        GroupSelector: true,
        ModelWhitelistSelector: true,
        QuotaLimitCard: true,
      },
    },
  })
}

async function selectButtonByText(wrapper: ReturnType<typeof mountModal>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text().includes(text))
  expect(button).toBeDefined()
  await button?.trigger('click')
}

async function submitApiKeyAccount(platform: 'openai' | 'anthropic', enableLongContextBilling = false) {
  const wrapper = mountModal()
  await selectButtonByText(wrapper, platform === 'openai' ? 'OpenAI' : 'admin.accounts.claudeConsole')
  if (platform === 'openai') {
    await selectButtonByText(wrapper, 'API Key')
  }
  await wrapper.get('form#create-account-form input[type="text"]').setValue(`${platform} account`)
  await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
  if (enableLongContextBilling) {
    await wrapper.get('[data-testid="openai-long-context-billing-toggle"]').trigger('click')
  }
  await wrapper.get('form#create-account-form').trigger('submit.prevent')
  await flushPromises()
}

async function openCodexImportStep(toggleClicks = 0) {
  const wrapper = mountModal()
  await selectButtonByText(wrapper, 'OpenAI')
  for (let click = 0; click < toggleClicks; click += 1) {
    await wrapper.get('[data-testid="openai-long-context-billing-toggle"]').trigger('click')
  }
  await wrapper.get('form#create-account-form input[type="text"]').setValue('Codex import')
  await wrapper.get('form#create-account-form').trigger('submit.prevent')
  return wrapper
}

describe('CreateAccountModal OpenAI long-context billing', () => {
  beforeEach(() => {
    createAccountMock.mockReset().mockResolvedValue({})
    importCodexSessionMock.mockReset().mockResolvedValue({
      created: 1,
      updated: 0,
      skipped: 0,
      failed: 0,
      errors: [],
      warnings: [],
    })
    createOpenAICodexPATMock.mockReset().mockResolvedValue({})
  })

  it('sends false explicitly for normal OpenAI account creation by default', async () => {
    await submitApiKeyAccount('openai')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(false)
  })

  it('sends true explicitly when OpenAI long-context billing is enabled', async () => {
    await submitApiKeyAccount('openai', true)

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(true)
  })

  it('omits the OpenAI setting for non-OpenAI account creation', async () => {
    await submitApiKeyAccount('anthropic')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBeUndefined()
  })

  it('leaves Codex session import billing ownership to the backend', async () => {
    const wrapper = await openCodexImportStep()
    await wrapper.get('[data-testid="import-codex-session"]').trigger('click')
    await flushPromises()

    expect(importCodexSessionMock).toHaveBeenCalledTimes(1)
    expect(importCodexSessionMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBeUndefined()
  })

  it('leaves Codex PAT import billing ownership to the backend', async () => {
    const wrapper = await openCodexImportStep()
    await wrapper.get('[data-testid="import-codex-pat"]').trigger('click')
    await flushPromises()

    expect(createOpenAICodexPATMock).toHaveBeenCalledTimes(1)
    expect(createOpenAICodexPATMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBeUndefined()
  })

  it('sends explicit true for Codex session import after the toggle is enabled', async () => {
    const wrapper = await openCodexImportStep(1)
    await wrapper.get('[data-testid="import-codex-session"]').trigger('click')
    await flushPromises()

    expect(importCodexSessionMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(true)
  })

  it('sends explicit false for Codex session import after the toggle is changed back', async () => {
    const wrapper = await openCodexImportStep(2)
    await wrapper.get('[data-testid="import-codex-session"]').trigger('click')
    await flushPromises()

    expect(importCodexSessionMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(false)
  })

  it('sends explicit true for Codex PAT import after the toggle is enabled', async () => {
    const wrapper = await openCodexImportStep(1)
    await wrapper.get('[data-testid="import-codex-pat"]').trigger('click')
    await flushPromises()

    expect(createOpenAICodexPATMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(true)
  })

  it('sends explicit false for Codex PAT import after the toggle is changed back', async () => {
    const wrapper = await openCodexImportStep(2)
    await wrapper.get('[data-testid="import-codex-pat"]').trigger('click')
    await flushPromises()

    expect(createOpenAICodexPATMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(false)
  })
})

describe('CreateAccountModal Coding Plans', () => {
  beforeEach(() => {
    createAccountMock.mockReset().mockResolvedValue({})
    getGitHubCopilotOAuthCapabilitiesMock.mockReset().mockResolvedValue({ configured: true })
    startGitHubCopilotOAuthDeviceMock.mockReset().mockResolvedValue({
      oauth_session_id: 'device-session-id',
      user_code: 'ABCD-EFGH',
      verification_uri: 'https://github.com/login/device',
      expires_in: 900,
      interval: 1,
    })
    pollGitHubCopilotOAuthDeviceMock.mockReset().mockResolvedValue({ status: 'pending' })
  })

  it.each([
    ['glm_coding_plan', 'https://open.bigmodel.cn/api/coding/paas/v4'],
    ['kimi_coding_plan', 'https://api.kimi.com/coding/v1']
  ] as const)('creates %s as a strict OpenAI-compatible API key account', async (provider, baseUrl) => {
    const wrapper = mountModal()
    await wrapper.get(`[data-testid="coding-plan-${provider}"]`).trigger('click')
    await wrapper.get('[data-tour="account-form-name"]').setValue(`${provider} account`)
    await wrapper.get('form#create-account-form input[type="password"]').setValue('coding-token')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    const payload = createAccountMock.mock.calls[0]?.[0]
    expect(payload).toMatchObject({
      platform: 'openai',
      type: 'apikey',
      extra: {
        upstream_provider: provider,
        openai_responses_mode: 'force_chat_completions'
      },
      credentials: {
        api_key: 'coding-token',
        openai_capabilities: ['chat_completions']
      }
    })
    expect(Object.keys(payload.credentials.model_mapping).length).toBeGreaterThan(0)
    expect(payload.credentials.base_url).toBe(baseUrl)
  })

  it('shows GitHub authorization as the recommended default and keeps manual Token as fallback', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="github-copilot-method-oauth"]').classes()).toContain('bg-white')
    expect(wrapper.get('[data-testid="github-copilot-method-oauth"]').text()).toContain(
      'admin.accounts.codingPlans.githubCopilot.recommended'
    )
    expect(wrapper.find('[data-testid="github-copilot-oauth"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="api-key-value"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="github-copilot-method-oauth"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-testid="github-copilot-method-manual"]').attributes('aria-pressed')).toBe('false')
  })

  it('creates GitHub Copilot with a manually entered Token and links to a prefilled fine-grained PAT', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
    await wrapper.get('[data-testid="github-copilot-method-manual"]').trigger('click')
    await wrapper.get('[data-tour="account-form-name"]').setValue('Copilot manual account')
    await wrapper.get('[data-testid="api-key-value"]').setValue('github_pat_manual-secret')

    expect(wrapper.get('[data-testid="github-copilot-create-pat"]').attributes('href')).toBe(
      'https://github.com/settings/personal-access-tokens/new?name=Sub2API%20Copilot&description=Use%20GitHub%20Copilot%20with%20Sub2API&copilot_requests=write'
    )

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    const payload = createAccountMock.mock.calls[0]?.[0]
    expect(payload).toMatchObject({
      platform: 'openai',
      type: 'apikey',
      extra: {
        upstream_provider: 'github_copilot',
        openai_responses_mode: 'force_chat_completions',
      },
      credentials: {
        api_key: 'github_pat_manual-secret',
        openai_capabilities: ['chat_completions'],
      },
    })
    expect(payload.credentials).not.toHaveProperty('github_copilot_oauth_session_id')
    expect(payload.credentials).not.toHaveProperty('base_url')
  })

  it('submits only the opaque OAuth session while retaining preset models and capabilities', async () => {
    vi.useFakeTimers()
    const authorizationWindow = {
      opener: window,
      closed: false,
      close: vi.fn(),
      location: { href: 'about:blank' },
    } as unknown as Window
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(authorizationWindow)
    startGitHubCopilotOAuthDeviceMock.mockResolvedValue({
      oauth_session_id: 'opaque-device-session',
      user_code: 'WXYZ-1234',
      verification_uri: 'https://github.com/login/device',
      expires_in: 900,
      interval: 1,
      access_token: 'must-never-reach-the-browser-ui',
    })
    pollGitHubCopilotOAuthDeviceMock.mockResolvedValue({
      status: 'authorized',
      github_login: 'octocat',
      access_token: 'must-never-reach-the-account-payload',
    })

    try {
      const wrapper = mountModal()
      await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()

      expect(openSpy).toHaveBeenCalledWith(
        'about:blank',
        '_blank'
      )
      expect(authorizationWindow.opener).toBeNull()
      expect(authorizationWindow.location.href).toBe('https://github.com/login/device')
      expect(wrapper.get('[data-testid="github-copilot-device-code"]').text()).toContain('WXYZ-1234')

      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()

      expect(wrapper.get('[data-testid="github-copilot-oauth-authorized"]').text()).toContain(
        'admin.accounts.codingPlans.githubCopilot.oauth.authorizedAs'
      )
      expect(wrapper.text()).not.toContain('must-never-reach')

      await wrapper.get('[data-tour="account-form-name"]').setValue('Copilot OAuth account')
      await wrapper.get('form#create-account-form').trigger('submit.prevent')
      await flushPromises()

      expect(createAccountMock).toHaveBeenCalledTimes(1)
      const payload = createAccountMock.mock.calls[0]?.[0]
      expect(payload).toMatchObject({
        platform: 'openai',
        type: 'apikey',
        extra: {
          upstream_provider: 'github_copilot',
          openai_responses_mode: 'force_chat_completions',
        },
        credentials: {
          github_copilot_oauth_session_id: 'opaque-device-session',
          openai_capabilities: ['chat_completions'],
        },
      })
      expect(payload.credentials).not.toHaveProperty('api_key')
      expect(payload.credentials).not.toHaveProperty('base_url')
      expect(Object.keys(payload.credentials.model_mapping).length).toBeGreaterThan(0)
      expect(JSON.stringify(payload)).not.toContain('must-never-reach')
    } finally {
      openSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('stops Device Flow polling when switching to manual Token', async () => {
    vi.useFakeTimers()
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)

    try {
      const wrapper = mountModal()
      await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()

      await wrapper.get('[data-testid="github-copilot-method-manual"]').trigger('click')
      await vi.advanceTimersByTimeAsync(2000)

      expect(pollGitHubCopilotOAuthDeviceMock).not.toHaveBeenCalled()
      expect(wrapper.find('[data-testid="github-copilot-oauth"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="api-key-value"]').exists()).toBe(true)
    } finally {
      openSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('expires an authorized Device Flow session and offers authorization again', async () => {
    vi.useFakeTimers()
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)
    startGitHubCopilotOAuthDeviceMock.mockResolvedValue({
      oauth_session_id: 'short-session',
      user_code: 'ABCD-EFGH',
      verification_uri: 'https://github.com/login/device',
      expires_in: 2,
      interval: 1,
    })
    pollGitHubCopilotOAuthDeviceMock.mockResolvedValue({
      status: 'authorized',
      github_login: 'octocat',
    })

    try {
      const wrapper = mountModal()
      await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()
      expect(wrapper.find('[data-testid="github-copilot-oauth-authorized"]').exists()).toBe(true)

      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()

      expect(wrapper.find('[data-testid="github-copilot-oauth-authorized"]').exists()).toBe(false)
      expect(wrapper.get('[data-testid="github-copilot-oauth-error"]').text()).toContain(
        'admin.accounts.codingPlans.githubCopilot.oauth.expired'
      )
    } finally {
      openSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('retries a transient entitlement check failure without restarting Device Flow', async () => {
    vi.useFakeTimers()
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)
    pollGitHubCopilotOAuthDeviceMock
      .mockRejectedValueOnce({ reason: 'GITHUB_COPILOT_OAUTH_UNAVAILABLE' })
      .mockResolvedValueOnce({ status: 'authorized', github_login: 'octocat' })

    try {
      const wrapper = mountModal()
      await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()

      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()
      expect(pollGitHubCopilotOAuthDeviceMock).toHaveBeenCalledTimes(1)
      expect(startGitHubCopilotOAuthDeviceMock).toHaveBeenCalledTimes(1)

      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()
      expect(pollGitHubCopilotOAuthDeviceMock).toHaveBeenCalledTimes(2)
      expect(startGitHubCopilotOAuthDeviceMock).toHaveBeenCalledTimes(1)
      expect(wrapper.find('[data-testid="github-copilot-oauth-authorized"]').exists()).toBe(true)
    } finally {
      openSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('clears an expired session returned by account creation and can authorize again', async () => {
    vi.useFakeTimers()
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)
    pollGitHubCopilotOAuthDeviceMock.mockResolvedValue({
      status: 'authorized',
      github_login: 'octocat',
    })
    createAccountMock.mockRejectedValueOnce({
      status: 410,
      reason: 'GITHUB_COPILOT_OAUTH_SESSION_EXPIRED',
    })

    try {
      const wrapper = mountModal()
      await wrapper.get('[data-testid="coding-plan-github_copilot"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()
      await vi.advanceTimersByTimeAsync(1000)
      await flushPromises()
      expect(wrapper.find('[data-testid="github-copilot-oauth-authorized"]').exists()).toBe(true)

      await wrapper.get('[data-tour="account-form-name"]').setValue('Expired Copilot account')
      await wrapper.get('form#create-account-form').trigger('submit.prevent')
      await flushPromises()

      expect(createAccountMock).toHaveBeenCalledTimes(1)
      expect(wrapper.find('[data-testid="github-copilot-oauth-authorized"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="github-copilot-start-oauth"]').exists()).toBe(true)

      await wrapper.get('[data-testid="github-copilot-start-oauth"]').trigger('click')
      await flushPromises()
      expect(startGitHubCopilotOAuthDeviceMock).toHaveBeenCalledTimes(2)
    } finally {
      openSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('clears the subtype when switching back to a normal OpenAI account', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="coding-plan-glm_coding_plan"]').trigger('click')
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    await wrapper.get('[data-tour="account-form-name"]').setValue('Normal OpenAI account')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('openai-key')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    const payload = createAccountMock.mock.calls[0]?.[0]
    expect(payload.extra).not.toHaveProperty('upstream_provider')
    expect(payload.extra).not.toHaveProperty('openai_responses_mode')
  })

  it('clears the shared credential whenever the upstream identity actually changes', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')

    const credentialInput = wrapper.get('[data-testid="api-key-value"]')
    await credentialInput.setValue('openai-secret')

    // Re-selecting the same regular provider is not an identity change.
    await selectButtonByText(wrapper, 'OpenAI')
    expect((credentialInput.element as HTMLInputElement).value).toBe('openai-secret')

    await wrapper.get('[data-testid="coding-plan-glm_coding_plan"]').trigger('click')
    expect((wrapper.get('[data-testid="api-key-value"]').element as HTMLInputElement).value).toBe('')

    await wrapper.get('[data-testid="api-key-value"]').setValue('glm-secret')
    // Re-selecting the same Coding Plan is also a no-op.
    await wrapper.get('[data-testid="coding-plan-glm_coding_plan"]').trigger('click')
    expect((wrapper.get('[data-testid="api-key-value"]').element as HTMLInputElement).value).toBe('glm-secret')

    await wrapper.get('[data-testid="coding-plan-kimi_coding_plan"]').trigger('click')
    expect((wrapper.get('[data-testid="api-key-value"]').element as HTMLInputElement).value).toBe('')

    await wrapper.get('[data-testid="api-key-value"]').setValue('kimi-secret')
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    expect((wrapper.get('[data-testid="api-key-value"]').element as HTMLInputElement).value).toBe('')

    await wrapper.get('[data-testid="api-key-value"]').setValue('next-openai-secret')
    await selectButtonByText(wrapper, 'Anthropic')
    expect((wrapper.get('[data-testid="api-key-value"]').element as HTMLInputElement).value).toBe('')
  })

  it('does not reset a custom Base URL when normal OpenAI is selected again', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')

    const baseUrlInput = wrapper.get('[data-testid="api-key-base-url"]')
    await baseUrlInput.setValue('https://openai-compatible.example/v1')
    await selectButtonByText(wrapper, 'OpenAI')

    expect((baseUrlInput.element as HTMLInputElement).value).toBe(
      'https://openai-compatible.example/v1'
    )
  })
})
