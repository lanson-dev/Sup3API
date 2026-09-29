<template>
  <div class="space-y-3">
    <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>
    <ModelWhitelistSelector v-model="allowedModels" :platform="platform" :account-id="accountId" :sync-credentials="syncCredentials" @upstream-synced="emit('upstream-synced')" />
    <p class="input-hint">不配置限制或映射则允许全部模型。</p>
    <label class="input-label">{{ t('admin.accounts.modelMapping') }}</label>
    <div v-for="entry in aliases" :key="entry.from" class="flex items-center gap-2 text-sm">
      <code class="min-w-0 flex-1 break-all">{{ entry.from }} → {{ entry.to }}</code>
      <button type="button" class="btn btn-secondary" @click="removeAlias(entry.from)">{{ t('common.delete') }}</button>
    </div>
    <div class="flex flex-wrap gap-2">
      <input v-model.trim="alias" class="input min-w-0 flex-1" :placeholder="t('admin.accounts.requestModel')" :aria-label="t('admin.accounts.requestModel')" />
      <input v-model.trim="target" class="input min-w-0 flex-1" :placeholder="t('admin.accounts.actualModel')" :aria-label="t('admin.accounts.actualModel')" />
      <button type="button" class="btn btn-secondary" :disabled="!alias || !target || alias.includes('*') || target.includes('*')" @click="addAlias">{{ t('common.add') }}</button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelWhitelistSelector from './ModelWhitelistSelector.vue'
import { buildModelMappingObject, splitModelMappingObject } from '@/composables/useModelWhitelist'
import type { SyncUpstreamPreviewParams } from '@/api/admin/accounts'
const props = defineProps<{ modelValue: Record<string, string>; platform: string; accountId?: number; syncCredentials?: SyncUpstreamPreviewParams }>()
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, string>]; 'upstream-synced': [] }>()
const { t } = useI18n()
const alias = ref('')
const target = ref('')
const aliases = computed(() => splitModelMappingObject(props.modelValue).modelMappings)
const allowedModels = computed({
  get: () => splitModelMappingObject(props.modelValue).allowedModels,
  set: models => emit('update:modelValue', buildModelMappingObject('combined', models, aliases.value) ?? {})
})
function addAlias() {
  if (!alias.value || !target.value || alias.value.includes('*') || target.value.includes('*')) return
  emit('update:modelValue', { ...props.modelValue, [alias.value]: target.value })
  alias.value = target.value = ''
}
function removeAlias(from: string) {
  const mapping = { ...props.modelValue }
  delete mapping[from]
  emit('update:modelValue', mapping)
}
watch(() => props.platform, () => { alias.value = target.value = '' })
</script>
