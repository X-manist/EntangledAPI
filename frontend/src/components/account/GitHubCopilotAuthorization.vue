<template>
  <div
    class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700/40"
    data-testid="github-copilot-oauth"
  >
    <div class="flex items-start gap-3">
      <div class="rounded-full bg-primary-100 p-2 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
        <Icon name="link" size="sm" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.codingPlans.githubCopilot.oauth.title') }}
        </p>
        <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-400">
          {{ t('admin.accounts.codingPlans.githubCopilot.oauth.description') }}
        </p>
      </div>
    </div>

    <div
      v-if="capabilitiesLoading"
      class="mt-4 flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400"
      role="status"
      aria-live="polite"
    >
      <span class="h-4 w-4 animate-spin rounded-full border-2 border-gray-300 border-t-primary-500" />
      {{ t('admin.accounts.codingPlans.githubCopilot.oauth.checking') }}
    </div>

    <div
      v-else-if="configured === false"
      class="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800/60 dark:bg-amber-900/20"
      data-testid="github-copilot-oauth-not-configured"
      role="alert"
    >
      <p class="text-sm text-amber-800 dark:text-amber-300">
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.notConfigured') }}
      </p>
      <button
        type="button"
        class="mt-2 text-sm font-medium text-amber-800 underline hover:no-underline dark:text-amber-300"
        data-testid="github-copilot-use-manual"
        @click="emit('use-manual')"
      >
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.useManual') }}
      </button>
    </div>

    <div
      v-if="errorMessage"
      class="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-900/20 dark:text-red-300"
      data-testid="github-copilot-oauth-error"
      role="alert"
    >
      <p>{{ errorMessage }}</p>
      <button
        type="button"
        class="mt-2 font-medium underline hover:no-underline"
        @click="startAuthorization"
      >
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.tryAgain') }}
      </button>
      <span class="mx-2">·</span>
      <button
        type="button"
        class="font-medium underline hover:no-underline"
        @click="emit('use-manual')"
      >
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.useManual') }}
      </button>
    </div>

    <div v-if="configured && !userCode && !authorizedSessionId && !errorMessage" class="mt-4">
      <button
        type="button"
        class="btn btn-primary w-full sm:w-auto"
        :disabled="starting"
        data-testid="github-copilot-start-oauth"
        @click="startAuthorization"
      >
        <span
          v-if="starting"
          class="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
        />
        <Icon v-else name="externalLink" size="sm" class="mr-2" />
        {{
          starting
            ? t('admin.accounts.codingPlans.githubCopilot.oauth.starting')
            : t('admin.accounts.codingPlans.githubCopilot.oauth.start')
        }}
      </button>
    </div>

    <div
      v-if="userCode && !authorizedSessionId && !errorMessage"
      class="mt-4 space-y-3"
      data-testid="github-copilot-device-code"
    >
      <p class="text-sm text-gray-700 dark:text-gray-300">
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.enterCode') }}
      </p>
      <div class="flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center justify-between rounded-lg border border-gray-300 bg-white px-4 py-3 font-mono text-xl font-semibold text-gray-900 hover:border-primary-400 dark:border-dark-500 dark:bg-dark-700 dark:text-white"
          data-testid="github-copilot-copy-code"
          :aria-label="t('admin.accounts.codingPlans.githubCopilot.oauth.copyCode')"
          @click="copyToClipboard(userCode)"
        >
          <span class="truncate">{{ userCode }}</span>
          <Icon :name="copied ? 'check' : 'copy'" size="sm" class="ml-3 shrink-0" />
        </button>
        <button
          type="button"
          class="btn btn-secondary shrink-0"
          data-testid="github-copilot-open-verification"
          @click="openVerificationPage"
        >
          <Icon name="externalLink" size="sm" class="mr-2" />
          {{ t('admin.accounts.codingPlans.githubCopilot.oauth.openGitHub') }}
        </button>
      </div>
      <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400" role="status" aria-live="polite">
        <span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-gray-300 border-t-primary-500" />
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.waiting') }}
      </div>
    </div>

    <div
      v-if="authorizedSessionId"
      class="mt-4 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800/60 dark:bg-green-900/20"
      data-testid="github-copilot-oauth-authorized"
      role="status"
      aria-live="polite"
    >
      <div class="flex items-center gap-2 text-sm font-medium text-green-800 dark:text-green-300">
        <Icon name="checkCircle" size="sm" />
        <span>
          {{
            githubLogin
              ? t('admin.accounts.codingPlans.githubCopilot.oauth.authorizedAs', { login: githubLogin })
              : t('admin.accounts.codingPlans.githubCopilot.oauth.authorized')
          }}
        </span>
      </div>
      <button
        type="button"
        class="mt-2 text-sm font-medium text-green-800 underline hover:no-underline dark:text-green-300"
        data-testid="github-copilot-reauthorize"
        @click="startAuthorization"
      >
        {{ t('admin.accounts.codingPlans.githubCopilot.oauth.reauthorize') }}
      </button>
    </div>

    <p class="mt-3 flex items-start gap-1.5 text-xs leading-5 text-gray-500 dark:text-gray-400">
      <Icon name="shield" size="sm" class="mt-0.5 shrink-0" />
      {{ t('admin.accounts.codingPlans.githubCopilot.oauth.tokenKeptOnServer') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useClipboard } from '@/composables/useClipboard'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  proxyId?: number | null
}>()

