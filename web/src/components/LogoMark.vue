<script setup lang="ts">
import { useId } from 'vue'

/**
 * The fader mark alone, ported from the {{define "logo"}} block of
 * old_simon-ostendorf.de's cmd/website/templates/layout.html: five mixer
 * channel faders, four knobs cut out of the tracks they sit between.
 *
 * The gradient stops are the theme's two logo tokens, --logo-from and
 * --logo-to, read through the stop-color rule in <style> below. They are
 * the favicon's cyan and violet and are the same in light and dark mode,
 * independent of the palette's accent roles: the logo is the brand's own
 * mark, not a role coloured element. That rule has to live in a stylesheet and not an
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
  stop-color: var(--logo-from);
}

.logo-mark-stop-end {
  stop-color: var(--logo-to);
}

/*
 * Knob and cut-out carry the same class, so the dot and the hole it sits
 * in travel together; the distances keep every cut-out inside the two
 * tracks it bridges, which is what the old site's own travel values were
 * chosen for.
 *
 * The slide plays once and then rests, and never runs longer than five
 * seconds: moving content that starts by itself and goes on longer than
 * that needs a pause control (WCAG 2.2.2), and a pause button on a logo
 * would be worse than a logo that comes to rest. A looping version is
 * what this replaced.
 */
.logo-mark-animated .logo-knob {
  animation: logo-fader 4s ease-in-out 1;
}

/*
 * Only the travel differs per knob. The old rules also carried a duration
 * and a delay each, but the shared rule above sets the animation shorthand
 * with higher specificity, so those never applied and all four knobs have
 * always slid together over 4 seconds. They are gone rather than made
 * live, so the animation looks as it always did.
 */
.knob-1 {
  --travel: 5px;
}

.knob-2 {
  --travel: -6px;
}

.knob-3 {
  --travel: 6px;
}

.knob-4 {
  --travel: -5px;
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
