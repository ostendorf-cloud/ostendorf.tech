<script setup lang="ts" generic="T">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { GovueButton } from '@ostendorf-cloud/govue'

import { prefersReducedMotion } from '../lib/motion'

/**
 * A row of cards that scrolls sideways: three side by side from 1024 px,
 * two from 768 px, one below that.
 *
 * It is a plain scroll container with CSS scroll snap, so a swipe, the
 * arrow keys, a trackpad and the Tab key all work without this script, and
 * every card stays reachable when it does not run. The two buttons are an
 * addition: each scrolls by one page, the width of the visible row. They
 * use aria-disabled instead of disabled at the ends, so a keyboard user's
 * focus is not thrown out of the control when the last page is reached.
 *
 * The row also advances by itself, one page every AUTO_SCROLL_INTERVAL_MS,
 * and wraps to the start after the last page. It holds still while the
 * pointer is over the carousel, while focus is inside it (the pause button
 * excepted), while a finger or a pressed pointer is on the track, while the
 * page is hidden and while the carousel is scrolled out of view. A wheel
 * scroll restarts the countdown, and no step starts within 150 ms of a
 * scroll event, so a swipe's momentum is never fought. It never runs for a
 * visitor who prefers reduced motion.
 * A pause button stops it for good (WCAG 2.2.2, moving content that lasts
 * longer than five seconds needs a way to pause it). A click on an arrow
 * restarts the countdown, so the row does not jump right after the visitor
 * moved it.
 *
 * What a screen reader hears follows the WAI-ARIA carousel pattern: a
 * group named by `label` with the role description "carousel", and one
 * group per card with the role description "slide", named "1 of N".
 *
 * Buttons are rendered after mount only. A page without the script (the
 * prerendered HTML before hydration, a crawler) would otherwise carry two
 * buttons that do nothing.
 */
const props = defineProps<{
  items: readonly T[]
  /** The accessible name of the whole carousel. */
  label: string
  previousLabel: string
  nextLabel: string
}>()

defineSlots<{
  slide(props: { item: T; index: number }): unknown
}>()

const { t } = useI18n()

const root = ref<HTMLElement | null>(null)
const track = ref<HTMLElement | null>(null)
const mounted = ref(false)
const atStart = ref(true)
const atEnd = ref(true)

/** Sub pixel rounding in scrollLeft and scrollWidth must not leave an
 *  arrow enabled with nowhere to go. */
const EDGE_TOLERANCE = 1

function update(): void {
  const el = track.value
  if (!el) {
    return
  }
  atStart.value = el.scrollLeft <= EDGE_TOLERANCE
  atEnd.value = el.scrollLeft + el.clientWidth >= el.scrollWidth - EDGE_TOLERANCE
}

/** One page: the visible width plus the gap, so the next card lands
 *  exactly on a snap point. */
function pageSize(el: HTMLElement): number {
  const gap = Number.parseFloat(getComputedStyle(el).columnGap)
  return el.clientWidth + (Number.isFinite(gap) ? gap : 0)
}

function scrollPage(direction: -1 | 1): void {
  const el = track.value
  if (!el) {
    return
  }
  if (direction === -1 ? atStart.value : atEnd.value) {
    return
  }
  el.scrollBy({
    left: direction * pageSize(el),
    behavior: prefersReducedMotion() ? 'auto' : 'smooth',
  })
}

/** The time between two automatic page turns. */
const AUTO_SCROLL_INTERVAL_MS = 6000

/** The visitor stopped the movement with the pause button. */
const userPaused = ref(false)
const hovering = ref(false)
const hasFocus = ref(false)
const pageHidden = ref(false)
const reducedMotion = ref(false)
/** False while the carousel is scrolled out of view. Starts true so a
 *  browser without IntersectionObserver keeps the movement. */
const inView = ref(true)
/** A finger is on the track, or a non touch pointer is pressed on it. */
const touching = ref(false)
const pressing = ref(false)

/** A tick never starts while the row is still moving from a swipe's
 *  momentum or a smooth scroll: scrollBy would fight the finger. */
const SCROLL_SETTLE_MS = 150
let lastScrollAt = Number.NEGATIVE_INFINITY

