<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovueFooter, type FooterLink } from '@ostendorf-cloud/govue'

import ChannelIcon from './ChannelIcon.vue'
import { channels } from '../data/channels'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const links = computed<FooterLink[]>(() => {
  const locale = route.params.locale
  const resolve = (name: string) => router.resolve({ name, params: { locale } }).href
  return [
    { href: resolve('privacy'), label: t('legal.privacy.title') },
    { href: resolve('imprint'), label: t('legal.imprint.title') },
    { href: resolve('agb'), label: t('legal.agb.title') },
  ]
})

const note = computed(() => t('footer.copyright', { year: new Date().getFullYear() }))
</script>

<template>
  <GovueFooter :links="links" :note="note" :aria-label="t('legal.eyebrow')">
    <ul class="flex items-center gap-1" role="list" :aria-label="t('footer.social.eyebrow')">
      <li v-for="channel in channels" :key="channel.key">
        <a
          :href="channel.href"
          :target="channel.external ? '_blank' : undefined"
          :rel="channel.external ? 'noopener noreferrer' : undefined"
          :aria-label="t(`footer.social.${channel.key}`)"
          class="flex min-h-11 min-w-11 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
        >
          <ChannelIcon :name="channel.key" />
        </a>
      </li>
    </ul>
  </GovueFooter>
</template>
