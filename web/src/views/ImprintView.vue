<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovueLegalPage } from '@ostendorf-cloud/govue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const germanHref = computed(
  () => router.resolve({ name: 'imprint', params: { locale: 'de' } }).href,
)
</script>

<template>
  <!-- The site's own skip link and language switch are in App.vue and the header. -->
  <GovueLegalPage
    page="imprint"
    :title="t('legal.imprint.title')"
    :skip-link="false"
    :language-switch="false"
  >
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

      <h2 class="text-xl font-semibold">Redaktionell verantwortlich nach § 18 Abs. 2 MStV</h2>
      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen
      </p>

      <h2 class="text-xl font-semibold">Streitbeilegung</h2>
      <p>
        Wir sind zur Teilnahme an einem Streitbeilegungsverfahren vor einer
        Verbraucherschlichtungsstelle weder verpflichtet noch bereit.
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

      <h2 class="text-xl font-semibold">
        Editorially responsible under Section 18(2) of the German Interstate Media Treaty (MStV)
      </h2>
      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen, Germany
      </p>

      <h2 class="text-xl font-semibold">Dispute resolution</h2>
      <p>
        We are neither obliged nor willing to take part in dispute resolution proceedings before a
        consumer arbitration board.
      </p>
    </div>
  </GovueLegalPage>
</template>
