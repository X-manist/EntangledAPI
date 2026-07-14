<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="border-b border-gray-200 bg-white/95 dark:border-dark-800 dark:bg-dark-900/95">
      <div class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-3">
          <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-base font-semibold text-gray-950 dark:text-white">
            {{ siteName }}
          </span>
        </RouterLink>
        <RouterLink
          to="/login"
          class="inline-flex flex-shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-sm shadow-primary-600/20 transition hover:bg-primary-700"
        >
          {{ t('auth.login') }}
        </RouterLink>
      </div>
    </header>

    <main class="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:py-10">
      <div v-if="loading" class="flex min-h-[320px] items-center justify-center">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <article v-else class="space-y-8">
        <!-- 页面标题 -->
        <div class="border-b border-gray-200 pb-6 dark:border-dark-700">
          <div class="flex items-start gap-4">
            <span class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-md bg-primary-50 text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">
              <Icon name="document" size="md" />
            </span>
            <div class="min-w-0">
              <h1 class="mt-2 text-2xl font-bold tracking-normal text-gray-950 dark:text-white sm:text-3xl">
                {{ t('public.contact.title') }}
              </h1>
              <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
                {{ t('public.contact.subtitle') }}
              </p>
            </div>
          </div>
        </div>

        <!-- 客服联系方式 -->
        <section class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.contact.customerService') }}</h2>
          <div class="mt-6 space-y-4">
            <div v-if="contactInfo.phone" class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                <Icon name="chat" size="sm" />
              </div>
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">{{ t('public.contact.phone') }}</h3>
                <p class="mt-1 text-gray-700 dark:text-dark-200">{{ contactInfo.phone }}</p>
                <p v-if="contactInfo.phoneHours" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ contactInfo.phoneHours }}
                </p>
              </div>
            </div>

            <div v-if="contactInfo.email" class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                <Icon name="document" size="sm" />
              </div>
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">{{ t('public.contact.email') }}</h3>
                <a :href="`mailto:${contactInfo.email}`" class="mt-1 block text-primary-600 hover:text-primary-700 dark:text-primary-300">
                  {{ contactInfo.email }}
                </a>
                <p v-if="contactInfo.emailNote" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ contactInfo.emailNote }}
                </p>
              </div>
            </div>

            <div v-if="contactInfo.wechat" class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                <Icon name="globe" size="sm" />
              </div>
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">{{ t('public.contact.wechat') }}</h3>
                <p class="mt-1 text-gray-700 dark:text-dark-200">{{ contactInfo.wechat }}</p>
              </div>
            </div>

            <div v-if="contactInfo.workingHours" class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                <Icon name="cog" size="sm" />
              </div>
              <div>
                <h3 class="font-medium text-gray-900 dark:text-white">{{ t('public.contact.workingHours') }}</h3>
                <p class="mt-1 text-gray-700 dark:text-dark-200">{{ contactInfo.workingHours }}</p>
              </div>
            </div>
          </div>
        </section>

        <!-- 公司地址 -->
        <section v-if="contactInfo.address" class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.contact.companyAddress') }}</h2>
          <div class="mt-4">
            <div class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                <Icon name="globe" size="sm" />
              </div>
              <div>
                <p class="text-gray-700 dark:text-dark-200">{{ contactInfo.address }}</p>
                <p v-if="contactInfo.postalCode" class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('public.contact.postalCode') }}: {{ contactInfo.postalCode }}
                </p>
              </div>
            </div>
          </div>
        </section>

        <!-- 常见问题 -->
        <section v-if="contactInfo.faqUrl" class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.contact.faq') }}</h2>
          <p class="mt-4 text-gray-700 dark:text-dark-200">
            {{ t('public.contact.faqDescription') }}
          </p>
          <a :href="contactInfo.faqUrl" target="_blank" rel="noopener noreferrer" class="mt-4 inline-flex items-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-primary-700">
            {{ t('public.contact.viewFaq') }}
            <Icon name="arrowRight" size="xs" />
          </a>
        </section>
      </article>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { getPublicSettings } from '@/api/auth'
import { resolveDisplayBrandName } from '@/utils/branding'
import { sanitizeUrl } from '@/utils/url'
import type { PublicSettings } from '@/types'

const { t } = useI18n()
const settings = ref<PublicSettings | null>(null)
const loading = ref(true)

const siteName = computed(() => resolveDisplayBrandName(settings.value?.site_name, 'zh'))
const siteLogo = computed(() => sanitizeUrl(settings.value?.site_logo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))

// 从设置中获取联系信息
const contactInfo = computed(() => {
  return {
    phone: settings.value?.contact_phone || '',
    phoneHours: settings.value?.contact_phone_hours || '',
    email: settings.value?.contact_email || '',
    emailNote: settings.value?.contact_email_note || '',
    wechat: settings.value?.contact_wechat || '',
    workingHours: settings.value?.contact_working_hours || '',
    address: settings.value?.contact_address || '',
    postalCode: settings.value?.contact_postal_code || '',
    faqUrl: settings.value?.contact_faq_url || '',
  }
})

onMounted(async () => {
  loading.value = true
  try {
    settings.value = await getPublicSettings()
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    loading.value = false
  }
})
</script>