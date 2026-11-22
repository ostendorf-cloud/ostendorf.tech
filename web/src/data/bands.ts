import herrenjedeckLogo from '../assets/bands/herrenjedeck-logo.webp'

/**
 * The bands and ensembles shown on the home page. Only the facts that do
 * not translate live here: the slug ties a band to its i18n messages under
 * bands.<slug>.name and bands.<slug>.paragraphs, and the Instagram link is
 * the same in both locales.
 *
 * hasDescription is false for a band with no description text yet, in
 * which case only the name and the Instagram link are rendered.
 *
 * accent names one of the theme's own colour roles (accent, info, success,
 * warning, danger). HomeView.vue maps it to a per band class that reads
 * that role's text and subtle tokens, so a band card has its own colour
 * in light and dark mode without a literal colour anywhere. The theme has
 * five roles for seven bands, so two roles repeat; neighbours differ.
 *
 * logo is optional and rendered only when given. To add one, put the file
 * into src/assets/bands/, import it here and set { src, width, height }
 * with the file's real pixel size, then add bands.<slug>.logoAlt to both
 * locales. Get the band's permission before a logo goes on the site.
 */
export type BandAccent = 'accent' | 'info' | 'success' | 'warning' | 'danger'

export interface BandLogo {
  src: string
  width: number
  height: number
}

export interface Band {
  slug: string
  instagramHandle: string
  instagramUrl: string
  hasDescription: boolean
  accent: BandAccent
  logo?: BandLogo
}

export const bands: Band[] = [
  {
    slug: 'rubik',
    instagramHandle: '@wirsindrubik',
    instagramUrl: 'https://www.instagram.com/wirsindrubik/',
    hasDescription: true,
    accent: 'accent',
  },
  {
    slug: 'bademeister',
    instagramHandle: '@bademeister.band',
    instagramUrl: 'https://www.instagram.com/bademeister.band',
    hasDescription: true,
    accent: 'info',
  },
  {
    slug: 'fourfold',
    instagramHandle: '@fourfoldband',
    instagramUrl: 'https://www.instagram.com/fourfoldband/',
    hasDescription: true,
    accent: 'success',
  },
  {
    slug: 'singingLittleBigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'warning',
  },
  {
    slug: 'bigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'danger',
  },
  {
    slug: 'jazzCombo',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'accent',
  },
  {
    slug: 'herrenjedeck',
    instagramHandle: '@herrenjedeck',
    instagramUrl: 'https://www.instagram.com/herrenjedeck/',
    hasDescription: true,
    accent: 'info',
    // The band's own logo, with the band's permission (see NOTICE).
    logo: { src: herrenjedeckLogo, width: 480, height: 489 },
  },
]
