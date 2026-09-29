import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { PublicSettings } from '@/types'
import Sup3APIShell from '../Sup3APIShell.vue'

const auth = reactive({ isAuthenticated: true, isAdmin: false, isSimpleMode: false, logout: vi.fn() })
const app = reactive({ cachedPublicSettings: {} as Partial<PublicSettings> })
const canUseBatchImage = ref(false)
const refreshBatchImageAccess = vi.fn()
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/composables/useBatchImageAccess', () => ({ useBatchImageAccess: () => ({ canUseBatchImage, refreshBatchImageAccess }) }))

beforeEach(() => {
  vi.clearAllMocks()
  auth.isAdmin = auth.isSimpleMode = canUseBatchImage.value = false
  app.cachedPublicSettings = { payment_enabled: false, subscription_enabled: false, channel_monitor_enabled: false, available_channels_enabled: false, affiliate_enabled: false }
})

async function setup() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  await router.push('/dashboard')
  await router.isReady()
  const wrapper = mount(Sup3APIShell, { props: { console: true }, global: { plugins: [router] } })
  return { wrapper, router }
}

describe('Sup3API customer navigation', () => {
  it('restores enabled customer features and custom menus without exposing disabled or admin entries', async () => {
    const { wrapper } = await setup()
    const links = () => wrapper.findAll('nav[aria-label="客户控制台"] a').map(link => link.attributes('href'))
    expect(links()).toEqual(['/dashboard', '/keys', '/usage', '/profile', '/redeem'])
    app.cachedPublicSettings = {
      payment_enabled: true, subscription_enabled: true, channel_monitor_enabled: true, available_channels_enabled: true, affiliate_enabled: true,
      custom_menu_items: [
        { id: 'admin', label: 'Private', visibility: 'admin', sort_order: 0, url: '/private', icon_svg: '' },
        { id: 'user', label: 'Customer docs', visibility: 'user', sort_order: 1, url: '/docs', icon_svg: '' }
      ]
    }
    canUseBatchImage.value = true
    await flushPromises()
    expect(links()).toEqual(['/dashboard', '/keys', '/usage', '/profile', '/batch-image', '/available-channels', '/monitor', '/subscriptions', '/purchase', '/orders', '/redeem', '/affiliate', '/custom/user'])
    expect(refreshBatchImageAccess).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('preserves simple mode and closes the native disclosure after navigation or Escape', async () => {
    auth.isSimpleMode = true
    app.cachedPublicSettings.channel_monitor_enabled = true
    const { wrapper, router } = await setup()
    expect(wrapper.find('a[href="/usage"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/redeem"]').exists()).toBe(false)
    const details = wrapper.find('details')
    const element = details.element as HTMLDetailsElement
    element.open = true
    await router.push('/monitor')
    expect(element.open).toBe(false)
    element.open = true
    await details.trigger('keydown', { key: 'Escape' })
    expect(element.open).toBe(false)
    wrapper.unmount()
  })
})
