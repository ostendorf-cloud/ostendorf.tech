import { readFileSync } from 'node:fs'
import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import BandLogo from './BandLogo.vue'

describe('BandLogo', () => {
  it('renders nothing without a logo', () => {
    const { container } = render(BandLogo, { props: { alt: 'Logo' } })

    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('.band-logo-plate')).toBeNull()
  })

  it('renders a lazy image with alt text and its size on a plate', () => {
    const { container } = render(BandLogo, {
      props: { logo: { src: '/logo.webp', width: 120, height: 40 }, alt: 'Logo of Rubik' },
    })

    const img = screen.getByRole('img', { name: 'Logo of Rubik' })
    expect(img).toHaveAttribute('src', '/logo.webp')
    expect(img).toHaveAttribute('width', '120')
    expect(img).toHaveAttribute('height', '40')
    expect(img).toHaveAttribute('loading', 'lazy')
    expect(img.parentElement).toBe(container.querySelector('.band-logo-plate'))
  })

  it('sizes the image by a fixed height so the card does not grow when it loads', () => {
    const source = readFileSync(`${process.cwd()}/src/components/BandLogo.vue`, 'utf-8')
    const rule = /\.band-logo\s*\{([^}]*)\}/.exec(source)?.[1] ?? ''

    expect(rule).toMatch(/\bheight:\s*5\.5rem/)
    expect(rule).not.toMatch(/\bheight:\s*auto/)
    expect(rule).not.toMatch(/max-height/)
  })
})
