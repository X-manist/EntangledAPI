<template>
  <footer class="border-t border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-900">
    <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 gap-8 md:grid-cols-3">
        <!-- 网站信息 -->
        <div>
          <div class="flex items-center gap-3">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-8 w-8 object-contain" />
            <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
          </div>
          <p v-if="siteDescription" class="mt-3 text-sm text-gray-600 dark:text-dark-300">
            {{ siteDescription }}
          </p>
        </div>

        <!-- 快捷链接 -->
        <div>
          <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-900 dark:text-white">
            {{ t('footer.quickLinks') }}
          </h3>
          <ul class="mt-4 space-y-2">
            <li>
              <RouterLink to="/about" class="text-sm text-gray-600 hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400">
                {{ t('footer.aboutUs') }}
              </RouterLink>
            </li>
            <li>
              <RouterLink to="/contact" class="text-sm text-gray-600 hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400">
                {{ t('footer.contactUs') }}
              </RouterLink>
            </li>
            <li v-for="doc in legalDocuments" :key="doc.id">
              <RouterLink :to="`/legal/${doc.id}`" class="text-sm text-gray-600 hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400">
                {{ doc.title }}
              </RouterLink>
            </li>
          </ul>
        </div>

        <!-- 联系方式 -->
        <div v-if="contactEmail || contactPhone">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-gray-900 dark:text-white">
            {{ t('footer.contact') }}
          </h3>
          <ul class="mt-4 space-y-2">
            <li v-if="contactEmail">
              <a :href="`mailto:${contactEmail}`" class="text-sm text-gray-600 hover:text-primary-600 dark:text-dark-300 dark:hover:text-primary-400">
                {{ contactEmail }}
              </a>
            </li>
            <li v-if="contactPhone">
              <span class="text-sm text-gray-600 dark:text-dark-300">{{ contactPhone }}</span>
            </li>
          </ul>
        </div>
      </div>

      <!-- 备案信息和版权 -->
      <div class="mt-8 border-t border-gray-200 pt-6 dark:border-dark-700">
        <div class="flex flex-col items-center justify-center gap-2 text-center text-sm text-gray-500 dark:text-dark-400">
          <p v-if="icpLicense">
            <a v-if="icpLicenseUrl" :href="icpLicenseUrl" target="_blank" rel="noopener noreferrer" class="hover:text-primary-600 dark:hover:text-primary-400">
              {{ icpLicense }}
            </a>
            <span v-else>{{ icpLicense }}</span>
          </p>
          <p v-if="companyName">
            {{ currentYear }} © {{ companyName }}. {{ t('footer.allRightsReserved') }}
          </p>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { sanitizeUrl } from '@/utils/url'
import type { PublicSettings } from '@/types'

const { t } = useI18n()

interface Props {
  settings: PublicSettings | null
}

const props = defineProps<Props>()

const currentYear = new Date().getFullYear()

const siteName = computed(() => props.settings?.site_name || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(props.settings?.site_logo || '', {
  allowRelative: true,
  allowDataUrl: true,
}))
const siteDescription = computed(() => props.settings?.site_description || '')
const companyName = computed(() => props.settings?.about_company_name || props.settings?.site_name || '')
const icpLicense = computed(() => props.settings?.icp_license || '')
const icpLicenseUrl = computed(() => props.settings?.icp_license_url || '')
const contactEmail = computed(() => props.settings?.contact_email || '')
const contactPhone = computed(() => props.settings?.contact_phone || '')
const legalDocuments = computed(() => props.settings?.login_agreement_documents || [])
</script>