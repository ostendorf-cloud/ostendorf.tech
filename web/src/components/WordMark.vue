<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from '@ostendorf-cloud/govue'

import wordmarkDark from '../assets/logo-wordmark-dark.svg'
import wordmarkLight from '../assets/logo-wordmark-light.svg'

withDefaults(defineProps<{ alt?: string }>(), { alt: '' })

const { resolvedMode } = useTheme()
const src = computed(() => (resolvedMode.value === 'dark' ? wordmarkDark : wordmarkLight))
</script>

<template>
  <img :src="src" :alt="alt" width="160" height="36" />
</template>
