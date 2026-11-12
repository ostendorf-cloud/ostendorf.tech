<script setup lang="ts">
import { useId } from 'vue'

/**
 * The fader mark alone, ported from the {{define "logo"}} block of
 * old_simon-ostendorf.de's cmd/website/templates/layout.html: five mixer
 * channel faders, four knobs cut out of the tracks they sit between.
 *
 * The gradient stops are bound to this theme's own accent tokens through
 * an inline style on each <stop>, rather than a fixed pair of hex values,
 * so the mark recolours itself with the theme and the light/dark mode
 * exactly like every other role coloured element on the page, instead of
 * carrying a colour of its own that could drift from the token contract.
 */
const gradientId = `logo-mark-gradient-${useId()}`
const maskId = `logo-mark-cut-${useId()}`
</script>

<template>
  <svg viewBox="0 0 79.8 80.6" width="32" height="32" aria-hidden="true" focusable="false">
    <defs>
      <linearGradient
        :id="gradientId"
        x1="0"
        y1="0"
        x2="0"
        y2="80.6"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0" style="stop-color: var(--color-accent)" />
        <stop offset="1" style="stop-color: var(--color-accent-text)" />
      </linearGradient>
      <mask :id="maskId" maskUnits="userSpaceOnUse" x="-10" y="-10" width="100" height="101">
        <rect x="-10" y="-10" width="100" height="101" fill="#fff" />
        <circle cx="13.7" cy="45" r="6.45" fill="#000" />
        <circle cx="31.15" cy="62.2" r="6.45" fill="#000" />
        <circle cx="48.6" cy="30.6" r="6.45" fill="#000" />
        <circle cx="66.05" cy="45" r="6.45" fill="#000" />
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
      <circle cx="13.7" cy="45" r="2.3" />
      <circle cx="31.15" cy="62.2" r="2.3" />
      <circle cx="48.6" cy="30.6" r="2.3" />
      <circle cx="66.05" cy="45" r="2.3" />
    </g>
  </svg>
</template>
