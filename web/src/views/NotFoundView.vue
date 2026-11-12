<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { GovuePage } from '@ostendorf-cloud/govue'

const route = useRoute()
const { t } = useI18n()
</script>

<template>
  <GovuePage width="reading" :title="t('notFound.title')" :lead="t('notFound.lead')">
    <p>
      <router-link
        class="text-accent-text hover:underline"
        :to="{ name: 'home', params: { locale: route.params.locale } }"
      >
        {{ t('notFound.backHome') }}
      </router-link>
    </p>
  </GovuePage>
</template>
