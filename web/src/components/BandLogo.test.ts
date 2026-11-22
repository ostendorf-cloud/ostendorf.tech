import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'

import BandLogo from './BandLogo.vue'

describe('BandLogo', () => {
  it('renders nothing without a logo', () => {
    const { container } = render(BandLogo, { props: {} })

    expect(container.querySelector('img')).toBeNull()
  })

  it('renders a lazy image with alt text and its size', () => {
    render(BandLogo, {
      props: { logo: { src: '/logo.svg', alt: 'Logo of Rubik', width: 120, height: 40 } },
    })

    const img = screen.getByRole('img', { name: 'Logo of Rubik' })
    expect(img).toHaveAttribute('src', '/logo.svg')
    expect(img).toHaveAttribute('width', '120')
    expect(img).toHaveAttribute('height', '40')
    expect(img).toHaveAttribute('loading', 'lazy')
  })
})
