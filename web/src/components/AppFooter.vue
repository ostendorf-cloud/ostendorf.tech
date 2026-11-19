<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { GovueFooter } from '@ostendorf-cloud/govue'

import ChannelIcon from './ChannelIcon.vue'
import { channels } from '../data/channels'
import { SITE_LEGAL_PAGES } from '../lib/legal'

const { t } = useI18n()

const note = computed(() => t('footer.copyright', { year: new Date().getFullYear() }))
</script>

<template>
  <!-- govue links the legal pages first, in the active language, at /<code>/<page>. -->
  <GovueFooter :legal="SITE_LEGAL_PAGES" :note="note" :aria-label="t('legal.eyebrow')">
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
