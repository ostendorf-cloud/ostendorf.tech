<script setup lang="ts">
/**
 * Each way of getting in touch is a channel strip, as on the old site:
 * the icon, the label and the value, and a level meter at the right that
 * idles low and jumps when the channel is picked up.
 *
 * The old strip also carried a channel number, 01 to 05, in a small box
 * at the left. That is left out. It is the same device as the CH 01 and
 * CH 02 eyebrows the section headers used to wear, which were dropped for
 * reading as a decoration rather than as information, and a number that
 * tells a visitor nothing does not get to survive just because it is
 * smaller here. The meter stays, because it is not decoration of that
 * kind: a level meter next to a way of reaching a sound engineer says
 * what the trade is, and it moves only when the visitor is already on the
 * row.
 */
import { useI18n } from 'vue-i18n'

import ChannelIcon from './ChannelIcon.vue'
import { channels } from '../data/channels'

const { t } = useI18n()
</script>

<template>
  <ul class="flex flex-col gap-3" role="list" :aria-label="t('home.contact.channelsLabel')">
    <li v-for="channel in channels" :key="channel.key">
      <a
        :href="channel.href"
        :target="channel.external ? '_blank' : undefined"
        :rel="channel.external ? 'noopener noreferrer' : undefined"
        class="strip"
      >
        <ChannelIcon :name="channel.key" class="text-accent-text" />
        <span class="flex min-w-0 flex-col">
          <span class="text-sm font-medium text-text-muted">
            {{ t(`home.contact.channels.${channel.key}.label`) }}
          </span>
          <span class="truncate text-base text-text">
            {{ t(`home.contact.channels.${channel.key}.value`) }}
          </span>
        </span>
        <span class="meter" aria-hidden="true">
          <span></span>
          <span></span>
          <span></span>
          <span></span>
          <span></span>
        </span>
      </a>
    </li>
  </ul>
</template>

<style scoped>
.strip {
  display: flex;
  min-height: calc(var(--spacing) * 11);
  align-items: center;
  gap: calc(var(--spacing) * 3);
  padding: calc(var(--spacing) * 4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background-color: var(--color-surface);
  box-shadow: var(--shadow-card);
  color: var(--color-text);
  transition:
    background-color var(--default-transition-duration) var(--ease-out),
    border-color var(--default-transition-duration) var(--ease-out);
}

/* Picked up: the brand gradient takes over the border, the way a lit
   channel does on a desk. */
.strip:hover,
.strip:focus-visible {
  border-color: transparent;
  background:
    linear-gradient(var(--color-surface-hover), var(--color-surface-hover)) padding-box,
    linear-gradient(90deg, var(--brand-gradient-from), var(--brand-gradient-to)) border-box;
}

.meter {
  display: flex;
  height: calc(var(--spacing) * 5.5);
  flex-shrink: 0;
  align-items: flex-end;
  gap: 3px;
  margin-left: auto;
}

.meter span {
  width: 3px;
  height: 100%;
  border-radius: 2px;
  opacity: 0.45;
  transform: scaleY(0.25);
  transform-origin: 50% 100%;
  transition:
    transform var(--default-transition-duration) var(--ease-out),
    opacity var(--default-transition-duration) var(--ease-out);
}

/* Five steps along the brand gradient rather than five values of their
   own, so the meter follows the theme like everything else does. */
.meter span:nth-child(1) {
  background: var(--brand-gradient-from);
}

.meter span:nth-child(2) {
  background: color-mix(in srgb, var(--brand-gradient-from) 75%, var(--brand-gradient-to));
  transform: scaleY(0.35);
}

.meter span:nth-child(3) {
  background: color-mix(in srgb, var(--brand-gradient-from) 50%, var(--brand-gradient-to));
}

.meter span:nth-child(4) {
  background: color-mix(in srgb, var(--brand-gradient-from) 25%, var(--brand-gradient-to));
  transform: scaleY(0.3);
}

.meter span:nth-child(5) {
  background: var(--brand-gradient-to);
}

.strip:hover .meter span,
.strip:focus-visible .meter span {
  opacity: 1;
  animation: level 700ms ease-in-out infinite alternate;
}

.strip:hover .meter span:nth-child(2),
.strip:focus-visible .meter span:nth-child(2) {
  animation-duration: 520ms;
  animation-delay: -200ms;
}

.strip:hover .meter span:nth-child(3),
.strip:focus-visible .meter span:nth-child(3) {
  animation-duration: 640ms;
  animation-delay: -350ms;
}

.strip:hover .meter span:nth-child(4),
.strip:focus-visible .meter span:nth-child(4) {
  animation-duration: 460ms;
  animation-delay: -100ms;
}

.strip:hover .meter span:nth-child(5),
.strip:focus-visible .meter span:nth-child(5) {
  animation-duration: 580ms;
  animation-delay: -420ms;
}

@keyframes level {
  from {
    transform: scaleY(0.3);
  }

  to {
    transform: scaleY(1);
  }
}

/* The desk holds its levels: the meter still lights up, it just stops
   moving. */
@media (prefers-reduced-motion: reduce) {
  .strip {
    transition: none;
  }

  /* The base transition as well as the hover one: clearing only the
     hover rule still animates the five bars back down on the way out. */
  .meter span {
    transition: none;
  }

  .strip:hover .meter span,
  .strip:focus-visible .meter span {
    animation: none;
    transform: scaleY(1);
    transition: none;
  }
}
</style>
