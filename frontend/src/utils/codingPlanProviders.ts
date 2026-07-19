import type { CodingPlanUpstreamProvider } from '@/types'

export interface CodingPlanProviderPreset {
  id: CodingPlanUpstreamProvider
  label: string
  baseUrl?: string
  credentialKind: 'api_key' | 'github_token'
  modelMapping: Record<string, string>
}

const CODING_PLAN_PROVIDER_PRESETS: Record<
  CodingPlanUpstreamProvider,
  CodingPlanProviderPreset
> = {
  glm_coding_plan: {
    id: 'glm_coding_plan',
    label: 'GLM Coding Plan',
    baseUrl: 'https://open.bigmodel.cn/api/coding/paas/v4',
    credentialKind: 'api_key',
    modelMapping: {
      'glm-5.2': 'GLM-5.2',
      'GLM-5.2': 'GLM-5.2',
      'glm-5-turbo': 'GLM-5-Turbo',
      'GLM-5-Turbo': 'GLM-5-Turbo',
      'glm-4.7': 'GLM-4.7',
      'GLM-4.7': 'GLM-4.7'
    }
  },
  kimi_coding_plan: {
    id: 'kimi_coding_plan',
    label: 'Kimi Coding Plan',
    baseUrl: 'https://api.kimi.com/coding/v1',
    credentialKind: 'api_key',
    modelMapping: {
      k3: 'k3',
      'kimi-for-coding': 'kimi-for-coding',
      'kimi-for-coding-highspeed': 'kimi-for-coding-highspeed'
    }
  },
  github_copilot: {
    id: 'github_copilot',
    label: 'GitHub Copilot',
    credentialKind: 'github_token',
    modelMapping: {
      'claude-sonnet-4.6': 'claude-sonnet-4.6',
      'claude-sonnet-4-6': 'claude-sonnet-4.6',
      'claude-haiku-4.5': 'claude-haiku-4.5',
      'claude-haiku-4-5': 'claude-haiku-4.5',
      'gpt-5.4': 'gpt-5.4',
      'gpt-5.3-codex': 'gpt-5.3-codex',
      'gemini-3.1-pro-preview': 'gemini-3.1-pro-preview',
      'gemini-3.5-flash': 'gemini-3.5-flash',
      'mai-code-1-flash': 'mai-code-1-flash'
    }
  }
}

export const codingPlanProviderPresets = Object.values(CODING_PLAN_PROVIDER_PRESETS)

export function getCodingPlanProviderPreset(
  provider: unknown
): CodingPlanProviderPreset | undefined {
  if (typeof provider !== 'string') return undefined
  return CODING_PLAN_PROVIDER_PRESETS[provider as CodingPlanUpstreamProvider]
}

export function buildCodingPlanCredentials(
  preset: CodingPlanProviderPreset,
  credential: string
): Record<string, unknown> {
  const credentials: Record<string, unknown> = {
    api_key: credential,
    model_mapping: { ...preset.modelMapping },
    openai_capabilities: ['chat_completions']
  }
  if (preset.baseUrl) {
    credentials.base_url = preset.baseUrl
  }
  return credentials
}

export function resolveCodingPlanModelMapping(
  preset: CodingPlanProviderPreset,
  currentMapping?: unknown
): Record<string, string> {
  if (currentMapping && typeof currentMapping === 'object' && !Array.isArray(currentMapping)) {
    const entries = Object.entries(currentMapping).filter(
      ([from, to]) => from.trim().length > 0 && typeof to === 'string' && to.trim().length > 0
    ) as Array<[string, string]>
    if (entries.length > 0) {
      return Object.fromEntries(entries)
    }
  }
  return { ...preset.modelMapping }
}

export function getCodingPlanDisplayModels(
  preset: CodingPlanProviderPreset,
  currentMapping?: unknown
): string[] {
  return [...new Set(Object.values(resolveCodingPlanModelMapping(preset, currentMapping)))]
}
