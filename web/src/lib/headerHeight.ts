/** The custom property styles.css reads for `scroll-padding-top`. */
export const HEADER_HEIGHT_PROPERTY = '--header-height'

/**
 * trackHeaderHeight keeps `--header-height` on the root element equal to
 * the real height of the sticky header, so an anchored heading lands below
 * it however many rows the header has at the current width. The property
 * is set through the CSSOM, which a Content-Security-Policy without
 * `unsafe-inline` for styles allows, unlike a style attribute in markup.
 *
 * It returns the function that stops observing. Without ResizeObserver the
 * height is measured once and the stylesheet's own fallback covers the
 * rest.
 */
export function trackHeaderHeight(header: HTMLElement): () => void {
  const root = document.documentElement

  const apply = (): void => {
    const height = header.getBoundingClientRect().height
    if (height > 0) {
      root.style.setProperty(HEADER_HEIGHT_PROPERTY, `${Math.ceil(height)}px`)
    }
  }

  apply()
  if (typeof ResizeObserver === 'undefined') {
    return () => {}
  }
  const observer = new ResizeObserver(apply)
  observer.observe(header)
  return () => {
    observer.disconnect()
    root.style.removeProperty(HEADER_HEIGHT_PROPERTY)
  }
}
