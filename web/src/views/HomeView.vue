<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { GovueCard, GovuePage } from '@ostendorf-cloud/govue'

import CardCarousel from '../components/CardCarousel.vue'
import ContactChannels from '../components/ContactChannels.vue'
import ContactForm from '../components/ContactForm.vue'
import CountingFacts from '../components/CountingFacts.vue'
import HeroStage from '../components/HeroStage.vue'
import InstagramFeed from '../components/InstagramFeed.vue'
import LogoMark from '../components/LogoMark.vue'
import ServiceIcon from '../components/ServiceIcon.vue'
import { bands } from '../data/bands'
import { serviceIcons } from '../data/services'
import { experienceYearsFloor } from '../lib/experienceYears'
import type { Fact } from '../lib/facts'
import {
  PORTRAIT_BOX,
  PORTRAIT_SIZES,
  portraitAvif,
  portraitFallback,
  portraitWebp,
} from '../lib/heroImage'

// Destructuring tm is vue-i18n's own documented usage; unbound-method's
// warning about it does not apply, tm never reads `this`.
// eslint-disable-next-line @typescript-eslint/unbound-method
const { t, tm, rt } = useI18n()

// Annotated so the v-for below sees a real array type and a numeric
// index, which tm's own return type does not provably give it.
const tagline = computed<string[]>(() => tm('home.hero.tagline'))

// Same reason as tagline above: the Leistungen list is paired with
// src/data/services.ts by position, and only an array type gives the
// v-for a numeric index to look an icon up with.
const serviceItems = computed<Array<{ title: string; text: string }>>(() =>
  tm('home.services.items'),
)

// The years figure is computed, never hand-edited; the other two facts
// are fixed counts that still count up alongside it, as one motion. Only
// the label text comes from the message catalogues.
const aboutFacts = computed<Fact[]>(() => [
  {
    label: t('home.about.facts.years.label'),
    target: experienceYearsFloor(new Date()),
    suffix: '+',
  },
  { label: t('home.about.facts.disciplines.label'), target: 4 },
  { label: t('home.about.facts.contact.label'), target: 1 },
])
</script>

