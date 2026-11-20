import { describe, expect, it } from 'vitest'

import { iconLinks, socialImageMeta } from './icons'

describe('iconLinks', () => {
  const head = iconLinks().join('\n')

  it('links the SVG icon, the PNG fallback and the touch icon', () => {
    expect(head).toContain('rel="icon" href="/favicon.svg" type="image/svg+xml"')
    expect(head).toContain('rel="icon" href="/favicon-32x32.png" type="image/png" sizes="32x32"')
    expect(head).toContain('rel="apple-touch-icon" href="/apple-touch-icon.png"')
  })

  it('uses root absolute paths only', () => {
    for (const href of head.matchAll(/href="([^"]+)"/g)) {
      expect(href[1]).toMatch(/^\/[^/]/)
    }
  })
})

describe('socialImageMeta', () => {
  it('builds an absolute og:image and twitter:image from the site URL', () => {
    const head = socialImageMeta('https://staging.example.test').join('\n')

    expect(head).toContain(
      '<meta property="og:image" content="https://staging.example.test/og-image.png">',
    )
    expect(head).toContain(
      '<meta name="twitter:image" content="https://staging.example.test/og-image.png">',
    )
    expect(head).toContain('<meta name="twitter:card" content="summary_large_image">')
  })
})
