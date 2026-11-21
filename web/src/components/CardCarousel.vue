<script setup lang="ts" generic="T">
import { onBeforeUnmount, onMounted, ref } from 'vue'
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
 * There is no autoplay and nothing moves unless the visitor asks.
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

let observer: ResizeObserver | undefined

onMounted(() => {
  mounted.value = true
  update()
  if (typeof ResizeObserver !== 'undefined' && track.value) {
    observer = new ResizeObserver(update)
    observer.observe(track.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
})
</script>

<template>
  <div role="group" aria-roledescription="carousel" :aria-label="label" class="carousel">
    <div v-if="mounted" class="carousel-controls">
      <GovueButton
        class="carousel-arrow"
        :aria-label="previousLabel"
        :title="previousLabel"
        :aria-disabled="atStart ? 'true' : 'false'"
        data-direction="previous"
        @click="scrollPage(-1)"
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
        @click="scrollPage(1)"
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
      @scroll.passive="update"
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
