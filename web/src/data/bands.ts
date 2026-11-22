import herrenjedeckLogo from '../assets/bands/herrenjedeck-logo.webp'

/**
 * The bands and ensembles shown on the home page, in the order of the cards.
 * Only the facts that do not translate live here: the slug ties a band to
 * its i18n messages under bands.<slug>.name and bands.<slug>.paragraphs, and
 * the Instagram link is the same in both locales.
 *
 * hasDescription is false for a band with no description text yet, in
 * which case only the name and the Instagram link are rendered.
 *
 * accent names one of three of the theme's own colour roles (accent, info,
 * success). HomeView.vue maps it to a per band class that reads that role's
 * text and subtle tokens, so a band card has its own colour in light and
 * dark mode without a literal colour anywhere. Warning and danger are left
 * out on purpose (owner decision): a decoration must not look like a
 * message. The roles repeat in that order, so any three bands in a row
 * differ, which is a full row of the carousel on a wide screen.
 *
 * logo is optional and rendered only when given. To add one, put the file
 * into src/assets/bands/, import it here and set { src, width, height }
 * with the file's real pixel size, then add bands.<slug>.logoAlt to both
 * locales. Get the band's permission before a logo goes on the site.
 */
export type BandAccent = 'accent' | 'info' | 'success'

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
    slug: 'herrenjedeck',
    instagramHandle: '@herrenjedeck',
    instagramUrl: 'https://www.instagram.com/herrenjedeck/',
    hasDescription: true,
    accent: 'accent',
    // The band's own logo; the owner confirmed the band's permission (see NOTICE).
    logo: { src: herrenjedeckLogo, width: 480, height: 489 },
  },
  {
    slug: 'rubik',
    instagramHandle: '@wirsindrubik',
    instagramUrl: 'https://www.instagram.com/wirsindrubik/',
    hasDescription: true,
    accent: 'info',
  },
  {
    slug: 'singingLittleBigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'success',
  },
  {
    slug: 'bademeister',
    instagramHandle: '@bademeister.band',
    instagramUrl: 'https://www.instagram.com/bademeister.band',
    hasDescription: true,
    accent: 'accent',
  },
  {
    slug: 'fourfold',
    instagramHandle: '@fourfoldband',
    instagramUrl: 'https://www.instagram.com/fourfoldband/',
    hasDescription: true,
    accent: 'info',
  },
  {
    slug: 'bigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'success',
  },
  {
    slug: 'jazzCombo',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
    accent: 'accent',
  },
]
