<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovueHeader, GovueNav, type NavItem } from '@ostendorf-cloud/govue'

import LocaleSwitch from './LocaleSwitch.vue'
import LogoMark from './LogoMark.vue'
import ThemeToggle from './ThemeToggle.vue'
import WordMark from './WordMark.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

/** Every nav item is an in-page anchor on the home page: this milestone
 *  has only one page with sections, so there is nowhere else to link a
 *  section heading to. GovueNav renders a plain anchor, not a router-link,
 *  so each href is resolved to a real path up front. */
const items = computed<NavItem[]>(() => {
  const locale = route.params.locale
  const home = (hash: string) => router.resolve({ name: 'home', params: { locale }, hash }).href
  return [
    { href: home('#services'), label: t('nav.services') },
    { href: home('#bands'), label: t('nav.bands') },
    { href: home('#about'), label: t('nav.about') },
    { href: home('#contact'), label: t('nav.contact') },
  ]
})
</script>

<template>
  <GovueHeader>
    <template #brand>
      <router-link
        :to="{ name: 'home', params: { locale: route.params.locale } }"
        class="inline-flex min-h-11 items-center gap-2"
        :aria-label="t('common.siteName')"
      >
        <LogoMark />
        <WordMark />
      </router-link>
    </template>
    <GovueNav :items="items" :aria-label="t('nav.main')" />
    <template #actions>
      <LocaleSwitch />
      <ThemeToggle />
    </template>
  </GovueHeader>
</template>
