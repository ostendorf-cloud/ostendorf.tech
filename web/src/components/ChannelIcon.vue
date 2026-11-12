<script setup lang="ts">
// Stroke paths ported from old_simon-ostendorf.de's cmd/website/templates/layout.html
// icon-mail/icon-phone/icon-whatsapp/icon-instagram/icon-facebook templates,
// recoloured through currentColor instead of a fixed fill so the icon
// follows the surrounding text colour in both themes.
import type { ChannelKey } from '../data/channels'

defineProps<{ name: ChannelKey }>()
</script>

<template>
  <svg
    viewBox="0 0 24 24"
    width="22"
    height="22"
    aria-hidden="true"
    focusable="false"
    class="shrink-0"
  >
    <g v-if="name === 'email'" fill="none" stroke="currentColor" stroke-width="1.6">
      <rect x="2.5" y="5" width="19" height="14" rx="2" />
      <path d="M3.5 6.5 12 13l8.5-6.5" />
    </g>
    <g v-else-if="name === 'phone'" fill="none" stroke="currentColor" stroke-width="1.6">
      <path
        d="M21.5 16.9v2.8a1.9 1.9 0 0 1-2.1 1.9 18.8 18.8 0 0 1-8.2-2.9 18.5 18.5 0 0 1-5.7-5.7 18.8 18.8 0 0 1-2.9-8.2A1.9 1.9 0 0 1 4.5 2.7h2.8a1.9 1.9 0 0 1 1.9 1.6c.1.9.4 1.8.7 2.7a1.9 1.9 0 0 1-.4 2L8.3 10.2a15.2 15.2 0 0 0 5.7 5.7l1.2-1.2a1.9 1.9 0 0 1 2-.4c.9.3 1.8.6 2.7.7a1.9 1.9 0 0 1 1.6 1.9z"
      />
    </g>
    <g v-else-if="name === 'whatsapp'" fill="none" stroke="currentColor" stroke-width="1.6">
      <path d="M12 2.8a9.2 9.2 0 0 0-7.9 13.9L2.8 21.2l4.6-1.2A9.2 9.2 0 1 0 12 2.8z" />
      <path
        d="M9.2 7.8c-.5 0-1.3.6-1.3 1.7 0 2.8 3.8 6.6 6.6 6.6 1.1 0 1.7-.8 1.7-1.3l-.2-.7-1.9-.9-1 1a5.6 5.6 0 0 1-3-3l1-1-.9-1.9z"
      />
    </g>
    <g v-else-if="name === 'instagram'" fill="none" stroke="currentColor" stroke-width="1.6">
      <rect x="2.5" y="2.5" width="19" height="19" rx="5.5" />
      <circle cx="12" cy="12" r="4.2" />
      <circle cx="17.4" cy="6.6" r="1.1" fill="currentColor" stroke="none" />
    </g>
    <g v-else-if="name === 'facebook'" fill="currentColor" stroke="none">
      <path
        d="M17 2.5h-2.8a4.7 4.7 0 0 0-4.7 4.7v3H7v3.8h2.5v8.5h3.8v-8.5h2.9l.8-3.8h-3.7V7.6a1 1 0 0 1 1-1H17z"
      />
    </g>
  </svg>
</template>
