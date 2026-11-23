<script setup lang="ts">
import { onMounted, ref, useId } from 'vue'

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
    /** Lets the four knobs slide in their tracks, each at its own tempo and
     *  without end, the way the old site's own hero and Über mich logos
     *  did. Off by default, so the header and the footer stay calm. The
     *  slide starts once the script has run, because only then can the
     *  visitor stop it; the prerendered mark is at rest. */
    animated?: boolean
    /** Freezes the knobs where they are: the visitor's own pause button
     *  (WCAG 2.2.2), the same state that freezes the hero's beams. */
    paused?: boolean
  }>(),
  { size: 32, animated: false, paused: false },
)

// Set in onMounted, so the server render and the first client render agree
// and the knobs never move where no pause button exists.
const live = ref(false)
onMounted(() => {
  live.value = true
})

const gradientId = `logo-mark-gradient-${useId()}`
const maskId = `logo-mark-cut-${useId()}`
</script>

<template>
  <svg
    viewBox="0 0 79.8 80.6"
    :width="size"
    :height="size"
    :class="{ 'logo-mark-animated': animated && live, 'is-paused': paused }"
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
 * The slide goes on for as long as the page is open, as on the old site.
 * Moving content that starts by itself and goes on longer than five
 * seconds needs a way to stop it (WCAG 2.2.2): that is the hero's pause
 * button, which sets .is-paused through the paused prop. The class is only
 * there once the script has run, the same moment the button appears.
 */
.logo-mark-animated .logo-knob {
  animation: logo-fader 4s ease-in-out infinite;
}

.logo-mark-animated.is-paused .logo-knob {
  animation-play-state: paused;
}

/*
 * Each knob has its own travel, tempo and phase, as on the old site. The
 * duration and delay sit under .logo-mark-animated so they win over the
 * shorthand above; set on a bare .knob-N they would be overridden and
 * never apply, which is how the knobs once came to slide in lockstep.
 */
.logo-mark-animated .knob-1 {
  --travel: 5px;

  animation-duration: 4.2s;
  animation-delay: -0.6s;
}

.logo-mark-animated .knob-2 {
  --travel: -6px;

  animation-duration: 5.1s;
  animation-delay: -2.4s;
}

.logo-mark-animated .knob-3 {
  --travel: 6px;

  animation-duration: 3.7s;
  animation-delay: -1.3s;
}

.logo-mark-animated .knob-4 {
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
