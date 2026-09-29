<template>
  <div class="ag-code"><div class="ag-code-bar"><span>{{ label || 'JSON' }}</span><button @click="copy">{{ copied ? '已复制' : '复制' }}</button></div><pre><code>{{ code }}</code></pre><span v-if="error" role="status" class="ag-copy-error">{{ error }}</span></div>
</template>
<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue'
const props = defineProps<{ code: string; label?: string }>()
const copied = ref(false), error = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
async function copy() { try { await navigator.clipboard.writeText(props.code); copied.value = true; error.value = ''; clearTimeout(timer); timer = setTimeout(() => copied.value = false, 1800) } catch { error.value = '复制不可用，请选择代码手动复制。' } }
onBeforeUnmount(() => clearTimeout(timer))
</script>
