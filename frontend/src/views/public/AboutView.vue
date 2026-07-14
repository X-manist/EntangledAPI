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
              <Icon name="globe" size="md" />
            </span>
            <div class="min-w-0">
              <h1 class="mt-2 text-2xl font-bold tracking-normal text-gray-950 dark:text-white sm:text-3xl">
                {{ t('public.about.title') }}
              </h1>
              <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
                {{ t('public.about.subtitle') }}
              </p>
            </div>
          </div>
        </div>

        <!-- 公司介绍 -->
        <section class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.about.companyIntro') }}</h2>
          <div class="mt-4 space-y-3 text-gray-700 dark:text-dark-200">
            <p v-if="aboutContent.companyName">
              <strong>{{ t('public.about.companyName') }}：</strong>{{ aboutContent.companyName }}
            </p>
            <p v-if="aboutContent.businessType">
              <strong>{{ t('public.about.businessType') }}：</strong>{{ aboutContent.businessType }}
            </p>
            <p v-if="aboutContent.registrationNumber">
              <strong>{{ t('public.about.registrationNumber') }}：</strong>{{ aboutContent.registrationNumber }}
            </p>
            <p v-if="aboutContent.businessScope">
              <strong>{{ t('public.about.businessScope') }}：</strong>{{ aboutContent.businessScope }}
            </p>
            <p v-if="aboutContent.description" class="mt-4 leading-relaxed">
              {{ aboutContent.description }}
            </p>
          </div>
        </section>

        <!-- 服务介绍 -->
        <section class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.about.services') }}</h2>
          <div class="mt-4 space-y-3 text-gray-700 dark:text-dark-200">
            <p v-if="aboutContent.serviceDescription" class="leading-relaxed">
              {{ aboutContent.serviceDescription }}
            </p>
            <ul v-if="aboutContent.serviceList && aboutContent.serviceList.length > 0" class="ml-6 list-disc space-y-2">
              <li v-for="(service, index) in aboutContent.serviceList" :key="index">{{ service }}</li>
            </ul>
          </div>
        </section>

        <!-- 联系方式 -->
        <section class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.about.contactInfo') }}</h2>
          <div class="mt-4 space-y-3 text-gray-700 dark:text-dark-200">
            <p v-if="aboutContent.address">
              <strong>{{ t('public.about.address') }}：</strong>{{ aboutContent.address }}
            </p>
            <p v-if="aboutContent.phone">
              <strong>{{ t('public.about.phone') }}：</strong>{{ aboutContent.phone }}
            </p>
            <p v-if="aboutContent.email">
              <strong>{{ t('public.about.email') }}：</strong>
              <a :href="`mailto:${aboutContent.email}`" class="text-primary-600 hover:text-primary-700 dark:text-primary-300">
                {{ aboutContent.email }}
              </a>
            </p>
            <p v-if="aboutContent.businessHours">
              <strong>{{ t('public.about.businessHours') }}：</strong>{{ aboutContent.businessHours }}
            </p>
          </div>
        </section>

        <!-- 备案信息 -->
        <section v-if="aboutContent.icpLicense" class="rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('public.about.icpInfo') }}</h2>
          <div class="mt-4 text-gray-700 dark:text-dark-200">
            <p>
              <strong>{{ t('public.about.icpLicense') }}：</strong>
              <a v-if="aboutContent.icpLicenseUrl" :href="aboutContent.icpLicenseUrl" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:text-primary-700 dark:text-primary-300">
                {{ aboutContent.icpLicense }}
              </a>
              <span v-else>{{ aboutContent.icpLicense }}</span>
            </p>
          </div>
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

// 从设置中获取关于我们的内容
const aboutContent = computed(() => {
  // 这里可以从后端设置中读取，暂时使用默认值
  return {
    companyName: settings.value?.about_company_name || '',
    businessType: settings.value?.about_business_type || '',
    registrationNumber: settings.value?.about_registration_number || '',
    businessScope: settings.value?.about_business_scope || '',
    description: settings.value?.about_description || '',
    serviceDescription: settings.value?.about_service_description || '',
    serviceList: settings.value?.about_service_list || [],
    address: settings.value?.about_address || '',
    phone: settings.value?.about_phone || '',
    email: settings.value?.about_email || '',
    businessHours: settings.value?.about_business_hours || '',
    icpLicense: settings.value?.icp_license || '',
    icpLicenseUrl: settings.value?.icp_license_url || '',
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