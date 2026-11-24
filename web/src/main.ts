import { createSSRApp } from 'vue'
import { createWebHistory } from 'vue-router'
import { useTheme } from '@ostendorf-cloud/govue'

import App from './App.vue'
import { createAppI18n, detectPreferredLocale, isLocale } from './i18n'
import { createSiteGovueI18n } from './i18n/govue'
import { DEFAULT_MODE, STORAGE_KEY, THEME_NAME } from './lib/theme'
import { attachLocaleSync, createAppRouter } from './router'
import './styles.css'

// The theme and the starting mode both come from src/lib/theme.ts, which
// the prerender script reads as well. Avoiding a flash of the wrong mode
// for a visitor with a stored light or system preference is the
// prerendered <head>'s own blocking boot script's job (see
// web/scripts/prerender.ts), not this call's: that script runs before
// first paint, this call only once the deferred bundle has loaded. This
// call brings govue's reactive theme state, which the header's theme
// switch operates on, in line with what the boot script already wrote
// into the DOM, using the same storage key so both read the same choice.
useTheme({ theme: THEME_NAME, mode: DEFAULT_MODE, storageKey: STORAGE_KEY })

const firstSegment = window.location.pathname.split('/')[1] ?? ''
const initialLocale = isLocale(firstSegment)
  ? firstSegment
  : detectPreferredLocale(navigator.languages ?? [navigator.language])

const i18n = createAppI18n(initialLocale)
const router = createAppRouter(createWebHistory())
attachLocaleSync(router, i18n.global.locale)

const app = createSSRApp(App)
app.use(router)
app.use(i18n)
// govue's language layer shares the site's own vue-i18n instance: it adds
// the govue catalogues (the footer's legal link labels, the legal page
// shell) and tells those components the active language. The route's own
// :locale still decides the locale, see attachLocaleSync. The site's own
// German contact form texts ("Sie") are merged over govue's.
app.use(createSiteGovueI18n({ i18n, locale: initialLocale }))

// router.isReady() resolves after the initial navigation, including the
// async route components above, which is what the hydration this app
// mounts into needs: the DOM mount has to match what the server rendered
// for this exact path, not a transient empty state.
void router.isReady().then(() => {
  app.mount('#app')
})
