<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { GovueSkipLink } from '@ostendorf-cloud/govue'

import AppFooter from './components/AppFooter.vue'
import AppHeader from './components/AppHeader.vue'
import { isLocale, type Locale } from './i18n'
import { usePageHead } from './lib/head'
import type { PageRoute } from './lib/seo'

const route = useRoute()
const i18n = useI18n()

const routeNameToPageRoute: Record<string, PageRoute> = {
  home: 'home',
  imprint: 'imprint',
  privacy: 'privacy',
  agb: 'agb',
  'not-found': 'notFound',
}

const pageRoute = computed<PageRoute>(() => routeNameToPageRoute[String(route.name)] ?? 'notFound')
const locale = computed<Locale>(() => {
  const value = route.params.locale
  return typeof value === 'string' && isLocale(value) ? value : 'de'
})

usePageHead(pageRoute, locale, i18n)
</script>

<template>
  <GovueSkipLink>{{ i18n.t('common.skipToContent') }}</GovueSkipLink>
  <AppHeader />
  <router-view />
  <AppFooter />
</template>
