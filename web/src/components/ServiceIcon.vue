<script setup lang="ts">
/**
 * The six Leistungen icons, ported stroke for stroke from
 * old_simon-ostendorf.de's cmd/website/templates/home.html: a fader bank,
 * a microphone, a speaker, a moving head, a camera and a flight case, one
 * per entry of home.services.items and in that same order.
 *
 * Decorative: every icon sits next to the name it illustrates, so it is
 * hidden from assistive technology rather than given a label of its own.
 */
import type { ServiceIconName } from '../data/services'

defineProps<{ name: ServiceIconName }>()
</script>

<template>
  <svg
    viewBox="0 0 24 24"
    width="24"
    height="24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.7"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <g v-if="name === 'faders'">
      <path d="M6 3v18M12 3v18M18 3v18" />
      <rect x="4" y="13" width="4" height="3.5" rx="1" />
      <rect x="10" y="6" width="4" height="3.5" rx="1" />
      <rect x="16" y="15" width="4" height="3.5" rx="1" />
    </g>
    <g v-else-if="name === 'microphone'">
      <rect x="9" y="2.5" width="6" height="11" rx="3" />
      <path d="M5.5 11a6.5 6.5 0 0 0 13 0M12 17.5v4M8.5 21.5h7" />
    </g>
    <g v-else-if="name === 'speaker'">
      <rect x="5" y="2.5" width="14" height="19" rx="2.5" />
      <circle cx="12" cy="14.5" r="3.8" />
      <circle cx="12" cy="7" r="1.4" />
    </g>
    <g v-else-if="name === 'movingHead'">
      <path d="M8 3h8l-1.5 7h-5z" />
      <path d="M9.5 10 5 21.5M14.5 10l4.5 11.5M12 10v11.5" />
    </g>
    <g v-else-if="name === 'camera'">
      <rect x="2.5" y="6.5" width="13" height="11" rx="2" />
      <path d="m15.5 10.5 6-3.5v10l-6-3.5z" />
    </g>
    <g v-else-if="name === 'case'">
      <rect x="3" y="7" width="18" height="13" rx="2" />
      <path d="M9 7V4.5h6V7M3 12.5h18M10.5 12.5v2h3v-2" />
    </g>
  </svg>
</template>
