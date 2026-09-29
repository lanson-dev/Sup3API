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
      <RouterLink to="/dashboard">概览</RouterLink><RouterLink to="/keys">API 密钥</RouterLink><RouterLink to="/usage">调用记录</RouterLink><RouterLink to="/profile">账户设置</RouterLink>
      <RouterLink v-if="app.cachedPublicSettings?.payment_enabled" to="/purchase">余额与订阅</RouterLink>
      <RouterLink v-if="app.cachedPublicSettings?.subscription_enabled !== false" to="/subscriptions">我的订阅</RouterLink>
      <RouterLink v-if="auth.isAdmin" to="/admin/dashboard">管理后台</RouterLink>
      <button class="ag-logout" @click="logout">退出</button>
    </nav>
    <main :class="['ag-main', { 'ag-main-wide': wide }]"><slot /></main>
    <footer class="ag-footer"><span>© {{ new Date().getFullYear() }} Sup3API</span><span>TEXT / IMAGE / MESH</span><RouterLink to="/docs/sources">技术与开源说明 ↗</RouterLink></footer>
  </div>
</template>
<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import '@/styles/sup3api.css'
defineProps<{ console?: boolean; wide?: boolean }>()
const auth = useAuthStore(), app = useAppStore(), router = useRouter()
async function logout() { await auth.logout(); await router.push('/home') }
</script>