<template>
  <GovuePage width="shell">
    <section aria-labelledby="hero-title" class="hero">
      <HeroStage />

      <div class="hero-inner">
        <LogoMark class="hero-logo" :size="104" animated />
        <p class="hero-eyebrow">{{ t('home.hero.eyebrow') }}</p>
        <h1 id="hero-title" class="hero-title">{{ t('home.hero.title') }}</h1>
        <p class="hero-tagline">
          <template v-for="(item, index) in tagline" :key="item">
            <span v-if="index > 0" aria-hidden="true" class="hero-separator">|</span>
            <span>{{ item }}</span>
          </template>
        </p>
        <div class="hero-actions">
          <a
            href="#contact"
            class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-accent-strong px-4 text-base font-medium text-accent-on transition-colors hover:bg-accent-strong-hover"
          >
            {{ t('home.hero.ctaContact') }}
          </a>
          <a
            href="#services"
            class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md border border-border-strong bg-surface px-4 text-base font-medium text-text transition-colors hover:bg-surface-hover"
          >
            {{ t('home.hero.ctaServices') }}
          </a>
        </div>
      </div>

      <!-- There is more below: a mouse outline with a dot rolling down it. -->
      <a href="#services" class="hero-cue">
        <span class="sr-only">{{ t('home.hero.scrollCue') }}</span>
      </a>
    </section>

    <section id="services" aria-labelledby="services-title" class="panel-section">
      <header class="panel-group-header flex flex-col gap-2">
        <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
          {{ t('home.services.eyebrow') }}
        </p>
        <h2 id="services-title" class="text-2xl font-semibold text-text">
          {{ t('home.services.title') }}
        </h2>
        <p class="text-lg text-text-muted">{{ t('home.services.lead') }}</p>
      </header>

      <div class="panel-grid grid md:grid-cols-2">
        <GovueCard class="panel-lit">
          <h3 class="mb-4 text-lg font-semibold text-text">
            {{ t('home.services.panelTechnology') }}
          </h3>
          <ul class="flex flex-col gap-1" role="list">
            <li v-for="(item, index) in serviceItems" :key="rt(item.title)" class="service">
              <span class="service-icon">
                <ServiceIcon :name="serviceIcons[index]" />
              </span>
              <span class="flex min-w-0 flex-col gap-0.5">
                <span class="font-medium text-text">{{ rt(item.title) }}</span>
                <span class="text-sm text-text-muted">{{ rt(item.text) }}</span>
              </span>
            </li>
          </ul>
        </GovueCard>

        <GovueCard class="panel-lit">
          <h3 class="mb-4 text-lg font-semibold text-text">{{ t('home.services.panelAreas') }}</h3>
          <ul class="flex flex-wrap gap-2" role="list">
            <li
              v-for="area in tm('home.services.areas')"
              :key="rt(area)"
              class="rounded-sm bg-bg-subtle px-2 py-1 text-sm text-text"
            >
              {{ rt(area) }}
            </li>
          </ul>
          <p class="mt-auto pt-5 text-sm">
            <a href="mailto:simon@ostendorf.tech" class="text-accent-text hover:underline">
              {{ t('home.services.more') }}
            </a>
          </p>
        </GovueCard>
      </div>
    </section>

    <!-- Renders itself only once the server side feed has something to
         show; see InstagramFeed.vue. -->
    <InstagramFeed />

    <section id="bands" aria-labelledby="bands-title" class="panel-section">
      <header class="panel-group-header flex flex-col gap-2">
        <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
          {{ t('home.bands.eyebrow') }}
        </p>
        <h2 id="bands-title" class="text-2xl font-semibold text-text">
          {{ t('home.bands.title') }}
        </h2>
        <p class="text-lg text-text-muted">{{ t('home.bands.lead') }}</p>
      </header>

      <CardCarousel
        :items="bands"
        :label="t('home.bands.title')"
        :previous-label="t('home.bands.previous')"
        :next-label="t('home.bands.next')"
      >
        <template #slide="{ item: band }">
          <GovueCard class="panel-lit">
            <h3 class="mb-3 text-lg font-semibold text-text">
              {{ t(`bands.${band.slug}.name`) }}
            </h3>
            <div v-if="band.hasDescription" class="flex flex-col gap-2 text-sm text-text-muted">
              <p v-for="(paragraph, index) in tm(`bands.${band.slug}.paragraphs`)" :key="index">
                {{ rt(paragraph) }}
              </p>
            </div>
            <a
              :href="band.instagramUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-auto inline-flex min-h-11 items-center gap-1 self-start pt-3 text-sm text-accent-text hover:underline"
            >
              {{ t('home.bands.instagramLabel') }}: {{ band.instagramHandle }}
            </a>
          </GovueCard>
        </template>
      </CardCarousel>
    </section>

    <section id="about" aria-labelledby="about-title" class="py-8 sm:py-14">
      <div class="grid items-center gap-8 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
        <!-- The gradient is the frame itself: two pixels of padding under
             the picture, so the brand holds the portrait rather than
             sitting next to it. -->
        <figure class="portrait">
          <picture>
            <source type="image/avif" :srcset="portraitAvif" :sizes="PORTRAIT_SIZES" />
            <source type="image/webp" :srcset="portraitWebp" :sizes="PORTRAIT_SIZES" />
            <img
              :src="portraitFallback"
              :width="PORTRAIT_BOX.width"
              :height="PORTRAIT_BOX.height"
              :alt="t('home.about.portraitAlt')"
              loading="lazy"
              decoding="async"
            />
          </picture>
        </figure>

        <div>
          <header class="mb-6 flex flex-col gap-2">
            <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
              {{ t('home.about.eyebrow') }}
            </p>
            <h2 id="about-title" class="text-2xl font-semibold text-text">
              {{ t('home.about.title') }}
            </h2>
          </header>
          <div class="flex flex-col gap-4 text-text-muted">
            <p v-for="(paragraph, index) in tm('home.about.paragraphs')" :key="index">
              {{ rt(paragraph) }}
            </p>
          </div>
          <CountingFacts :facts="aboutFacts" class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3" />
          <p class="mt-6">
            <a
              href="#contact"
              class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-accent-strong px-4 text-base font-medium text-accent-on transition-colors hover:bg-accent-strong-hover"
            >
              {{ t('home.about.cta') }}
            </a>
          </p>
        </div>
      </div>
    </section>

    <section id="contact" aria-labelledby="contact-title" class="py-8 sm:py-14">
      <header class="mb-6 flex flex-col gap-2">
        <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
          {{ t('home.contact.eyebrow') }}
        </p>
        <h2 id="contact-title" class="text-2xl font-semibold text-text">
          {{ t('home.contact.title') }}
        </h2>
        <p class="text-lg text-text-muted">{{ t('home.contact.lead') }}</p>
      </header>

      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)]">
        <ContactChannels />
        <ContactForm />
      </div>
    </section>
  </GovuePage>
</template>

