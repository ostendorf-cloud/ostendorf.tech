import { render, screen, waitFor } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import { createAppI18n } from '../i18n'
import InstagramFeed, { type InstagramPost } from './InstagramFeed.vue'

const posts: InstagramPost[] = [
  {
    id: '17912345678901234',
    alt: 'Lichtpult im Saal',
    permalink: 'https://www.instagram.com/p/A/',
  },
  { id: 'abc_DEF', alt: 'Bühne von hinten', permalink: 'https://www.instagram.com/p/B/' },
]

/** fakeClient is the API client as the component uses it: one GET that
 *  answers whatever a test hands it, or rejects. */
function fakeClient(answer: () => Promise<unknown>) {
  return { get: <T>() => answer() as Promise<T> }
}

function renderFeed(answer: () => Promise<unknown>) {
  return render(InstagramFeed, {
    props: { client: fakeClient(answer) },
    global: { plugins: [createAppI18n('de')] },
  })
}

describe('InstagramFeed', () => {
  it('renders one tile per post, lazily loaded and with the alt text the API gave', async () => {
    renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Impressionen' })).toBeVisible()
    })

    const images = screen.getAllByRole('img')
    expect(images).toHaveLength(2)
    expect(images[0]).toHaveAttribute('alt', 'Lichtpult im Saal')
    expect(images[0]).toHaveAttribute('loading', 'lazy')
    expect(images[1]).toHaveAttribute('alt', 'Bühne von hinten')
  })

  it('loads every picture from this site, never from Meta', async () => {
    renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getAllByRole('img')).toHaveLength(2)
    })

    for (const image of screen.getAllByRole('img')) {
      expect(image.getAttribute('src')).toMatch(/^\/api\/v1\/instagram\//)
    }
  })

  it('links each tile to the post on Instagram, in a new tab', async () => {
    renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Lichtpult im Saal' })).toBeVisible()
    })

    const link = screen.getByRole('link', { name: 'Lichtpult im Saal' })
    expect(link).toHaveAttribute('href', 'https://www.instagram.com/p/A/')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('offers the account to follow', async () => {
    renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Auf Instagram folgen' })).toHaveAttribute(
        'href',
        'https://www.instagram.com/simonostendorf.vtech/',
      )
    })
  })

  it('renders nothing at all when the feed is empty', async () => {
    const { container } = renderFeed(() => Promise.resolve({ posts: [] }))

    await waitFor(() => {
      expect(container.querySelector('section')).toBeNull()
    })
    expect(screen.queryByRole('heading', { name: 'Impressionen' })).toBeNull()
  })

  it('renders nothing when the request fails, without the failure escaping', async () => {
    const { container } = renderFeed(() => Promise.reject(new Error('no connection to the server')))

    await waitFor(() => {
      expect(container.querySelector('section')).toBeNull()
    })
    expect(screen.queryByRole('heading', { name: 'Impressionen' })).toBeNull()
    expect(screen.queryAllByRole('img')).toHaveLength(0)
  })

  it('renders nothing when the answer carries no posts field at all', async () => {
    const { container } = renderFeed(() => Promise.resolve({}))

    await waitFor(() => {
      expect(container.querySelector('section')).toBeNull()
    })
  })
})
