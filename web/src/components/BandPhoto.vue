<script setup lang="ts">
/**
 * A band's square profile picture or photo, when the data file carries one.
 * Renders nothing for a band without one. Width and height are the file's
 * pixel size, so the browser reserves the space before the image loads.
 *
 * Unlike BandLogo it has no plate: a photo fills its own square, with the
 * same 14px corners as the Instagram tiles.
 */
import type { BandPhoto } from '../data/bands'

defineProps<{
  photo?: BandPhoto
  /** The accessible name of the picture, in the page's language. */
  alt: string
}>()
</script>

<template>
  <img
    v-if="photo"
    class="band-photo"
    :src="photo.src"
    :alt="alt"
    :width="photo.width"
    :height="photo.height"
    loading="lazy"
    decoding="async"
  />
</template>

<style scoped>
.band-photo {
  display: block;
  align-self: flex-start;
  width: 7rem;
  height: 7rem;
  margin-bottom: 0.75rem;
  border-radius: 14px;
  object-fit: cover;
}
</style>