const autoScrolling = computed(
  () =>
    mounted.value &&
    !reducedMotion.value &&
    !userPaused.value &&
    !hovering.value &&
    !hasFocus.value &&
    !pageHidden.value &&
    inView.value &&
    !touching.value &&
    !pressing.value,
)

let timer: ReturnType<typeof setInterval> | undefined

function stopTimer(): void {
  if (timer !== undefined) {
    clearInterval(timer)
    timer = undefined
  }
}

/** Starts the countdown from zero when the carousel is meant to move, so
 *  it also serves as "restart after the visitor's own interaction". */
function startTimer(): void {
  stopTimer()
  if (autoScrolling.value) {
    timer = setInterval(advance, AUTO_SCROLL_INTERVAL_MS)
  }
}

/** One automatic step: the page the next arrow scrolls by, or, on the last
 *  page, back to the start. A row with nothing to scroll stays put. */
function advance(): void {
  const el = track.value
  if (!el || (atStart.value && atEnd.value)) {
    return
  }
  if (Date.now() - lastScrollAt < SCROLL_SETTLE_MS) {
    return
  }
  if (atEnd.value) {
    el.scrollBy({ left: -el.scrollLeft, behavior: 'smooth' })
    return
  }
  scrollPage(1)
}

function onArrow(direction: -1 | 1): void {
  scrollPage(direction)
  startTimer()
}

function togglePause(): void {
  userPaused.value = !userPaused.value
}

/** Focus on the pause button does not stop the movement: after "Start"
 *  the visitor has just asked for it, and the button keeps focus. */
function onFocusIn(event: FocusEvent): void {
  const target = event.target
  const onPause = target instanceof Element && target.closest('[data-action="pause"]') !== null
  hasFocus.value = !onPause
}

function onScroll(): void {
  lastScrollAt = Date.now()
  update()
}

/** A wheel or trackpad scroll over the track restarts the countdown, so a
 *  tick does not land in the middle of it. */
function onWheel(): void {
  startTimer()
}

function onTouchStart(): void {
  touching.value = true
}

function onTouchEnd(): void {
  touching.value = false
}

function onPointerDown(event: PointerEvent): void {
  if (event.pointerType !== 'touch') {
    pressing.value = true
  }
}

function onPointerUp(): void {
  pressing.value = false
}

function onFocusOut(event: FocusEvent): void {
  const root = event.currentTarget as HTMLElement
  if (!(event.relatedTarget instanceof Node) || !root.contains(event.relatedTarget)) {
    hasFocus.value = false
  }
}

/** A touch also fires pointerenter and leaves the pointer "over" the
 *  carousel until the next tap elsewhere, which would stop it for good
 *  without the visitor having asked, so only a mouse or a pen counts. */
function onPointerEnter(event: PointerEvent): void {
  if (event.pointerType !== 'touch') {
    hovering.value = true
  }
}

function onVisibilityChange(): void {
  pageHidden.value = document.hidden
}

function onReducedMotionChange(event: MediaQueryListEvent): void {
  reducedMotion.value = event.matches
}

watch(autoScrolling, startTimer, { flush: 'sync' })

let observer: ResizeObserver | undefined
let visibility: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined

onMounted(() => {
  update()
  if (typeof ResizeObserver !== 'undefined' && track.value) {
    observer = new ResizeObserver(update)
    observer.observe(track.value)
  }
  if (typeof IntersectionObserver !== 'undefined' && root.value) {
    visibility = new IntersectionObserver((entries) => {
      const latest = entries[entries.length - 1]
      inView.value = latest.isIntersecting
    })
    visibility.observe(root.value)
  }
  reducedMotion.value = prefersReducedMotion()
  if (typeof window.matchMedia === 'function') {
    motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    motionQuery.addEventListener?.('change', onReducedMotionChange)
  }
  pageHidden.value = document.hidden
  document.addEventListener('visibilitychange', onVisibilityChange)
  mounted.value = true
})

onBeforeUnmount(() => {
  observer?.disconnect()
  visibility?.disconnect()
  stopTimer()
  document.removeEventListener('visibilitychange', onVisibilityChange)
  motionQuery?.removeEventListener?.('change', onReducedMotionChange)
})
</script>

