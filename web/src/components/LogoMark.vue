<script setup lang="ts">
import { useId } from 'vue'

/**
 * The fader mark alone, ported from the {{define "logo"}} block of
 * old_simon-ostendorf.de's cmd/website/templates/layout.html: five mixer
 * channel faders, four knobs cut out of the tracks they sit between.
 *
 * The gradient stops are bound to this theme's own accent tokens through
 * the stop-color rule in <style> below, rather than a fixed pair of hex
 * values, so the mark recolours itself with the theme and the light/dark
 * mode exactly like every other role coloured element on the page,
 * instead of carrying a colour of its own that could drift from the
 * token contract. That rule has to live in a stylesheet and not an
 * inline style attribute on each <stop>: middleware.DefaultSecurity's
 * Content-Security-Policy sets style-src 'self' with no 'unsafe-inline',
 * and a nonce only ever covers an inline <style> or <script> tag, never
 * a style="..." attribute, so a browser enforcing that policy refuses an
 * inline stop-color outright and the gradient disappears.
 */
withDefaults(
  defineProps<{
    /** Edge length in pixels. The header and the footer take the default;
     *  the hero asks for a much larger one. */
    size?: number
    /** Lets the four knobs slide in their tracks, each at its own tempo,
     *  the way the old site's own hero and Über mich logos did. Off by
     *  default, so the header and the footer stay calm. */
    animated?: boolean
  }>(),
  { size: 32, animated: false },
)

const gradientId = `logo-mark-gradient-${useId()}`
const maskId = `logo-mark-cut-${useId()}`
</script>

<template>
  <svg
    viewBox="0 0 79.8 80.6"
    :width="size"
    :height="size"
    :class="{ 'logo-mark-animated': animated }"
    aria-hidden="true"
    focusable="false"
  >
    <defs>
      <linearGradient
        :id="gradientId"
        x1="0"
        y1="0"
        x2="0"
        y2="80.6"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0" class="logo-mark-stop-start" />
        <stop offset="1" class="logo-mark-stop-end" />
      </linearGradient>
      <mask :id="maskId" maskUnits="userSpaceOnUse" x="-10" y="-10" width="100" height="101">
        <rect x="-10" y="-10" width="100" height="101" fill="#fff" />
        <circle class="logo-knob knob-1" cx="13.7" cy="45" r="6.45" fill="#000" />
        <circle class="logo-knob knob-2" cx="31.15" cy="62.2" r="6.45" fill="#000" />
        <circle class="logo-knob knob-3" cx="48.6" cy="30.6" r="6.45" fill="#000" />
        <circle class="logo-knob knob-4" cx="66.05" cy="45" r="6.45" fill="#000" />
      </mask>
    </defs>
    <g :fill="`url(#${gradientId})`">
      <g :mask="`url(#${maskId})`">
        <rect x="0" y="0" width="10" height="61.2" rx="5" />
        <rect x="17.45" y="19.4" width="10" height="61.2" rx="5" />
        <rect x="34.9" y="0" width="10" height="80.6" rx="5" />
        <rect x="52.35" y="0" width="10" height="61.2" rx="5" />
        <rect x="69.8" y="19.4" width="10" height="61.2" rx="5" />
      </g>
      <circle class="logo-knob knob-1" cx="13.7" cy="45" r="2.3" />
      <circle class="logo-knob knob-2" cx="31.15" cy="62.2" r="2.3" />
      <circle class="logo-knob knob-3" cx="48.6" cy="30.6" r="2.3" />
      <circle class="logo-knob knob-4" cx="66.05" cy="45" r="2.3" />
    </g>
  </svg>
</template>

<style scoped>
.logo-mark-stop-start {
  stop-color: var(--color-accent);
}

.logo-mark-stop-end {
  stop-color: var(--color-accent-text);
}

/*
 * Knob and cut-out carry the same class, so the dot and the hole it sits
 * in travel together; the distances keep every cut-out inside the two
 * tracks it bridges, which is what the old site's own travel values were
 * chosen for.
 */
.logo-mark-animated .logo-knob {
  animation: logo-fader 4s ease-in-out infinite;
}

.knob-1 {
  --travel: 5px;
  animation-duration: 4.2s;
  animation-delay: -0.6s;
}

.knob-2 {
  --travel: -6px;
  animation-duration: 5.1s;
  animation-delay: -2.4s;
}

.knob-3 {
  --travel: 6px;
  animation-duration: 3.7s;
  animation-delay: -1.3s;
}

.knob-4 {
  --travel: -5px;
  animation-duration: 4.6s;
  animation-delay: -3.1s;
}

@keyframes logo-fader {
  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(var(--travel));
  }
}

@media (prefers-reduced-motion: reduce) {
  .logo-mark-animated .logo-knob {
    animation: none;
  }
}
</style>
