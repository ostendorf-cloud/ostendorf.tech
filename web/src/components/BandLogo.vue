<script setup lang="ts">
/**
 * A band's own logo, when the data file carries one. Renders nothing for a
 * band without one. Width and height are the file's pixel size, so the
 * browser reserves the space before the image loads and nothing shifts.
 */
import type { BandLogo } from '../data/bands'

defineProps<{ logo?: BandLogo }>()
</script>

<template>
  <img
    v-if="logo"
    class="band-logo"
    :src="logo.src"
    :alt="logo.alt"
    :width="logo.width"
    :height="logo.height"
    loading="lazy"
    decoding="async"
  />
</template>

<style scoped>
.band-logo {
  max-width: 100%;
  height: auto;
  max-height: 4rem;
  width: auto;
  object-fit: contain;
  align-self: flex-start;
  margin-bottom: 0.75rem;
}
</style>
