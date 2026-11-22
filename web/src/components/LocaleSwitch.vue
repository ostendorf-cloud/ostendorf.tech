<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

import { type Locale } from '../i18n'
import { NOT_FOUND_ROUTE_NAME } from '../router'

/** The other locale's version of the current route: same name, same
 *  params and hash, only :locale swapped, so switching never drops a
 *  visitor back to the home page. */
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const otherLocale = computed<Locale>(() => (route.params.locale === 'de' ? 'en' : 'de'))
// A repeatable param resolved with no entries, the not-found route's own
// pathMatch for instance, drops the trailing slash the route pattern
// carries; restored here for that route only, so the link is already
// canonical instead of relying on the server's own redirect for it. The
// legal pages never end in a slash: govue's /<code>/<page> is the one
// canonical path and the server answers a slash with a redirect.
const target = computed(() => {
  const href = router.resolve({
    name: route.name ?? undefined,
    params: { ...route.params, locale: otherLocale.value },
    hash: route.hash,
  }).href
  return route.name === NOT_FOUND_ROUTE_NAME && !href.endsWith('/') ? `${href}/` : href
})
</script>

<template>
  <router-link
    :to="target"
    class="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 rounded-md px-2 text-base min-[480px]:px-4 font-medium text-accent-text transition-colors hover:bg-accent-subtle"
  >
    {{ t('common.switchLocale') }}
  </router-link>
</template>
