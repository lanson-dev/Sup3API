<template>
  <BaseDialog :show="show" :title="account ? '编辑供应商账号' : '添加供应商账号'" width="normal" @close="emit('close')">
    <form id="asset-account-form" class="space-y-4" @submit.prevent="save">
      <p class="text-sm text-gray-500">平台使用此账号调用上游。应用请使用「API 密钥」中创建的 Sup3API 密钥。</p>
      <label class="block"><span class="input-label">供应商</span>
        <select v-model="platform" class="input" :disabled="!!account"><option value="tripo">Tripo</option><option value="meshy">Meshy</option></select>
      </label>
      <label class="block"><span class="input-label">账号名称</span><input v-model.trim="name" class="input" required maxlength="100" /></label>
      <label class="block"><span class="input-label">上游 API Key</span>
        <input v-model.trim="apiKey" class="input" type="password" autocomplete="new-password" :required="!account" :placeholder="account ? '留空保留当前凭据' : '填写供应商开放平台的 API Key'" />
      </label>
      <p class="text-xs text-gray-500">使用官方 API 地址和 API 额度。Studio 订阅不等同于 API 额度。</p>
      <label class="block"><span class="input-label">优先级（数值越小越优先）</span><input v-model.number="priority" class="input" type="number" min="0" required /></label>
      <label v-if="account" class="block"><span class="input-label">状态</span>
        <select v-model="status" class="input"><option value="active">启用</option><option value="inactive">停用</option><option value="error">异常</option></select>
      </label>
      <p class="text-xs text-gray-500">现有任务绑定原账号；停用或替换凭据后，不会转交其他账号。</p>
      <p v-if="error" role="alert" class="text-sm text-red-500">{{ error }}</p>
    </form>
    <template #footer><div class="flex justify-end gap-3">
      <button class="btn btn-secondary" :disabled="saving" @click="emit('close')">取消</button>
      <button class="btn btn-primary" form="asset-account-form" type="submit" :disabled="saving">{{ saving ? '保存中…' : '保存账号' }}</button>
    </div></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

const props = defineProps<{ show: boolean; provider: 'tripo' | 'meshy'; account: Account | null }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const platform = ref<'tripo' | 'meshy'>('tripo')
const name = ref('')
const apiKey = ref('')
const priority = ref(1)
const status = ref<'active' | 'inactive' | 'error'>('active')
const saving = ref(false)
const error = ref('')
watch(() => props.show, (show) => {
  apiKey.value = ''
  if (!show) return
  platform.value = props.provider
  name.value = props.account?.name ?? ''
  priority.value = props.account?.priority ?? 1
  status.value = props.account?.status ?? 'active'
  error.value = ''
})
async function save() {
  if (saving.value) return
  saving.value = true
  error.value = ''
  try {
    if (props.account) {
      await adminAPI.accounts.update(props.account.id, {
        name: name.value, priority: priority.value, status: status.value,
        ...(apiKey.value ? { credentials: { api_key: apiKey.value } } : {})
      })
    } else {
      await adminAPI.accounts.create({ name: name.value, platform: platform.value, type: 'apikey',
        credentials: { api_key: apiKey.value }, concurrency: 1, priority: priority.value, group_ids: [] })
    }
    apiKey.value = ''
    emit('saved')
    emit('close')
  } catch {
    error.value = '保存失败，请检查账号配置后重试。'
  } finally {
    saving.value = false
  }
}
</script>
