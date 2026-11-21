<script setup lang="ts">
/**
 * The lit stage behind the home page's hero: the photograph, the haze on
 * the stage floor, four sweeping light beams and the scrim over all of
 * them. Ported from old_simon-ostendorf.de's .stage/.beam rules, with one
 * rule added that the old site never needed.
 *
 * That rule is the layer order. The old site was dark only, so white text
 * on a darkened photograph was safe by construction. This hero is dark in
 * both of this theme's modes, which gets the photograph back to full
 * strength, but it still does not lean on the photograph being dark: the
 * scrim is the last decorative layer, over the photo and over the beams
 * alike, and the hero's text sits on nothing else. What is behind the
 * scrim then cannot affect the hero's contrast at all, whatever
 * photograph is in it; see --hero-scrim-alpha in src/theme/vtech.css and
 * the hero half of src/theme/contrast.test.ts for the number and its
 * proof.
 *
 * Every colour below comes from a theme token, and .hero pins
 * color-scheme: dark, so all of them resolve to their dark values here
 * however the rest of the page is painted.
 *
 * That only describes the ground the hero's text reads on if the text
 * paints after this whole element, which is what --hero-layer-stage below
 * and --hero-layer-content on the hero's own content are for: this is a
 * positioned element, so without an explicit z-index it would paint over
 * static in-flow content regardless of document order. The test that
 * holds those two apart is src/views/heroLayers.test.ts.
 *
 * Decorative throughout, so the photograph carries an empty alt and the
 * whole element is hidden from assistive technology: every word the hero
 * says is in the markup next to it.
 */
import { HERO_INTRINSIC, HERO_SIZES, heroAvif, heroFallback, heroWebp } from '../lib/heroImage'
</script>

<template>
  <div class="hero-stage" aria-hidden="true">
    <picture>
      <source type="image/avif" :srcset="heroAvif" :sizes="HERO_SIZES" />
      <source type="image/webp" :srcset="heroWebp" :sizes="HERO_SIZES" />
      <!-- The Largest Contentful Paint element: fetched at high priority,
           preloaded from the prerendered <head>, and given its own pixel
           size so the box never moves once it lands. -->
      <img
        class="hero-photo"
        :src="heroFallback"
        :width="HERO_INTRINSIC.width"
        :height="HERO_INTRINSIC.height"
        alt=""
        fetchpriority="high"
        decoding="async"
      />
    </picture>

    <div class="hero-haze"></div>

    <div class="hero-beams">
      <span class="hero-beam"></span>
      <span class="hero-beam"></span>
      <span class="hero-beam"></span>
      <span class="hero-beam"></span>
    </div>

    <div class="hero-scrim"></div>
  </div>
</template>

<style scoped>
.hero-stage {
  position: absolute;
  z-index: var(--hero-layer-stage);
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.hero-photo {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: 50% 42%;
  filter: var(--hero-photo-filter);
}

/* Haze glowing on the stage floor, as on the old site. */
.hero-haze {
  position: absolute;
  inset: 0;
  mix-blend-mode: var(--hero-beam-blend);
  background:
    radial-gradient(
      ellipse 90% 45% at 50% 105%,
      color-mix(in srgb, var(--color-accent-text) 36%, transparent),
      transparent 70%
    ),
    radial-gradient(
      ellipse 60% 30% at 50% 100%,
      color-mix(in srgb, var(--color-accent) 18%, transparent),
      transparent 70%
    );
}

.hero-beams {
  position: absolute;
  inset: 0;
}

/*
 * A beam is a soft cone hanging from above, swaying between --from and
 * --to. The cone sits on ::before because a clip-path on the blurred
 * element would cut the blur off again.
 */
.hero-beam {
  position: absolute;
  top: -12%;
  width: 36vmax;
  height: 125%;
  margin-left: -18vmax;
  transform-origin: 50% 0;
  transform: rotate(var(--from));
  filter: blur(16px);
  mix-blend-mode: var(--hero-beam-blend);
  opacity: 0.75;
  will-change: transform;
  /* Each beam sways once, from --from to --to in under five seconds, and
     stays where it ends: moving content that starts by itself and goes on
     longer than that needs a pause control (WCAG 2.2.2), and decoration is
     not exempt. */
  animation: hero-sway 4s ease-in-out 1 both;
}

.hero-beam::before {
  content: '';
  position: absolute;
  inset: 0;
  clip-path: polygon(47% 0, 53% 0, 100% 100%, 0 100%);
  background: linear-gradient(to bottom, var(--beam) 0%, transparent 80%);
}

.hero-beam:nth-child(1) {
  left: 8%;
  --beam: color-mix(in srgb, var(--color-accent) 60%, transparent);
  --from: -28deg;
  --to: -8deg;
}

.hero-beam:nth-child(2) {
  left: 34%;
  --beam: color-mix(in srgb, var(--color-accent-text) 70%, transparent);
  --from: 14deg;
  --to: -10deg;
  animation-duration: 5s;
}

.hero-beam:nth-child(3) {
  left: 66%;
  --beam: color-mix(in srgb, var(--color-accent) 50%, transparent);
  --from: -12deg;
  --to: 14deg;
  animation-duration: 4.5s;
}

.hero-beam:nth-child(4) {
  left: 92%;
  --beam: color-mix(in srgb, var(--color-accent-text) 70%, transparent);
  --from: 28deg;
  --to: 8deg;
  animation-duration: 3.5s;
}

@keyframes hero-sway {
  from {
    transform: rotate(var(--from));
  }

  to {
    transform: rotate(var(--to));
  }
}

/*
 * The scrim. Its weakest stop is --hero-scrim-alpha, which is the one
 * number the hero's whole contrast guarantee rests on; it only ever gets
 * more opaque further down, where it reaches the hero's own floor.
 */
.hero-scrim {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    to bottom,
    color-mix(in srgb, var(--hero-scrim-color) calc(var(--hero-scrim-alpha) * 100%), transparent) 0%,
    color-mix(in srgb, var(--hero-scrim-color) calc(var(--hero-scrim-alpha) * 100%), transparent)
      62%,
    var(--hero-scrim-color) 100%
  );
}

@media (prefers-reduced-motion: reduce) {
  .hero-beam {
    animation: none;
  }
}
</style>
