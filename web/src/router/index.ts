import type { Ref } from 'vue'
import { createRouter, type RouteRecordRaw, type RouterHistory, type Router } from 'vue-router'

import { detectPreferredLocale, isLocale } from '../i18n'

const routes: RouteRecordRaw[] = [
  {
    path: '/:locale(de|en)/',
    name: 'home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/:locale(de|en)/imprint/',
    name: 'imprint',
    component: () => import('../views/ImprintView.vue'),
  },
  {
    path: '/:locale(de|en)/privacy/',
    name: 'privacy',
    component: () => import('../views/PrivacyView.vue'),
  },
  {
    path: '/:locale(de|en)/agb/',
    name: 'agb',
    component: () => import('../views/AgbView.vue'),
  },
  {
    path: '/:locale(de|en)/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('../views/NotFoundView.vue'),
  },
  {
    // No locale prefix at all, "/" included: redirected to the preferred
    // locale's home page, the same rule the server applies to "/" itself.
    path: '/:pathMatch(.*)*',
    redirect: () => `/${detectPreferredLocale(navigatorLanguages())}/`,
  },
]

/** navigatorLanguages reads the browser's language preference, or an empty
 *  list when run outside a browser, such as the prerender script. */
function navigatorLanguages(): readonly string[] {
  if (typeof navigator === 'undefined') {
    return []
  }
  return navigator.languages ?? [navigator.language]
}

/** createAppRouter builds the router over history, so the client supplies
 *  a browser history and the prerender script supplies a memory one. */
export function createAppRouter(history: RouterHistory): Router {
  return createRouter({ history, routes })
}

/**
 * attachLocaleSync keeps the active locale ref, and the document's lang
 * attribute, in step with the route's own :locale param on every
 * navigation, including the first one. A locale switch is then nothing
 * more than a normal navigation to the same route under the other
 * locale prefix; nothing else has to set the active locale by hand.
 *
 * Takes the locale ref itself, i18n.global.locale, rather than the whole
 * Composer: the Composer's own generic type parameters are inferred from
 * the exact message schema it was built with, which makes "Composer" on
 * its own, with no type arguments, too narrow a parameter type to accept
 * the real instance createAppI18n returns.
 */
export function attachLocaleSync(router: Router, locale: Ref<string>): void {
  router.beforeEach((to) => {
    const next = to.params.locale
    if (typeof next === 'string' && isLocale(next)) {
      locale.value = next
      if (typeof document !== 'undefined') {
        document.documentElement.lang = next
      }
    }
  })
}
