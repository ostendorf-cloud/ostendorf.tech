import { readFileSync } from 'node:fs'
import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import BandPhoto from './BandPhoto.vue'

describe('BandPhoto', () => {
  it('renders nothing without a photo', () => {
    const { container } = render(BandPhoto, { props: { alt: 'Photo' } })

    expect(container.querySelector('img')).toBeNull()
  })

  it('renders a lazy image with alt text and its size', () => {
    render(BandPhoto, {
      props: { photo: { src: '/photo.webp', width: 315, height: 315 }, alt: 'Photo of Rubik' },
    })

    const img = screen.getByRole('img', { name: 'Photo of Rubik' })
    expect(img).toHaveAttribute('src', '/photo.webp')
    expect(img).toHaveAttribute('width', '315')
    expect(img).toHaveAttribute('height', '315')
    expect(img).toHaveAttribute('loading', 'lazy')
  })

  it('shows a square with 14px corners that is cropped, not stretched', () => {
    const source = readFileSync(`${process.cwd()}/src/components/BandPhoto.vue`, 'utf-8')
    const rule = /\.band-photo\s*\{([^}]*)\}/.exec(source)?.[1] ?? ''

    expect(rule).toMatch(/border-radius:\s*14px/)
    expect(rule).toMatch(/object-fit:\s*cover/)
    expect(rule.match(/\bwidth:\s*([\d.]+rem)/)?.[1]).toBe(
      rule.match(/\bheight:\s*([\d.]+rem)/)?.[1],
    )
  })
})
