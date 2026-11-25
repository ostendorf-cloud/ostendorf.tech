<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  GovueHeader,
  GovueLanguageSwitch,
  GovueNav,
  GovueThemeSwitch,
  type NavItem,
} from '@ostendorf-cloud/govue'

import { trackHeaderHeight } from '../lib/headerHeight'
import { useMediaQuery } from '../lib/mediaQuery'
import { IMPRESSIONS_HIDDEN_QUERY, IMPRESSIONS_ID, impressionsOnPage } from '../lib/pageSections'
import { NOT_FOUND_ROUTE_NAME } from '../router'
import LogoMark from './LogoMark.vue'
import WordMark from './WordMark.vue'

// The header's height is not a constant: below 480px govue's header puts
// the navigation on a second row (styles.css holds the prerender fallback
// per breakpoint). Measuring it once here keeps
// scroll-padding-top (styles.css) equal to what is really on screen.
const header = ref<{ $el: HTMLElement } | null>(null)
let stopTracking = (): void => {}
onMounted(() => {
  if (header.value) {
    stopTracking = trackHeaderHeight(header.value.$el)
  }
})
onBeforeUnmount(() => {
  stopTracking()
})

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const impressionsHiddenInRow = useMediaQuery(IMPRESSIONS_HIDDEN_QUERY)

/** Every nav item is an in-page anchor on the home page: this milestone
 *  has only one page with sections, so there is nowhere else to link a
 *  section heading to. GovueNav renders a plain anchor, not a router-link,
 *  so each href is resolved to a real path up front. The contact link is
 *  the call to action, govue's `cta` item. */
const items = computed<NavItem[]>(() => {
  const locale = route.params.locale
  const home = (hash: string) => router.resolve({ name: 'home', params: { locale }, hash }).href
  return [
    { href: home('#services'), label: t('nav.services') },
    // Only while the section is on the page: it renders in the browser, after
    // the feed answered, so the prerendered nav never has this entry. And not
    // in the one-row range from 640 to 1023 px, where the row has no room for
    // a fifth link: govue's menu below 640 px lists it, the row from 1024 px
    // shows it.
    ...(impressionsOnPage.value && !impressionsHiddenInRow.value
      ? [{ href: home(`#${IMPRESSIONS_ID}`), label: t('nav.impressions') }]
      : []),
    { href: home('#bands'), label: t('nav.bands') },
    { href: home('#about'), label: t('nav.about') },
    { href: home('#contact'), label: t('nav.contact'), variant: 'cta' },
  ]
})

/** linkFor is the current page in another language: same route name,
 *  params and hash, only :locale swapped, so switching never drops a
 *  visitor back to the home page. A repeatable param resolved with no
 *  entries, the not-found route's own pathMatch for instance, drops the
 *  trailing slash the route pattern carries; it is restored for that route
 *  only, so the link is canonical. The legal pages never end in a slash:
 *  govue's /<code>/<page> is the one canonical path. */
function linkFor(code: string): string {
  const href = router.resolve({
    name: route.name ?? undefined,
    params: { ...route.params, locale: code },
    hash: route.hash,
  }).href
  return route.name === NOT_FOUND_ROUTE_NAME && !href.endsWith('/') ? `${href}/` : href
}

/** overlay is govue's overlay header, on the home page only: transparent
 *  over the hero until the page scrolls, solid after. Every other page has
 *  no hero behind the header, so it keeps the solid sticky header. The
 *  site's own contrast bound for the hero is in src/theme/contrast.test.ts,
 *  and src/styles.css gives the transparent header the light roles in
 *  light mode, where the hero is a pale ground. */
const overlay = computed(() => route.name === 'home')

/** navigate hands a plain click to the router, so the page does not reload. */
function navigate(href: string): boolean {
  void router.push(href)
  return true
}
</script>

<template>
  <GovueHeader ref="header" nav-placement="center" :overlay="overlay">
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

    <!-- The same two controls at every width: the one cycle button and only
         the link to the other language. GovueHeader puts both after the
         navigation, at the right edge of the bar, theme button first and the
         language link last. The navigation itself sits in the middle. -->
    <template #theme>
      <GovueThemeSwitch cycle />
    </template>

    <template #actions>
      <GovueLanguageSwitch :link-for="linkFor" :navigate="navigate" others-only />
    </template>
  </GovueHeader>
</template>
