import { createSSRApp } from 'vue'
import { createWebHistory } from 'vue-router'
import { useTheme } from '@ostendorf-cloud/govue'

import App from './App.vue'
import { createAppI18n, detectPreferredLocale, isLocale } from './i18n'
import { DEFAULT_MODE, THEME_NAME } from './lib/theme'
import { attachLocaleSync, createAppRouter } from './router'
import './styles.css'

// The theme and the starting mode both come from src/lib/theme.ts, which
// the prerender script reads as well: the markup a visitor first paints
// is already in the mode this call would otherwise switch them to. The
// visitor's own choice, once they use the toggle, still wins over both.
useTheme({ theme: THEME_NAME, mode: DEFAULT_MODE })

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

// router.isReady() resolves after the initial navigation, including the
// async route components above, which is what the hydration this app
// mounts into needs: the DOM mount has to match what the server rendered
// for this exact path, not a transient empty state.
void router.isReady().then(() => {
  app.mount('#app')
})
