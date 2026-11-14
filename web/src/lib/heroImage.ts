/**
 * The stage photograph behind the home page's hero, and the portrait in
 * Über mich. Both are the owner's own pictures, carried over from
 * old_simon-ostendorf.de, where each shipped as one large JPEG.
 *
 * Here each one ships as AVIF and WebP at several widths with a JPEG
 * fallback for a browser that reads neither, generated once from the
 * original and committed; see AGENTS.md's "Generated files" for the
 * command. Vite content hashes every file below, which is what lets the
 * service answer /assets/ with an immutable cache.
 *
 * The widths are listed next to their imports rather than parsed back out
 * of the file names, so a srcset can never claim a width the file does
 * not have.
 */
import hero640Avif from '../assets/hero-640.avif'
import hero960Avif from '../assets/hero-960.avif'
import hero1280Avif from '../assets/hero-1280.avif'
import hero1600Avif from '../assets/hero-1600.avif'
import hero1920Avif from '../assets/hero-1920.avif'
import hero2400Avif from '../assets/hero-2400.avif'
import hero640Webp from '../assets/hero-640.webp'
import hero960Webp from '../assets/hero-960.webp'
import hero1280Webp from '../assets/hero-1280.webp'
import hero1600Webp from '../assets/hero-1600.webp'
import hero1920Webp from '../assets/hero-1920.webp'
import hero2400Webp from '../assets/hero-2400.webp'
import heroJpeg from '../assets/hero-1280.jpg'
import portrait460Avif from '../assets/portrait-460.avif'
import portrait920Avif from '../assets/portrait-920.avif'
import portrait460Webp from '../assets/portrait-460.webp'
import portrait920Webp from '../assets/portrait-920.webp'
import portraitJpeg from '../assets/portrait-460.jpg'

function srcset(variants: Array<[url: string, width: number]>): string {
  return variants.map(([url, width]) => `${url} ${width}w`).join(', ')
}

/** The original's own pixel size. The <img> carries it so the hero
 *  reserves its box before a single byte of the picture has arrived. */
export const HERO_INTRINSIC = { width: 2400, height: 1800 } as const

/** The hero spans the full window at every width, so the browser needs no
 *  more than that to choose a candidate. */
export const HERO_SIZES = '100vw'

export const heroAvif = srcset([
  [hero640Avif, 640],
  [hero960Avif, 960],
  [hero1280Avif, 1280],
  [hero1600Avif, 1600],
  [hero1920Avif, 1920],
  [hero2400Avif, 2400],
])

export const heroWebp = srcset([
  [hero640Webp, 640],
  [hero960Webp, 960],
  [hero1280Webp, 1280],
  [hero1600Webp, 1600],
  [hero1920Webp, 1920],
  [hero2400Webp, 2400],
])

export const heroFallback = heroJpeg

/** The box the portrait is laid out in, for the <img>'s own width and
 *  height attributes. The source file is square, 1200x1200, but
 *  .portrait img crops it to 4 / 5 with object-fit, so the file's own
 *  size would have the browser reserve a box the page never uses. These
 *  two numbers carry that 4 / 5 instead, which is the only thing the
 *  attributes are read for. */
export const PORTRAIT_BOX = { width: 1200, height: 1500 } as const

/** The portrait is capped at 460 CSS pixels by the Über mich layout. */
export const PORTRAIT_SIZES = '(min-width: 920px) 460px, min(100vw - 2rem, 460px)'

export const portraitAvif = srcset([
  [portrait460Avif, 460],
  [portrait920Avif, 920],
])

export const portraitWebp = srcset([
  [portrait460Webp, 460],
  [portrait920Webp, 920],
])

export const portraitFallback = portraitJpeg
