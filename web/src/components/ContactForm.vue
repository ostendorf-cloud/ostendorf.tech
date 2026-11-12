<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  createClient,
  useForm,
  ApiError,
  ErrorCode,
  GovueButton,
  GovueCheckbox,
  GovueField,
  GovueInput,
  GovueTextarea,
} from '@ostendorf-cloud/govue'

import { createChallengeSolver } from '../lib/contactChallenge'

interface ContactFormValues {
  name: string
  email: string
  message: string
  consent: boolean
  /** The honeypot: invisible to people, never styled with display:none, so
   *  a bot that only checks for that one property still fills it. */
  website: string
}

interface ContactResponse {
  message: string
}

const route = useRoute()
const { t, locale } = useI18n()

// The CSRF token is minted together with the proof of work challenge (see
// lib/contactChallenge.ts); the client reads it fresh on every unsafe
// request, so a token refreshed mid-visit needs no extra wiring here.
const csrfToken = ref('')
const client = createClient({ csrfToken: () => csrfToken.value })
const solver = createChallengeSolver(client, () => locale.value)

const successMessage = ref('')
const sent = ref(false)

const form = useForm<ContactFormValues>({
  initial: { name: '', email: '', message: '', consent: false, website: '' },
  submit: async (values) => {
    if (values.website.trim() !== '') {
      // A filled honeypot is a bot. It gets the same success behaviour as
      // a person and learns nothing; there is nothing worth solving a
      // challenge for, so this never reaches the network call below.
      successMessage.value = t('home.contact.form.success')
      return
    }

    try {
      const solution = await solver.solved()
      csrfToken.value = solution.csrfToken
      const response = await client.post<ContactResponse>('/contact', {
        name: values.name,
        email: values.email,
        message: values.message,
        consent: values.consent,
        website: values.website,
        locale: locale.value,
        pow: solution.pow,
      })
      successMessage.value = response.message
    } catch (caught) {
      if (caught instanceof ApiError) {
        throw caught
      }
      // Fetching or solving the challenge failed before any request the
      // server could answer even went out; useForm shows an ApiError's own
      // message, so wrapping this in one is what makes the translated
      // fallback appear instead of a hardcoded English line.
      throw new ApiError({ code: ErrorCode.network, message: t('home.contact.form.fallbackError') })
    }
  },
})

async function onSubmit(): Promise<void> {
  sent.value = await form.handleSubmit()
}

function onFocusIn(): void {
  solver.prepare()
}
</script>

<template>
  <div class="rounded-lg border border-border bg-surface p-4 sm:p-6">
    <p v-if="sent" role="status" class="text-base text-text">
      {{ successMessage }}
    </p>

    <form
      v-else
      class="flex flex-col gap-4"
      novalidate
      @focusin="onFocusIn"
      @submit.prevent="onSubmit"
    >
      <h3 class="text-lg font-semibold text-text">{{ t('home.contact.form.title') }}</h3>

      <GovueField :label="t('home.contact.form.name')" required :error="form.errorOf('name')">
        <GovueInput
          v-model="form.values.name"
          name="name"
          autocomplete="name"
          :maxlength="200"
          @blur="form.blur('name')"
        />
      </GovueField>

      <GovueField :label="t('home.contact.form.email')" required :error="form.errorOf('email')">
        <GovueInput
          v-model="form.values.email"
          type="email"
          name="email"
          autocomplete="email"
          inputmode="email"
          :maxlength="254"
          @blur="form.blur('email')"
        />
      </GovueField>

      <GovueField :label="t('home.contact.form.message')" required :error="form.errorOf('message')">
        <GovueTextarea
          v-model="form.values.message"
          name="message"
          :rows="6"
          :maxlength="5000"
          @blur="form.blur('message')"
        />
      </GovueField>

      <div class="absolute -left-[9999px]" aria-hidden="true">
        <label for="contact-website">{{ t('home.contact.form.honeypot') }}</label>
        <input
          id="contact-website"
          v-model="form.values.website"
          type="text"
          name="website"
          tabindex="-1"
          autocomplete="off"
        />
      </div>

      <div>
        <GovueCheckbox
          v-model="form.values.consent"
          :label="
            t('home.contact.form.consent.prefix') +
            ' ' +
            t('home.contact.form.consent.linkLabel') +
            ' ' +
            t('home.contact.form.consent.suffix')
          "
        />
        <p class="-mt-1 pl-10 text-sm">
          <router-link
            :to="{ name: 'privacy', params: { locale: route.params.locale } }"
            target="_blank"
            rel="noopener"
            class="text-accent-text hover:underline"
          >
            {{ t('home.contact.form.consent.linkLabel') }}
          </router-link>
        </p>
      </div>

      <p
        v-if="form.problem.value"
        role="alert"
        class="rounded-md bg-danger-subtle p-3 text-danger-text"
      >
        {{ form.problem.value }}
      </p>

      <div class="flex flex-col gap-2">
        <GovueButton
          type="submit"
          variant="primary"
          :disabled="!form.values.consent"
          :loading="form.submitting.value"
          :loading-label="t('home.contact.form.submitting')"
        >
          {{ t('home.contact.form.submit') }}
        </GovueButton>
        <p class="text-sm text-text-muted">
          <span aria-hidden="true">*</span> {{ t('home.contact.form.hint') }}
        </p>
        <noscript>
          <p class="text-sm text-text-muted">{{ t('home.contact.form.noscript') }}</p>
        </noscript>
      </div>
    </form>
  </div>
</template>
