/** The attribute on the root element that names the ground the page shows. */
export const SECTION_BG_ATTRIBUTE = 'data-section-bg'

/** Marks a section that lies on the raised ground: `data-section-tone="alt"`. */
export const SECTION_TONE_SELECTOR = '[data-section-tone="alt"]'

/** Where in the window the ground is chosen, as a share of its height. */
export const PROBE_RATIO = 0.5

export type SectionTone = 'base' | 'alt'

export interface SectionBox {
  top: number
  bottom: number
}

/**
 * sectionToneAt picks the ground for the line at `probe` (pixels from the top
 * of the window): raised while the line crosses an alt section, the page's own
 * ground everywhere else, which includes the hero, the gaps and the footer.
 */
export function sectionToneAt(alt: readonly SectionBox[], probe: number): SectionTone {
  return alt.some((box) => box.top <= probe && probe < box.bottom) ? 'alt' : 'base'
}

/**
 * trackSectionBackground keeps `data-section-bg` on the root element equal to
 * the tone under the probe line. Only a custom property of the root changes,
 * never a size, so nothing on the page moves; the stylesheet turns the value
 * into a colour change. It returns the function that stops tracking and
 * leaves the root as it found it.
 */
export function trackSectionBackground(doc: Document = document): () => void {
  const root = doc.documentElement
  const view = doc.defaultView
  if (!view) {
    return () => {}
  }

  const apply = (): void => {
    const boxes = Array.from(doc.querySelectorAll<HTMLElement>(SECTION_TONE_SELECTOR)).map(
      (element) => element.getBoundingClientRect(),
    )
    const tone = sectionToneAt(boxes, view.innerHeight * PROBE_RATIO)
    if (root.getAttribute(SECTION_BG_ATTRIBUTE) !== tone) {
      root.setAttribute(SECTION_BG_ATTRIBUTE, tone)
    }
  }

  let frame = 0
  const schedule = (): void => {
    if (frame !== 0) {
      return
    }
    frame = view.requestAnimationFrame(() => {
      frame = 0
      apply()
    })
  }

  apply()
  view.addEventListener('scroll', schedule, { passive: true })
  view.addEventListener('resize', schedule)
  // A section that renders late (the feed) changes what is under the line
  // without a scroll; the page's own size changing is the signal for it.
  const observer = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(schedule)
  observer?.observe(doc.body)

  return () => {
    view.removeEventListener('scroll', schedule)
    view.removeEventListener('resize', schedule)
    observer?.disconnect()
    if (frame !== 0) {
      view.cancelAnimationFrame(frame)
    }
    root.removeAttribute(SECTION_BG_ATTRIBUTE)
  }
}
