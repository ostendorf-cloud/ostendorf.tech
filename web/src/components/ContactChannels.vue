<script setup lang="ts">
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
        class="flex min-h-11 items-center gap-3 rounded-lg border border-border bg-surface p-4 text-text shadow-card transition-colors hover:bg-surface-hover"
      >
        <ChannelIcon :name="channel.key" class="text-accent-text" />
        <span class="flex flex-col">
          <span class="text-sm font-medium text-text-muted">
            {{ t(`home.contact.channels.${channel.key}.label`) }}
          </span>
          <span class="text-base text-text">
            {{ t(`home.contact.channels.${channel.key}.value`) }}
          </span>
        </span>
      </a>
    </li>
  </ul>
</template>
