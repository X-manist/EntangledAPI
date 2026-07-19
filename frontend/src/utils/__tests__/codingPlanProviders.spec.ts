import { describe, expect, it } from 'vitest'

import {
  buildCodingPlanCredentials,
  getCodingPlanDisplayModels,
  getCodingPlanProviderPreset,
  resolveCodingPlanModelMapping
} from '../codingPlanProviders'

describe('codingPlanProviders', () => {
  it('uses the current strict GLM Coding Plan aliases', () => {
    const preset = getCodingPlanProviderPreset('glm_coding_plan')!

    expect(preset.modelMapping).toEqual({
      'glm-5.2': 'GLM-5.2',
      'GLM-5.2': 'GLM-5.2',
      'glm-5-turbo': 'GLM-5-Turbo',
      'GLM-5-Turbo': 'GLM-5-Turbo',
      'glm-4.7': 'GLM-4.7',
      'GLM-4.7': 'GLM-4.7'
    })
    expect(getCodingPlanDisplayModels(preset)).toEqual(['GLM-5.2', 'GLM-5-Turbo', 'GLM-4.7'])
  })

  it('never sends a client-controlled base URL for GitHub Copilot', () => {
    const preset = getCodingPlanProviderPreset('github_copilot')!

    expect(preset.modelMapping['claude-sonnet-4-6']).toBe('claude-sonnet-4.6')
    expect(preset.modelMapping['claude-haiku-4-5']).toBe('claude-haiku-4.5')
    expect(buildCodingPlanCredentials(preset, 'github-token')).toEqual({
      api_key: 'github-token',
      model_mapping: preset.modelMapping,
      openai_capabilities: ['chat_completions']
    })
  })

  it('keeps a non-empty synced mapping and only falls back when it is missing', () => {
    const preset = getCodingPlanProviderPreset('github_copilot')!
    const synced = { 'new-model': 'new-model-2026' }

    expect(resolveCodingPlanModelMapping(preset, synced)).toEqual(synced)
    expect(getCodingPlanDisplayModels(preset, synced)).toEqual(['new-model-2026'])
    expect(resolveCodingPlanModelMapping(preset, {})).toEqual(preset.modelMapping)
  })
})
