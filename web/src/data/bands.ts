/**
 * The bands and ensembles shown on the home page. Only the facts that do
 * not translate live here: the slug ties a band to its i18n messages under
 * bands.<slug>.name and bands.<slug>.paragraphs, and the Instagram link is
 * the same in both locales.
 *
 * hasDescription is false for a band with no description text yet, in
 * which case only the name and the Instagram link are rendered.
 */
export interface Band {
  slug: string
  instagramHandle: string
  instagramUrl: string
  hasDescription: boolean
}

export const bands: Band[] = [
  {
    slug: 'rubik',
    instagramHandle: '@wirsindrubik',
    instagramUrl: 'https://www.instagram.com/wirsindrubik/',
    hasDescription: true,
  },
  {
    slug: 'bademeister',
    instagramHandle: '@bademeister.band',
    instagramUrl: 'https://www.instagram.com/bademeister.band',
    hasDescription: true,
  },
  {
    slug: 'fourfold',
    instagramHandle: '@fourfoldband',
    instagramUrl: 'https://www.instagram.com/fourfoldband/',
    hasDescription: true,
  },
  {
    slug: 'singingLittleBigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
  },
  {
    slug: 'bigBand',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
  },
  {
    slug: 'jazzCombo',
    instagramHandle: '@fhaachenbigband',
    instagramUrl: 'https://www.instagram.com/fhaachenbigband/',
    hasDescription: true,
  },
  {
    slug: 'herrenjedeck',
    instagramHandle: '@herrenjedeck',
    instagramUrl: 'https://www.instagram.com/herrenjedeck/',
    hasDescription: true,
  },
]
