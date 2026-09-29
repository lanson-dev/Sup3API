<template>
  <div class="ag-site">
    <header class="ag-header">
      <RouterLink to="/home" class="ag-brand" aria-label="Sup3API 首页"><img src="/sup3api-mark.svg" alt="" />Sup3API</RouterLink>
      <nav class="ag-nav" aria-label="主导航">
        <RouterLink to="/connect">API 接入</RouterLink>
        <RouterLink to="/docs">API 文档</RouterLink>
        <RouterLink :to="auth.isAuthenticated ? '/dashboard' : '/login'" class="ag-nav-console">控制台 <span aria-hidden="true">↗</span></RouterLink>
      </nav>
    </header>
    <nav v-if="console" class="ag-console-nav" aria-label="客户控制台">
      <RouterLink to="/dashboard">概览</RouterLink><RouterLink to="/keys">API 密钥</RouterLink><RouterLink v-if="!auth.isSimpleMode" to="/usage">调用记录</RouterLink><RouterLink to="/profile">账户设置</RouterLink>
      <details v-if="moreLinks.length" ref="moreMenu" class="ag-more" @keydown.esc.prevent="closeMore(true)">
        <summary>更多</summary>
        <div class="ag-more-links"><RouterLink v-for="item in moreLinks" :key="item.path" :to="item.path">{{ item.label }}</RouterLink></div>
      </details>
      <RouterLink v-if="auth.isAdmin" to="/admin/dashboard">管理后台</RouterLink>
      <button class="ag-logout" @click="logout">退出</button>
    </nav>
    <main :class="['ag-main', { 'ag-main-wide': wide }]"><slot /></main>
    <footer class="ag-footer"><span>© {{ new Date().getFullYear() }} Sup3API</span><RouterLink to="/docs/sources">技术与开源 ↗</RouterLink></footer>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import '@/styles/sup3api.css'
const props = defineProps<{ console?: boolean; wide?: boolean }>()
const auth = useAuthStore(), app = useAppStore(), router = useRouter(), route = useRoute()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const moreMenu = ref<HTMLDetailsElement>()
const moreLinks = computed(() => {
  const settings = app.cachedPublicSettings
  const enabled = (flag: keyof typeof FeatureFlags) => resolveFeatureFlag(settings, FeatureFlags[flag])
  const purchaseLabel = { recharge_only: 'nav.recharge', subscription_only: 'nav.subscribe', recharge_and_subscription: 'nav.buySubscription' }[resolveSiteBillingMode(settings)]
  return [
    { path: '/batch-image', label: t('nav.batchImage'), visible: !auth.isSimpleMode && canUseBatchImage.value },
    { path: '/available-channels', label: t('nav.availableChannels'), visible: !auth.isSimpleMode && enabled('availableChannels') },
    { path: '/monitor', label: t('nav.channelStatus'), visible: enabled('channelMonitor') },
    { path: '/subscriptions', label: t('nav.mySubscriptions'), visible: !auth.isSimpleMode && enabled('subscription') },
    { path: '/purchase', label: t(purchaseLabel), visible: !auth.isSimpleMode && enabled('payment') },
    { path: '/orders', label: t('nav.myOrders'), visible: !auth.isSimpleMode && enabled('payment') },
    { path: '/redeem', label: t('nav.redeem'), visible: !auth.isSimpleMode },
    { path: '/affiliate', label: t('nav.affiliate'), visible: !auth.isSimpleMode && enabled('affiliate') },
    ...(settings?.custom_menu_items ?? []).filter(item => item.visibility === 'user').sort((a, b) => a.sort_order - b.sort_order).map(item => ({ path: `/custom/${item.id}`, label: item.label, visible: true }))
  ].filter(item => item.visible)
})
function closeMore(restoreFocus = false) {
  if (!moreMenu.value) return
  moreMenu.value.open = false
  if (restoreFocus) moreMenu.value.querySelector('summary')?.focus()
}
watch(() => route.path, () => closeMore())
onMounted(() => { if (props.console && auth.isAuthenticated) void refreshBatchImageAccess() })
async function logout() { await auth.logout(); await router.push('/home') }
</script>

<style scoped>
.ag-more { position: relative; }
.ag-more summary { cursor: pointer; min-height: 44px; display: flex; align-items: center; gap: 8px; }
.ag-more summary::after { content: '⌄'; }
.ag-more summary::-webkit-details-marker { display: none; }
.ag-more-links { position: absolute; top: 100%; left: 0; z-index: 30; min-width: 180px; max-height: min(65vh, 460px); overflow-y: auto; padding: 8px; border: 1px solid var(--ag-line); border-radius: 12px; background: white; box-shadow: 0 12px 32px #18243b18; }
.ag-more-links a { display: flex; align-items: center; min-height: 44px; padding: 8px 12px; border-radius: 6px; }
.ag-more-links a:hover { background: var(--ag-paper); }
</style>
