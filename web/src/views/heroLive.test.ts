/**
 * Proves the hero's beams and their pause button come to life together,
 * and only once the script has run: the prerendered page, and a visitor
 * without JavaScript, get the beams at rest and no button that does
 * nothing.
 */
import { render, screen } from '@testing-library/vue'
import { createSSRApp, nextTick } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import HeroStage from '../components/HeroStage.vue'
import { createAppI18n } from '../i18n'
import HomeView from './HomeView.vue'

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/:locale?/privacy', name: 'privacy', component: { render: () => null } },
      { path: '/:pathMatch(.*)*', component: { render: () => null } },
    ],
  })
}

async function prerenderedHome(): Promise<string> {
  const router = makeRouter()
  await router.push('/')
  const app = createSSRApp(HomeView)
  app.use(createAppI18n('de')).use(router)
  return renderToString(app)
}

describe('the prerendered hero', () => {
  it('has no live beams', async () => {
    const html = await prerenderedHome()
    expect(html).toContain('hero-stage')
    expect(html).not.toContain('is-live')
  })

  it('has no pause button, since there is nothing it could do', async () => {
    const html = await prerenderedHome()
    expect(html).not.toContain('hero-motion')
    expect(html).not.toContain('Animation pausieren')
  })
})

describe('the mounted hero', () => {
  it('makes the beams live once mounted', async () => {
    const { container } = render(HeroStage)
    await nextTick()
    expect(container.querySelector('.hero-stage')).toHaveClass('is-live')
  })

  it('shows the pause button once mounted', async () => {
    const router = makeRouter()
    await router.push('/')
    render(HomeView, { global: { plugins: [createAppI18n('de'), router] } })
    await nextTick()
    expect(screen.getByRole('button', { name: 'Animation pausieren' })).toBeInTheDocument()
  })
})
