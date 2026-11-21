<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTheme, GovueButton } from '@ostendorf-cloud/govue'

import { prefersReducedMotion } from '../lib/motion'

/**
 * The header's colour mode switch. useTheme's mode has three states,
 * light, dark and system, in that cycling order; this button only ever
 * cycles through them. A visitor who has not chosen yet is on dark, the
 * default main.ts and the prerender script both start useTheme with
 * (src/lib/theme.ts).
 *
 * The icon shows the CHOSEN mode, not the resolved one: a sun for light,
 * a moon for dark and a monitor for "follow the system", so the system
 * choice no longer looks like light or dark. The accessible name states
 * the current mode and what activating the button does.
 *
 * Why not GovueThemeSwitch: it is a radio group of three text buttons,
 * at least about 200 px wide. Next to the wordmark and the language link
 * it does not fit the 328 px of a 360 px phone header in one row, and it
 * has no compact icon variant. That is a govue gap: no compact icon
 * variant of GovueThemeSwitch, see the govue issue to be filed. The button
 * does not use aria-pressed because that is binary and this is a three
 * state cycle; its name carries the state instead.
 */
const cycleOrder = ['light', 'dark', 'system'] as const

const { mode, cycleMode } = useTheme()
const { t } = useI18n()

const nextMode = computed(
  () => cycleOrder[(cycleOrder.indexOf(mode.value) + 1) % cycleOrder.length],
)
const label = computed(() =>
  t('common.theme.switch', {
    current: t(`common.theme.modes.${mode.value}`),
    next: t(`common.theme.modes.${nextMode.value}`),
  }),
)

// The circle's curve mirrors base.css's own --ease-out token: the Web
// Animations API takes a literal easing function, not a CSS var()
// reference, so the same cubic-bezier is written out here by hand.
const REVEAL_EASING = 'cubic-bezier(0.22, 1, 0.36, 1)'
const REVEAL_DURATION_MS = 300

/** activationPoint is the point the reveal grows from. A MouseEvent whose
 *  detail is 0 was synthesized by keyboard activation (Enter or Space)
 *  rather than a real pointer click, and carries no usable coordinates,
 *  so the button's own centre stands in for it. */
function activationPoint(event: MouseEvent): { x: number; y: number } {
  if (event.detail === 0) {
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  }
  return { x: event.clientX, y: event.clientY }
}

/**
 * onToggle replaces the instant mode switch with a circle that expands
 * from the activation point, using the View Transitions API. A browser
 * without it, and a visitor who asked for reduced motion, both get the
 * same instant switch cycleMode always did; what the mode actually means
 * (light, dark, system) is untouched, this only changes how the change is
 * presented.
 */
function onToggle(event: MouseEvent): void {
  if (typeof document.startViewTransition !== 'function' || prefersReducedMotion()) {
    cycleMode()
    return
  }

  const { x, y } = activationPoint(event)
  const endRadius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y),
  )

  const transition = document.startViewTransition(() => {
    cycleMode()
  })

  void transition.ready
    .then(() => {
      document.documentElement.animate(
        { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${endRadius}px at ${x}px ${y}px)`] },
        {
          duration: REVEAL_DURATION_MS,
          easing: REVEAL_EASING,
          pseudoElement: '::view-transition-new(root)',
        },
      )
    })
    .catch((reason: unknown) => {
      // transition.ready rejects when the browser skips the transition, for
      // example a second click arriving before the first one finished.
      // cycleMode above already ran, so the mode change itself is not lost,
      // only its reveal, and that is not worth telling anyone about. A
      // failure from animate() lands here too and is a real defect, so it
      // goes to the console rather than disappearing with the expected case.
      if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
        console.error('theme reveal failed', reason)
      }
    })
}
</script>

<template>
  <GovueButton
    variant="quiet"
    class="min-w-11 px-2!"
    :data-mode="mode"
    :title="label"
    :aria-label="label"
    @click="onToggle"
  >
    <svg
      v-if="mode === 'system'"
      class="size-5 shrink-0"
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
    >
      <rect
        x="3"
        y="4"
        width="18"
        height="12"
        rx="2"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
      />
      <path
        d="M8 20h8M12 16v4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
      />
    </svg>
    <svg
      v-else-if="mode === 'dark'"
      class="size-5 shrink-0"
      viewBox="0 0 24 24"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z"
        fill="none"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <svg v-else class="size-5 shrink-0" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <circle cx="12" cy="12" r="4.5" fill="none" stroke="currentColor" stroke-width="1.7" />
      <path
        d="M12 2.5v2.5M12 19v2.5M4.2 4.2l1.8 1.8M18 18l1.8 1.8M2.5 12H5M19 12h2.5M4.2 19.8 6 18M18 6l1.8-1.8"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linecap="round"
      />
    </svg>
  </GovueButton>
</template>
