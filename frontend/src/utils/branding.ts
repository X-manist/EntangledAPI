export const BRAND_NAME_EN = 'EntanglementAI'
export const BRAND_NAME_ZH = '纠缠智能'
export const LEGACY_BRAND_NAME = 'Sub2API'

const DEFAULT_BRAND_NAMES = new Set([
  BRAND_NAME_EN.toLowerCase(),
  BRAND_NAME_ZH.toLowerCase(),
  LEGACY_BRAND_NAME.toLowerCase(),
])

export function defaultBrandName(locale?: string | null): string {
  return typeof locale === 'string' && locale.toLowerCase().startsWith('zh')
    ? BRAND_NAME_ZH
    : BRAND_NAME_EN
}

export function resolveDisplayBrandName(
  configuredName?: string | null,
  locale?: string | null
): string {
  const trimmed = typeof configuredName === 'string' ? configuredName.trim() : ''
  if (!trimmed || DEFAULT_BRAND_NAMES.has(trimmed.toLowerCase())) {
    return defaultBrandName(locale)
  }
  return trimmed
}
