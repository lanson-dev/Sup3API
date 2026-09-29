<template>
  <HomeView v-if="hasCustomHome" />
  <!-- Trusted, bundled static markup. This source contains no user input or scripts. -->
  <div v-else v-html="landingMarkup"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import HomeView from './HomeView.vue'
import landingSource from '../content/agraphs-home.html?raw'

const appStore = useAppStore()
const hasCustomHome = computed(() => Boolean(appStore.cachedPublicSettings?.home_content?.trim()))
const styles = landingSource.match(/<style>([\s\S]*?)<\/style>/)?.[1] ?? ''
const body = landingSource.match(/<body>([\s\S]*?)<\/body>/)?.[1] ?? ''
const landingMarkup = `<style>${styles}</style>${body}`
</script>
