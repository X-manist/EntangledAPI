import { computed, ref } from 'vue'
import { paymentAPI } from '@/api/payment'

const DEFAULT_BALANCE_RECHARGE_MULTIPLIER = 1
const USER_CURRENCY_SYMBOL = '¥'

const balanceRechargeMultiplier = ref(DEFAULT_BALANCE_RECHARGE_MULTIPLIER)
const configLoaded = ref(false)
let pendingLoad: Promise<number> | null = null

export function normalizeUserCurrencyMultiplier(value: unknown): number {
  const numeric = Number(value)
  return Number.isFinite(numeric) && numeric > 0
    ? numeric
    : DEFAULT_BALANCE_RECHARGE_MULTIPLIER
}

export function setUserCurrencyMultiplier(value: unknown): number {
  const normalized = normalizeUserCurrencyMultiplier(value)
  balanceRechargeMultiplier.value = normalized
  configLoaded.value = true
  return normalized
}

export async function loadUserCurrencyConfig(force = false): Promise<number> {
  if (configLoaded.value && !force) {
    return balanceRechargeMultiplier.value
  }
  if (pendingLoad && !force) {
    return pendingLoad
  }

  const getConfig = paymentAPI.getConfig
  if (typeof getConfig !== 'function') {
    return balanceRechargeMultiplier.value
  }

  pendingLoad = getConfig()
    .then((response) => setUserCurrencyMultiplier(response.data?.balance_recharge_multiplier))
    .catch(() => balanceRechargeMultiplier.value)
    .finally(() => {
      pendingLoad = null
    })

  return pendingLoad
}

export function backendAmountToUserAmount(
  value: number | null | undefined,
  multiplier = balanceRechargeMultiplier.value
): number {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return 0
  return numeric / normalizeUserCurrencyMultiplier(multiplier)
}

export function userAmountToBackendAmount(
  value: number | null | undefined,
  multiplier = balanceRechargeMultiplier.value
): number {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return 0
  return numeric * normalizeUserCurrencyMultiplier(multiplier)
}

export function formatUserAmount(value: number | null | undefined, fractionDigits = 2): string {
  const numeric = Number(value)
  const safeValue = Number.isFinite(numeric) ? numeric : 0
  return `${USER_CURRENCY_SYMBOL}${safeValue.toFixed(fractionDigits)}`
}

export function formatBackendUserAmount(
  value: number | null | undefined,
  fractionDigits = 2,
  multiplier = balanceRechargeMultiplier.value
): string {
  return formatUserAmount(backendAmountToUserAmount(value, multiplier), fractionDigits)
}

export function useUserCurrency() {
  const multiplier = computed(() => balanceRechargeMultiplier.value)

  return {
    userCurrencySymbol: USER_CURRENCY_SYMBOL,
    balanceRechargeMultiplier: multiplier,
    loadUserCurrencyConfig,
    setUserCurrencyMultiplier,
    backendAmountToUserAmount: (value: number | null | undefined) =>
      backendAmountToUserAmount(value, multiplier.value),
    userAmountToBackendAmount: (value: number | null | undefined) =>
      userAmountToBackendAmount(value, multiplier.value),
    formatUserAmount,
    formatBackendUserAmount: (value: number | null | undefined, fractionDigits = 2) =>
      formatBackendUserAmount(value, fractionDigits, multiplier.value),
  }
}
