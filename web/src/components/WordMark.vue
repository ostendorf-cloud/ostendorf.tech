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
  <!-- 160 px is the design width. Below 480 px the header has to hold the
       mark, this and the actions in one row of 288 to 448 px, so the
       wordmark steps down; the height follows the aspect ratio. -->
  <img
    class="h-auto w-24 min-[360px]:w-32 min-[400px]:w-36 min-[480px]:w-40"
    :src="src"
    :alt="alt"
    width="160"
    height="36"
  />
</template>
