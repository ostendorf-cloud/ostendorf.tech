/**
 * prefersReducedMotion reports the visitor's own operating system
 * preference. Both the circular theme reveal and the counting facts check
 * this before animating anything, matchMedia missing entirely (an old
 * browser, or a test environment) is treated the same as "no preference
 * expressed", so the animation still runs.
 */
export function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}
