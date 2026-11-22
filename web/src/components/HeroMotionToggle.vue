<script setup lang="ts">
/**
 * The pause control for the hero's light beams.
 *
 * The beams sway for as long as the page is open, as on the old site, so
 * WCAG 2.2.2 asks for a way to stop them. This is that way: a real button
 * whose label flips between "Pause animation" and "Start animation". It
 * carries no aria-pressed, because the label itself says what the next
 * press does, and it remembers nothing: a reload starts the beams again.
 *
 * Under prefers-reduced-motion the beams do not move at all, so the button
 * is hidden there by the stylesheet below, which also takes it out of the
 * tab order.
 */
import { useI18n } from 'vue-i18n'

defineProps<{ paused: boolean }>()
defineEmits<{ toggle: [] }>()

const { t } = useI18n()
</script>

<template>
  <button type="button" class="hero-motion-toggle" @click="$emit('toggle')">
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true" focusable="false">
      <path v-if="paused" d="M7 4.5v15l12-7.5z" fill="currentColor" />
      <path v-else d="M6.5 4.5h4v15h-4zM13.5 4.5h4v15h-4z" fill="currentColor" />
    </svg>
    <span>{{ paused ? t('home.hero.startMotion') : t('home.hero.pauseMotion') }}</span>
  </button>
</template>

<style scoped>
.hero-motion-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 2.75rem;
  min-height: 2.75rem;
  padding-inline: 0.875rem;
  border: 1px solid var(--color-border-strong);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-weight: 600;
  cursor: pointer;
  transition: background-color var(--default-transition-duration) var(--ease-out);
}

.hero-motion-toggle:hover {
  background: var(--color-surface-hover);
}

/* On a phone the label would run into the scroll cue in the middle, so
   only the glyph stays on screen; the label is still the button's name. */
@media (width < 40rem) {
  .hero-motion-toggle {
    justify-content: center;
    width: 2.75rem;
    padding-inline: 0;
  }

  .hero-motion-toggle span {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-motion-toggle {
    display: none;
  }
}
</style>
