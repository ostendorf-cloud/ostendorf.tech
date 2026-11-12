<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovuePage } from '@ostendorf-cloud/govue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const germanHref = computed(
  () => router.resolve({ name: 'imprint', params: { locale: 'de' } }).href,
)
</script>

<template>
  <GovuePage width="reading" :title="t('legal.imprint.title')">
    <div v-if="route.params.locale === 'de'" class="flex flex-col gap-4 text-text">
      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen
      </p>

      <h2 class="text-xl font-semibold">Kontakt</h2>
      <p>
        Telefon:
        <a class="text-accent-text hover:underline" href="tel:+4915730971784">+49 1573 0971784</a
        ><br />
        E-Mail:
        <a class="text-accent-text hover:underline" href="mailto:simon@ostendorf.tech"
          >simon@ostendorf.tech</a
        >
      </p>

      <h2 class="text-xl font-semibold">Umsatzsteuer-ID</h2>
      <p>Umsatzsteuer-Identifikationsnummer gemäß §27a Umsatzsteuergesetz: DE362164357</p>

      <h2 class="text-xl font-semibold">Verbraucherstreitbeilegung/Universalschlichtungsstelle</h2>
      <p>
        Wir nehmen an einem Streitbeilegungsverfahren vor einer Verbraucherschlichtungsstelle teil.
        Zuständig ist die Universalschlichtungsstelle des Zentrums für Schlichtung e.V., Straßburger
        Straße 8, 77694 Kehl am Rhein (<a
          class="text-accent-text hover:underline"
          href="https://www.verbraucher-schlichter.de"
          target="_blank"
          rel="noopener noreferrer"
          >https://www.verbraucher-schlichter.de</a
        >).
      </p>
    </div>

    <div v-else class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        {{ t('legal.bindingNotice') }}
        <a class="text-accent-text hover:underline" :href="germanHref">{{
          t('legal.viewGerman')
        }}</a>
      </p>

      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen, Germany
      </p>

      <h2 class="text-xl font-semibold">Contact</h2>
      <p>
        Phone:
        <a class="text-accent-text hover:underline" href="tel:+4915730971784">+49 1573 0971784</a
        ><br />
        Email:
        <a class="text-accent-text hover:underline" href="mailto:simon@ostendorf.tech"
          >simon@ostendorf.tech</a
        >
      </p>

      <h2 class="text-xl font-semibold">VAT ID</h2>
      <p>VAT identification number per Section 27a of the German VAT Act: DE362164357</p>
    </div>
  </GovuePage>
</template>