<style scoped>
/*
 * The hero.
 *
 * It breaks out of GovuePage's measure to span the window: a percentage
 * margin resolves against the page's content box, so "50% - 50vw" is
 * exactly the distance from that box's edge to the window's, whatever the
 * page's own padding and maximum width happen to be. The negative top
 * margin eats the page's first row of padding, so the stage starts
 * directly under the header rather than on a strip of page background.
 *
 * It is dark in both modes, and this is the one declaration that makes it
 * so: color-scheme inherits, and light-dark() resolves against the
 * element a token is used on rather than the one it was declared on, so
 * pinning the scheme here gives the stage ground, the headline gradient,
 * the text and both buttons their dark values out of the theme's own
 * palette. No hero rule names a colour of its own. See "the hero stage"
 * in src/theme/vtech.css for why the hero stays dark and what the scrim
 * then has to prove, and src/theme/contrast.test.ts for the proof, which
 * reads this declaration back out of this file.
 *
 * The hero therefore ends on a hard edge against a light mode page
 * instead of fading into it: in dark mode, which is what a visitor gets
 * unless they ask for light, the hero's floor and the page's background
 * are the same colour and there is no edge at all, so softening it would
 * mean drawing a divider across the mode almost everybody reads in to
 * smooth a seam only a light mode visitor ever meets. The band is
 * deliberate; the header above it already sits on its own ground the same
 * way.
 */
.hero {
  color-scheme: dark;
  position: relative;
  isolation: isolate;
  display: grid;
  place-items: center;
  min-height: calc(100svh - var(--spacing-header));
  margin-top: calc(-1 * var(--spacing) * 8);
  margin-inline: calc(50% - 50vw);
  padding-block: clamp(3rem, 9vh, 6rem) clamp(5rem, 13vh, 8rem);
  padding-inline: 1rem;
  overflow: hidden;
  text-align: center;
}

@media (min-width: 40rem) {
  .hero {
    margin-top: calc(-1 * var(--spacing) * 10);
  }
}

/*
 * Both of the hero's own children carry an explicit layer. HeroStage is
 * absolutely positioned, so it would otherwise paint over anything static
 * beside it whatever the markup order says, and the hero-rise animation
 * would hide that for exactly as long as it holds a transform. .hero
 * isolates, so the two numbers are local to the hero and mean nothing
 * outside it; see --hero-layer-stage in src/theme/vtech.css.
 */
.hero-inner {
  position: relative;
  z-index: var(--hero-layer-content);
  display: flex;
  max-width: 48rem;
  flex-direction: column;
  align-items: center;
}

/*
 * The headline is the Largest Contentful Paint candidate for the text, so
 * it paints at once, fully opaque. Only the decoration around it rises in:
 * fading the whole block from opacity 0 held the headline back for the
 * length of the animation.
 */
.hero-logo,
.hero-tagline,
.hero-actions {
  animation: hero-rise 800ms var(--ease-out) both;
}

.hero-logo {
  width: clamp(5rem, 11vw, 7.25rem);
  height: auto;
}

.hero-eyebrow {
  /* Balances the letter spacing after the last letter, so the word stays
     centred under the mark above it. */
  margin: 1.75rem 0 0;
  padding-left: 0.32em;
  color: var(--color-text);
  font-size: clamp(0.8rem, 2.4vw, 0.95rem);
  font-weight: 700;
  letter-spacing: 0.32em;
  text-transform: uppercase;
}

.hero-title {
  margin: 0.75rem 0 0;
  color: var(--brand-gradient-from);
  font-size: clamp(2.3rem, 8.4vw, 6.2rem);
  font-weight: 850;
  letter-spacing: -0.035em;
  line-height: 1.02;
  hyphens: auto;
  overflow-wrap: break-word;
}

/* The gradient needs the text clipped out of it; a browser that cannot do
   that keeps the flat colour above, which is a token in its own right. */
@supports (background-clip: text) or (-webkit-background-clip: text) {
  .hero-title {
    background: linear-gradient(90deg, var(--brand-gradient-from), var(--brand-gradient-to));
    background-clip: text;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
}

/* Forced colours replaces the gradient with the user's own background and
   keeps the transparent fill, which leaves the headline invisible. The
   fill goes back to the text colour the mode itself picked. */
@media (forced-colors: active) {
  .hero-title {
    background: none;
    -webkit-text-fill-color: currentcolor;
    color: canvastext;
  }
}

.hero-tagline {
  display: flex;
  max-width: 47.5rem;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.375rem 0.75rem;
  margin: 1.625rem 0 0;
  color: var(--color-text);
  font-family: var(--font-mono);
  font-size: clamp(0.78rem, 2.2vw, 0.95rem);
}

.hero-separator {
  color: var(--color-accent);
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 2.375rem;
}

/*
 * A mouse outline with a dot rolling down it. The link itself is a 44 by
 * 52 px target (WCAG 2.5.8 asks for 24, this site holds 44 everywhere);
 * the outline is drawn by ::after, centred inside it, at its old 26 by
 * 42 px, and the dot by ::before.
 *
 * The dot rolls twice, 3.6 seconds in all, and then rests: moving content
 * that starts by itself and runs longer than five seconds needs a way to
 * stop it (WCAG 2.2.2), and this one simply ends before that.
 */
.hero-cue {
  position: absolute;
  z-index: var(--hero-layer-content);
  bottom: 1.25rem;
  left: 50%;
  width: 2.75rem;
  height: 3.25rem;
  margin-left: -1.375rem;
}

.hero-cue::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 1.625rem;
  height: 2.625rem;
  margin: -1.3125rem 0 0 -0.8125rem;
  box-sizing: border-box;
  border: 2px solid var(--color-accent);
  border-radius: 0.875rem;
}

