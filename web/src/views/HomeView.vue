<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { GovueCard, GovuePage } from '@ostendorf-cloud/govue'

import ContactChannels from '../components/ContactChannels.vue'
import ContactForm from '../components/ContactForm.vue'
import CountingFacts from '../components/CountingFacts.vue'
import { bands } from '../data/bands'
import { experienceYearsFloor } from '../lib/experienceYears'
import type { Fact } from '../lib/facts'

// Destructuring tm is vue-i18n's own documented usage; unbound-method's
// warning about it does not apply, tm never reads `this`.
// eslint-disable-next-line @typescript-eslint/unbound-method
const { t, tm, rt } = useI18n()

// Annotated so the v-for below sees a real array type and a numeric
// index, which tm's own return type does not provably give it.
const tagline = computed<string[]>(() => tm('home.hero.tagline'))

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
    <section aria-labelledby="hero-title" class="flex flex-col items-start gap-4 py-8 sm:py-14">
      <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
        {{ t('home.hero.eyebrow') }}
      </p>
      <h1 id="hero-title" class="text-3xl font-semibold text-accent-text sm:text-4xl">
        {{ t('home.hero.title') }}
      </h1>
      <p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-lg text-text-muted">
        <template v-for="(item, index) in tagline" :key="item">
          <span v-if="index > 0" aria-hidden="true" class="text-border-strong">|</span>
          <span>{{ item }}</span>
        </template>
      </p>
      <div class="mt-2 flex flex-wrap gap-3">
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
    </section>

    <section id="services" aria-labelledby="services-title" class="scroll-mt-20 py-8 sm:py-14">
      <header class="mb-6 flex flex-col gap-2">
        <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
          {{ t('home.services.eyebrow') }}
        </p>
        <h2 id="services-title" class="text-2xl font-semibold text-text">
          {{ t('home.services.title') }}
        </h2>
        <p class="text-lg text-text-muted">{{ t('home.services.lead') }}</p>
      </header>

      <div class="grid gap-6 md:grid-cols-2">
        <GovueCard>
          <h3 class="mb-4 text-lg font-semibold text-text">
            {{ t('home.services.panelTechnology') }}
          </h3>
          <ul class="flex flex-col gap-3" role="list">
            <li
              v-for="item in tm('home.services.items')"
              :key="rt(item.title)"
              class="flex flex-col gap-0.5"
            >
              <span class="font-medium text-text">{{ rt(item.title) }}</span>
              <span class="text-sm text-text-muted">{{ rt(item.text) }}</span>
            </li>
          </ul>
        </GovueCard>

        <GovueCard>
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
          <p class="mt-4 text-sm">
            <a href="mailto:simon@ostendorf.tech" class="text-accent-text hover:underline">
              {{ t('home.services.more') }}
            </a>
          </p>
        </GovueCard>
      </div>
    </section>

    <section id="bands" aria-labelledby="bands-title" class="scroll-mt-20 py-8 sm:py-14">
      <header class="mb-6 flex flex-col gap-2">
        <p class="text-sm font-semibold tracking-wide text-text-muted uppercase">
          {{ t('home.bands.eyebrow') }}
        </p>
        <h2 id="bands-title" class="text-2xl font-semibold text-text">
          {{ t('home.bands.title') }}
        </h2>
        <p class="text-lg text-text-muted">{{ t('home.bands.lead') }}</p>
      </header>

      <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <GovueCard v-for="band in bands" :key="band.slug">
          <h3 class="mb-2 text-lg font-semibold text-text">{{ t(`bands.${band.slug}.name`) }}</h3>
          <div v-if="band.hasDescription" class="mb-3 flex flex-col gap-2 text-sm text-text-muted">
            <p v-for="(paragraph, index) in tm(`bands.${band.slug}.paragraphs`)" :key="index">
              {{ rt(paragraph) }}
            </p>
          </div>
          <a
            :href="band.instagramUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex min-h-11 items-center gap-1 text-sm text-accent-text hover:underline"
          >
            {{ t('home.bands.instagramLabel') }}: {{ band.instagramHandle }}
          </a>
        </GovueCard>
      </div>
    </section>

    <section id="about" aria-labelledby="about-title" class="scroll-mt-20 py-8 sm:py-14">
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
    </section>

    <section id="contact" aria-labelledby="contact-title" class="scroll-mt-20 py-8 sm:py-14">
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