const emit = defineEmits<{
  'update:session-id': [value: string]
  authorized: [githubLogin?: string]
  'use-manual': []
}>()

const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const capabilitiesLoading = ref(true)
const configured = ref<boolean | null>(null)
const starting = ref(false)
const polling = ref(false)
const errorMessage = ref('')
const deviceSessionId = ref('')
const authorizedSessionId = ref('')
const userCode = ref('')
const verificationUri = ref('')
const githubLogin = ref('')

let pollTimer: ReturnType<typeof setTimeout> | undefined
let expiryTimer: ReturnType<typeof setTimeout> | undefined
let activeRequest: AbortController | undefined
let pollIntervalMs = 5000
let expiresAt = 0
let generation = 0

const errorText = (error: unknown, fallbackKey: string) => {
  const apiError = error as {
    reason?: string
    code?: string
    message?: string
  }
  const reason = apiError.reason || apiError.code || ''
  const reasonMessages: Record<string, string> = {
    GITHUB_COPILOT_OAUTH_NOT_CONFIGURED:
      'admin.accounts.codingPlans.githubCopilot.oauth.notConfigured',
    GITHUB_COPILOT_OAUTH_SESSION_NOT_FOUND:
      'admin.accounts.codingPlans.githubCopilot.oauth.expired',
    GITHUB_COPILOT_OAUTH_SESSION_EXPIRED:
      'admin.accounts.codingPlans.githubCopilot.oauth.expired',
    GITHUB_COPILOT_OAUTH_ENTITLEMENT_REQUIRED:
      'admin.accounts.codingPlans.githubCopilot.oauth.entitlementRequired'
  }
  const messageKey = reasonMessages[reason]
  if (messageKey) return t(messageKey)
  return t(fallbackKey)
}

const oauthErrorReason = (error: unknown) => {
  const apiError = error as { reason?: string; code?: string }
  return apiError.reason || apiError.code || ''
}

const stopPolling = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = undefined
  }
  activeRequest?.abort()
  activeRequest = undefined
  polling.value = false
}

const stopExpiryTimer = () => {
  if (expiryTimer) {
    clearTimeout(expiryTimer)
    expiryTimer = undefined
  }
}

const clearAuthorization = () => {
  generation += 1
  stopPolling()
  stopExpiryTimer()
  starting.value = false
  errorMessage.value = ''
  deviceSessionId.value = ''
  authorizedSessionId.value = ''
  userCode.value = ''
  verificationUri.value = ''
  githubLogin.value = ''
  expiresAt = 0
  emit('update:session-id', '')
}

const loadCapabilities = async () => {
  capabilitiesLoading.value = true
  const request = new AbortController()
  activeRequest = request
  try {
    const result = await adminAPI.accounts.getGitHubCopilotOAuthCapabilities({
      signal: request.signal
    })
    if (request.signal.aborted) return
    configured.value = result.configured
  } catch (error) {
    if (request.signal.aborted) return
    configured.value = null
    errorMessage.value = errorText(
      error,
      'admin.accounts.codingPlans.githubCopilot.oauth.capabilitiesFailed'
    )
  } finally {
    if (activeRequest === request) activeRequest = undefined
    if (!request.signal.aborted) capabilitiesLoading.value = false
  }
}

const openVerificationPage = () => {
  if (!verificationUri.value) return
  window.open(verificationUri.value, '_blank', 'noopener,noreferrer')
}

const safeGitHubVerificationUri = (value: string) => {
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && url.hostname === 'github.com' ? url.toString() : ''
  } catch {
    return ''
  }
}

const schedulePoll = (currentGeneration: number, delay = pollIntervalMs) => {
  if (currentGeneration !== generation || !deviceSessionId.value) return
  polling.value = true
  pollTimer = setTimeout(() => void pollAuthorization(currentGeneration), delay)
}

const scheduleAuthorizedExpiry = (currentGeneration: number) => {
  stopExpiryTimer()
  const delay = expiresAt - Date.now()
  if (delay <= 0) {
    clearAuthorization()
    errorMessage.value = t('admin.accounts.codingPlans.githubCopilot.oauth.expired')
    return
  }
  expiryTimer = setTimeout(() => {
    if (currentGeneration !== generation) return
    clearAuthorization()
    errorMessage.value = t('admin.accounts.codingPlans.githubCopilot.oauth.expired')
  }, delay)
}