.hero-cue::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0.25rem;
  height: 0.5rem;
  margin: -0.875rem 0 0 -0.125rem;
  border-radius: 0.125rem;
  background: var(--color-accent);
  animation: hero-roll 1.8s ease-in-out 2;
}

@media (max-height: 40rem) {
  .hero-cue {
    display: none;
  }
}

@keyframes hero-rise {
  from {
    opacity: 0;
    transform: translateY(0.875rem);
  }
}

@keyframes hero-roll {
  0% {
    transform: translateY(0);
    opacity: 1;
  }

  70% {
    transform: translateY(0.875rem);
    opacity: 0;
  }

  100% {
    transform: translateY(0);
    opacity: 0;
  }
}

/*
 * The two panel groups: Leistungen and Einsatzbereiche above, the bands
 * below. Both are built from GovueCard, whose own padding and radius are
 * sized for a form field or a notice rather than for a panel that carries
 * a whole section of the page, so all four measurements below scale with
 * the viewport instead of holding one value at every width: the padding
 * inside a panel, its corner radius, the gap between panels, and the room
 * the group gets from the heading above it and the sections either side.
 *
 * Every clamp has a real floor rather than a proportional one, so a
 * 360px phone gets a deliberately smaller version of the same spacing
 * instead of a desktop value squeezed into a third of the room.
 */
.panel-section {
  padding-block: clamp(2.5rem, 1.6rem + 3.6vw, 4.75rem);
}

.panel-group-header {
  margin-bottom: clamp(1.75rem, 1.2rem + 2.2vw, 3rem);
}

.panel-grid {
  gap: clamp(1.25rem, 0.85rem + 1.6vw, 2.25rem);
}

/* A thin lit edge along the top of every panel, as on the old site: the
   depth comes from the light, not from a heavier shadow. */
.panel-lit {
  --panel-padding: clamp(1.25rem, 0.95rem + 1.2vw, 2rem);

  position: relative;
  border-radius: clamp(0.875rem, 0.7rem + 0.7vw, 1.25rem);
}

/*
 * GovueCard's own content wrapper, its only child here: neither of these
 * groups fills the header or the footer slot. Made a column so a panel
 * whose neighbour in the row is taller can push its closing link to the
 * bottom edge rather than leaving it floating mid-card.
 */
.panel-lit :deep(> div) {
  display: flex;
  flex-direction: column;
  padding: var(--panel-padding);
}

.panel-lit::before {
  content: '';
  position: absolute;
  top: -1px;
  right: var(--panel-padding);
  left: var(--panel-padding);
  height: 1px;
  background: linear-gradient(90deg, var(--brand-gradient-from), var(--brand-gradient-to));
}

.service {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-inline: -0.5rem;
  padding: 0.5rem;
  border-radius: 0.75rem;
  transition: background-color var(--default-transition-duration) var(--ease-out);
}

.service:hover {
  background-color: var(--color-surface-hover);
}

.service-icon {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 3rem;
  height: 3rem;
  border: 1px solid var(--color-border);
  border-radius: 0.875rem;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--color-accent) 16%, transparent),
    color-mix(in srgb, var(--color-accent-text) 22%, transparent)
  );
  color: var(--color-accent-text);
}

.portrait {
  width: 100%;
  max-width: 28.75rem;
  margin: 0 auto;
  padding: 2px;
  border-radius: 1.625rem;
  background: linear-gradient(90deg, var(--brand-gradient-from), var(--brand-gradient-to));
  box-shadow: var(--shadow-dialog);
}

.portrait img {
  display: block;
  width: 100%;
  height: auto;
  aspect-ratio: 4 / 5;
  border-radius: 1.5rem;
  object-fit: cover;
  object-position: 50% 30%;
}

/* The stage stands still; hover keeps only its colour change. */
@media (prefers-reduced-motion: reduce) {
  .hero-logo,
  .hero-tagline,
  .hero-actions,
  .hero-cue::before {
    animation: none;
  }

  .service {
    transition: none;
  }
}
</style>
