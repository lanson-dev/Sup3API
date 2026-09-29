<template>
  <!-- Trusted, bundled static markup. This source contains no user input or scripts. -->
  <div v-html="landingMarkup"></div>
</template>

<script setup lang="ts">
import landingSource from '../content/sup3api-home.html?raw'

const styles = landingSource.match(/<style>([\s\S]*?)<\/style>/)?.[1] ?? ''
const body = landingSource.match(/<body>([\s\S]*?)<\/body>/)?.[1] ?? ''
const landingMarkup = `<style>${styles}</style>${body}`
</script>
