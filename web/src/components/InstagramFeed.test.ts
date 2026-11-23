import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
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

  it('puts a decorative overlay with the Instagram glyph in every tile', async () => {
    const { container } = renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(container.querySelectorAll('.insta-overlay')).toHaveLength(2)
    })

    for (const overlay of container.querySelectorAll('.insta-overlay')) {
      expect(overlay).toHaveAttribute('aria-hidden', 'true')
      const glyph = overlay.querySelector('svg')
      expect(glyph).not.toBeNull()
      expect(glyph).toHaveAttribute('aria-hidden', 'true')
      expect(glyph?.querySelector('circle')).not.toBeNull()
    }
  })

  it('keeps the link named by the picture, not by the glyph', async () => {
    renderFeed(() => Promise.resolve({ posts }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'Lichtpult im Saal' })).toBeVisible()
    })
    expect(screen.getByRole('link', { name: 'Bühne von hinten' })).toBeVisible()
    expect(screen.getByRole('link', { name: 'Lichtpult im Saal' }).textContent?.trim()).toBe('')
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

  describe('load more', () => {
    function manyPosts(n: number): InstagramPost[] {
      return Array.from({ length: n }, (_, i) => ({
        id: `p${i}`,
        alt: `Bild ${i}`,
        permalink: `https://www.instagram.com/p/${i}/`,
      }))
    }

    async function renderMany(n: number, locale: 'de' | 'en' = 'de') {
      const result = render(InstagramFeed, {
        props: { client: fakeClient(() => Promise.resolve({ posts: manyPosts(n) })) },
        global: { plugins: [createAppI18n(locale)] },
      })
      await waitFor(() => {
        expect(screen.getAllByRole('img').length).toBeGreaterThan(0)
      })
      return result
    }

    it('shows a first batch of 12 and a button for the rest', async () => {
      await renderMany(20)
      expect(screen.getAllByRole('img')).toHaveLength(12)
      const button = screen.getByRole('button', { name: 'Mehr laden' })
      expect(button).toHaveAttribute('type', 'button')
    })

    it('reveals the next batch on click and drops the button at the end', async () => {
      await renderMany(20)
      await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden' }))
      expect(screen.getAllByRole('img')).toHaveLength(20)
      expect(screen.queryByRole('button', { name: 'Mehr laden' })).toBeNull()
    })

    it('reveals exactly one batch per click', async () => {
      await renderMany(30)
      await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden' }))
      expect(screen.getAllByRole('img')).toHaveLength(24)
      expect(screen.getByRole('button', { name: 'Mehr laden' })).toBeVisible()
    })

    it('shows no button when the posts fit one batch', async () => {
      await renderMany(12)
      expect(screen.getAllByRole('img')).toHaveLength(12)
      expect(screen.queryByRole('button')).toBeNull()
    })

    it('announces the new pictures politely, in German', async () => {
      await renderMany(20)
      const region = screen.getByTestId('insta-announce')
      expect(region).toHaveAttribute('aria-live', 'polite')
      expect(region.textContent?.trim()).toBe('')
      await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden' }))
      expect(region.textContent?.trim()).toBe('8 weitere Bilder geladen')
    })

    it('labels the button and the announcement in English', async () => {
      await renderMany(20, 'en')
      await fireEvent.click(screen.getByRole('button', { name: 'Load more' }))
      expect(screen.getByTestId('insta-announce').textContent?.trim()).toBe(
        '8 more pictures loaded',
      )
    })

    it('reveals every post of a large feed batch by batch, with one request', async () => {
      let requests = 0
      render(InstagramFeed, {
        props: {
          client: fakeClient(() => {
            requests += 1
            return Promise.resolve({ posts: manyPosts(200) })
          }),
        },
        global: { plugins: [createAppI18n('de')] },
      })
      await waitFor(() => {
        expect(screen.getAllByRole('img')).toHaveLength(12)
      })
      let clicks = 0
      while (screen.queryByRole('button', { name: 'Mehr laden' })) {
        await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden' }))
        clicks += 1
      }
      expect(screen.getAllByRole('img')).toHaveLength(200)
      expect(clicks).toBe(Math.ceil(200 / 12) - 1)
      expect(requests).toBe(1)
    })

    it('announces a single new picture in the singular', async () => {
      await renderMany(13)
      await fireEvent.click(screen.getByRole('button', { name: 'Mehr laden' }))
      expect(screen.getByTestId('insta-announce').textContent?.trim()).toBe(
        '1 weiteres Bild geladen',
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
