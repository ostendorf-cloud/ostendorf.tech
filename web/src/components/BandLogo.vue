<script setup lang="ts">
/**
 * A band's own logo, when the data file carries one. Renders nothing for a
 * band without one. Width and height are the file's pixel size, so the
 * browser reserves the space before the image loads and nothing shifts.
 *
 * The logo sits on a light plate. A band's mark is drawn for its own
 * ground, often a dark badge on a transparent background, and would sink
 * into a dark card; --color-accent-on is near white in light and in dark
 * mode, so the plate reads in both. src/theme/contrast.test.ts holds that.
 */
import type { BandLogo } from '../data/bands'

defineProps<{
  logo?: BandLogo
  /** The accessible name of the logo, in the page's language. */
  alt: string
}>()
</script>

<template>
  <div v-if="logo" class="band-logo-plate">
    <img
      class="band-logo"
      :src="logo.src"
      :alt="alt"
      :width="logo.width"
      :height="logo.height"
      loading="lazy"
      decoding="async"
    />
  </div>
</template>

<style scoped>
.band-logo-plate {
  align-self: flex-start;
  margin-bottom: 0.75rem;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 0.75rem;
  background: var(--color-accent-on);
}

.band-logo {
  /* A fixed height with the width following from the file's own aspect
     ratio, which the width and height attributes carry: the card keeps its
     size while the lazy image loads. */
  display: block;
  width: auto;
  max-width: 100%;
  height: 5.5rem;
  object-fit: contain;
}
</style>
