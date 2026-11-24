import { describe, expect, it } from 'vitest'

import { createAppI18n } from '../i18n'
import * as icons from './icons'
import { socialImageMeta } from './icons'

describe('icons', () => {
  it('writes no icon link: govue injects them at startup', () => {
    expect(Object.keys(icons)).not.toContain('iconLinks')
  })
})

describe('socialImageMeta', () => {
  it('builds an absolute og:image and twitter:image from the site URL', () => {
    const head = socialImageMeta('https://staging.example.test', 'Alt text').join('\n')

    expect(head).toContain(
      '<meta property="og:image" content="https://staging.example.test/og-image.png">',
    )
    expect(head).toContain(
      '<meta name="twitter:image" content="https://staging.example.test/og-image.png">',
    )
    expect(head).toContain('<meta name="twitter:card" content="summary_large_image">')
  })

  it('pins the image size to the generated 1200 by 630 file', () => {
    const head = socialImageMeta('https://staging.example.test', 'Alt text').join('\n')

    expect(head).toContain('<meta property="og:image:width" content="1200">')
    expect(head).toContain('<meta property="og:image:height" content="630">')
  })

  it('describes the image for og:image and twitter:image', () => {
    const head = socialImageMeta('https://staging.example.test', 'Alt text').join('\n')

    expect(head).toContain('<meta property="og:image:alt" content="Alt text">')
    expect(head).toContain('<meta name="twitter:image:alt" content="Alt text">')
  })

  it('has a German and an English description naming the brand', () => {
    for (const [locale, brand] of [
      ['de', 'Veranstaltungstechnik'],
      ['en', 'Event Technology'],
    ] as const) {
      const alt = createAppI18n(locale).global.t('common.socialImageAlt')

      expect(alt).toContain('Simon Ostendorf')
      expect(alt).toContain(brand)
    }
  })
})
