<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

import type { Fact } from '../lib/facts'
import { prefersReducedMotion } from '../lib/motion'

/**
 * A list of facts (years of experience, number of disciplines, ...) that
 * count up from zero to their final value once, the first time the list
 * scrolls into view. The initial display is already the final value, not
 * zero: that is what the prerenderer writes and what a reader with
 * JavaScript disabled sees, and the animation below starts from it rather
 * than creating it.
 */
const props = defineProps<{ facts: Fact[] }>()

const displayValues = ref<number[]>(props.facts.map((fact) => fact.target))

const rootEl = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null
let hasCounted = false

/** The whole list animates as one motion: a single clock drives every
 *  fact's value from zero to its own target over the same duration. */
const DURATION_MS = 600

function animateCount(): void {
  const targets = props.facts.map((fact) => fact.target)
  displayValues.value = targets.map(() => 0)

  const start = performance.now()
  function tick(now: number): void {
    const progress = Math.min((now - start) / DURATION_MS, 1)
    displayValues.value = targets.map((target) => Math.round(target * progress))
    if (progress < 1) {
      requestAnimationFrame(tick)
    }
  }
  requestAnimationFrame(tick)
}

function onIntersect(entries: IntersectionObserverEntry[]): void {
  if (hasCounted) {
    return
  }
  const entry = entries[0]
  if (!entry?.isIntersecting) {
    return
  }
  hasCounted = true
  observer?.disconnect()
  if (!prefersReducedMotion()) {
    animateCount()
  }
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined' || !rootEl.value) {
    return
  }
  observer = new IntersectionObserver(onIntersect)
  observer.observe(rootEl.value)
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<template>
  <ul ref="rootEl" role="list">
    <li v-for="(fact, index) in facts" :key="fact.label" class="flex flex-col">
      <span class="text-2xl font-semibold text-accent-text"
        >{{ displayValues[index] }}{{ fact.suffix ?? '' }}</span
      >
      <span class="text-sm text-text-muted">{{ fact.label }}</span>
    </li>
  </ul>
</template>