const pollAuthorization = async (currentGeneration: number) => {
  if (currentGeneration !== generation || !deviceSessionId.value) return
  if (expiresAt && Date.now() >= expiresAt) {
    stopPolling()
    errorMessage.value = t('admin.accounts.codingPlans.githubCopilot.oauth.expired')
    return
  }

  const request = new AbortController()
  activeRequest = request
  try {
    const result = await adminAPI.accounts.pollGitHubCopilotOAuthDevice(
      deviceSessionId.value,
      { signal: request.signal }
    )
    if (request.signal.aborted || currentGeneration !== generation) return

    if (result.status === 'authorized') {
      stopPolling()
      const sessionId = result.oauth_session_id || deviceSessionId.value
      authorizedSessionId.value = sessionId
      githubLogin.value = result.github_login || ''
      emit('update:session-id', sessionId)
      emit('authorized', githubLogin.value || undefined)
      scheduleAuthorizedExpiry(currentGeneration)
      return
    }

    if (result.status === 'pending') {
      if (result.interval) pollIntervalMs = Math.max(1, result.interval) * 1000
      schedulePoll(currentGeneration)
      return
    }

    if (result.status === 'slow_down') {
      pollIntervalMs = result.interval
        ? Math.max(1, result.interval) * 1000
        : pollIntervalMs + 5000
      schedulePoll(currentGeneration)
      return
    }

    stopPolling()
    if (result.status === 'expired') {
      errorMessage.value = result.message || t('admin.accounts.codingPlans.githubCopilot.oauth.expired')
    } else if (result.status === 'denied') {
      errorMessage.value = result.message || t('admin.accounts.codingPlans.githubCopilot.oauth.denied')
    } else {
      errorMessage.value = result.message || t('admin.accounts.codingPlans.githubCopilot.oauth.failed')
    }
  } catch (error) {
    if (request.signal.aborted || currentGeneration !== generation) return
    const reason = oauthErrorReason(error)
    const terminal = reason === 'GITHUB_COPILOT_OAUTH_ENTITLEMENT_REQUIRED'
      || reason === 'GITHUB_COPILOT_OAUTH_SESSION_EXPIRED'
      || reason === 'GITHUB_COPILOT_OAUTH_SESSION_NOT_FOUND'
    if (terminal) {
      stopPolling()
      errorMessage.value = errorText(error, 'admin.accounts.codingPlans.githubCopilot.oauth.failed')
    } else {
      schedulePoll(currentGeneration)
    }
  } finally {
    if (activeRequest === request) activeRequest = undefined
  }
}

const startAuthorization = async () => {
  // Reserve a tab synchronously while this function still has the user's click
  // activation. Browsers commonly block window.open after the API await below.
  const authorizationWindow = window.open('about:blank', '_blank')
  if (authorizationWindow) authorizationWindow.opener = null
  clearAuthorization()
  const currentGeneration = generation
  starting.value = true
  const request = new AbortController()
  activeRequest = request
  try {
    const result = await adminAPI.accounts.startGitHubCopilotOAuthDevice(
      { proxy_id: props.proxyId ?? null },
      { signal: request.signal }
    )
    if (request.signal.aborted || currentGeneration !== generation) {
      authorizationWindow?.close()
      return
    }
    const safeVerificationUri = safeGitHubVerificationUri(
      result.verification_uri_complete || result.verification_uri
    )
    if (!result.oauth_session_id || !result.user_code || !safeVerificationUri) {
      throw new Error(t('admin.accounts.codingPlans.githubCopilot.oauth.startFailed'))
    }
    deviceSessionId.value = result.oauth_session_id
    userCode.value = result.user_code
    verificationUri.value = safeVerificationUri
    pollIntervalMs = Math.max(1, result.interval || 5) * 1000
    expiresAt = Date.now() + Math.max(1, result.expires_in || 900) * 1000
    if (authorizationWindow && !authorizationWindow.closed) {
      authorizationWindow.location.href = safeVerificationUri
    }
    schedulePoll(currentGeneration)
  } catch (error) {
    authorizationWindow?.close()
    if (request.signal.aborted || currentGeneration !== generation) return
    errorMessage.value = errorText(error, 'admin.accounts.codingPlans.githubCopilot.oauth.startFailed')
  } finally {
    if (activeRequest === request) activeRequest = undefined
    if (currentGeneration === generation) starting.value = false
  }
}

const reset = () => {
  clearAuthorization()
}

defineExpose({ reset })

onMounted(() => void loadCapabilities())
onUnmounted(clearAuthorization)
</script>