<template>
  <div
    ref="root"
    role="group"
    aria-roledescription="carousel"
    :aria-label="label"
    class="carousel"
    @pointerenter="onPointerEnter"
    @pointerleave="hovering = false"
    @focusin="onFocusIn"
    @focusout="onFocusOut"
  >
    <div v-if="mounted" class="carousel-controls">
      <!-- Not rendered for a visitor who prefers reduced motion: nothing
           moves for them, so there is nothing to pause. -->
      <GovueButton
        v-if="!reducedMotion"
        class="carousel-arrow carousel-pause"
        :aria-label="userPaused ? t('common.carousel.play') : t('common.carousel.pause')"
        :title="userPaused ? t('common.carousel.play') : t('common.carousel.pause')"
        type="button"
        data-action="pause"
        @click="togglePause"
      >
        <svg
          v-if="userPaused"
          class="size-5 shrink-0"
          viewBox="0 0 24 24"
          aria-hidden="true"
          focusable="false"
        >
          <path d="M8 5l11 7-11 7z" fill="currentColor" />
        </svg>
        <svg
          v-else
          class="size-5 shrink-0"
          viewBox="0 0 24 24"
          aria-hidden="true"
          focusable="false"
        >
          <path d="M7 5h3.5v14H7zM13.5 5H17v14h-3.5z" fill="currentColor" />
        </svg>
      </GovueButton>
      <GovueButton
        class="carousel-arrow"
        :aria-label="previousLabel"
        :title="previousLabel"
        :aria-disabled="atStart ? 'true' : 'false'"
        data-direction="previous"
        @click="onArrow(-1)"
      >
        <svg class="size-5 shrink-0" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path
            d="M15 5l-7 7 7 7"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </GovueButton>
      <GovueButton
        class="carousel-arrow"
        :aria-label="nextLabel"
        :title="nextLabel"
        :aria-disabled="atEnd ? 'true' : 'false'"
        data-direction="next"
        @click="onArrow(1)"
      >
        <svg class="size-5 shrink-0" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path
            d="M9 5l7 7-7 7"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </GovueButton>
    </div>

    <!-- tabindex 0: a scroll container with no focusable content in view
         could otherwise not be scrolled from the keyboard. A focusable
         element needs a role and a name, hence the region. -->
    <div
      ref="track"
      class="carousel-track"
      role="region"
      :aria-label="t('common.carousel.region', { label })"
      tabindex="0"
      @scroll.passive="onScroll"
      @wheel.passive="onWheel"
      @touchstart.passive="onTouchStart"
      @touchend="onTouchEnd"
      @touchcancel="onTouchEnd"
      @pointerdown="onPointerDown"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
    >
      <div
        v-for="(item, index) in props.items"
        :key="index"
        class="carousel-slide"
        role="group"
        aria-roledescription="slide"
        :aria-label="t('common.carousel.slide', { n: index + 1, total: props.items.length })"
      >
        <slot name="slide" :item="item" :index="index" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.carousel {
  --carousel-gap: clamp(1.25rem, 0.85rem + 1.6vw, 2.25rem);
}

.carousel-controls {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.carousel-arrow {
  min-width: 2.75rem;
  padding-inline: 0.5rem;
}

.carousel-arrow[aria-disabled='true'] {
  cursor: not-allowed;
  opacity: 0.5;
}

/* The track keeps a little room above and below: overflow-x makes the
   other axis clip too, and the cards carry a lit edge one pixel above
   themselves. */
.carousel-track {
  display: flex;
  gap: var(--carousel-gap);
  overflow-x: auto;
  padding-block: 0.25rem;
  scroll-snap-type: x mandatory;
  overscroll-behavior-x: contain;
}

.carousel-slide {
  display: grid;
  flex: 0 0 100%;
  min-width: 0;
  scroll-snap-align: start;
}

@media (min-width: 48rem) {
  .carousel-slide {
    flex-basis: calc((100% - var(--carousel-gap)) / 2);
  }
}

@media (min-width: 64rem) {
  .carousel-slide {
    flex-basis: calc((100% - 2 * var(--carousel-gap)) / 3);
  }
}
</style>
