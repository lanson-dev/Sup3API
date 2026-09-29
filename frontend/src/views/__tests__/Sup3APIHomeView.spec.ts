import {
  mount,
  flushPromises,
  RouterLinkStub,
  type VueWrapper,
} from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from '../Sup3APIHomeView.vue'
import { landingExamples } from '@/content/sup3api-landing'

const auth = vi.hoisted(() => ({ isAuthenticated: false }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
const wrappers: VueWrapper[] = []
const clipboard = vi.fn()
let reducedMotion = false
let mediaListener: ((event: { matches: boolean }) => void) | undefined
const scroll = vi.fn()

function mountHome() {
  const wrapper = mount(HomeView, {
    attachTo: document.body,
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LandingMesh: true,
        PlatformIcon: true,
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => {
  auth.isAuthenticated = false
  reducedMotion = false
  clipboard.mockReset().mockResolvedValue(undefined)
  scroll.mockReset()
  vi.stubGlobal('fetch', vi.fn())
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      matches: query.includes('reduced-motion') && reducedMotion,
      addEventListener: (_: string, callback: typeof mediaListener) => {
        mediaListener = callback
      },
      removeEventListener: vi.fn(),
    })),
  )
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: clipboard },
  })
  Element.prototype.scrollIntoView = scroll
})
afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
})

describe('Sup3API landing interactions', () => {
  it('routes the console correctly for signed-out and signed-in visitors', async () => {
    const wrapper = mountHome()
    expect(
      wrapper
        .findAllComponents(RouterLinkStub)
        .find((link) => link.classes().includes('sh-console'))
        ?.props('to'),
    ).toBe('/login')
    auth.isAuthenticated = true
    const signedIn = mountHome()
    expect(
      signedIn
        .findAllComponents(RouterLinkStub)
        .find((link) => link.classes().includes('sh-console'))
        ?.props('to'),
    ).toBe('/dashboard')
    expect(fetch).not.toHaveBeenCalled()
  })
  it('supports arrow navigation, wraparound and Home/End in capability tabs', async () => {
    const wrapper = mountHome()
    const first = wrapper.get('#capability-tab-text')
    await first.trigger('keydown', { key: 'ArrowLeft' })
    expect(
      wrapper.get('#capability-tab-mesh').attributes('aria-selected'),
    ).toBe('true')
    expect(document.activeElement?.id).toBe('capability-tab-mesh')
    expect(wrapper.get('#capability-panel-mesh').isVisible()).toBe(true)
    expect(wrapper.get('#capability-panel-text').isVisible()).toBe(false)
    await wrapper
      .get('#capability-tab-mesh')
      .trigger('keydown', { key: 'Home' })
    expect(first.attributes('tabindex')).toBe('0')
    await first.trigger('keydown', { key: 'End' })
    expect(document.activeElement?.id).toBe('capability-tab-mesh')
    await wrapper
      .get('#capability-tab-mesh')
      .trigger('keydown', { key: 'ArrowRight' })
    expect(first.attributes('aria-selected')).toBe('true')
  })
  it('opens mobile navigation, closes with Escape and restores focus', async () => {
    const wrapper = mountHome()
    const button = wrapper.get('.sh-menu-button')
    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('true')
    await wrapper.get('#home-mobile-menu').trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('#home-mobile-menu').exists()).toBe(false)
    expect(document.activeElement).toBe(button.element)
    await button.trigger('click')
    mediaListener?.({ matches: true })
    await flushPromises()
    expect(button.attributes('aria-expanded')).toBe('false')
  })
  it.each([false, true])(
    'moves section focus and respects reduced motion = %s',
    async (preference) => {
      reducedMotion = preference
      const wrapper = mountHome()
      await wrapper.get('.sh-menu-button').trigger('click')
      await wrapper
        .get('#home-mobile-menu a[href="#capabilities"]')
        .trigger('click')
      expect(document.activeElement?.id).toBe('capabilities')
      expect(scroll).toHaveBeenLastCalledWith({
        behavior: preference ? 'instant' : 'smooth',
        block: 'start',
      })
      expect(wrapper.find('#home-mobile-menu').exists()).toBe(false)
    },
  )
  it('copies the selected native example and provides feedback without sending requests', async () => {
    const wrapper = mountHome()
    await wrapper
      .get('#example-tab-unified')
      .trigger('keydown', { key: 'ArrowRight' })
    expect(document.activeElement?.id).toBe('example-tab-tripo')
    await wrapper.get('.sh-copy').trigger('click')
    await flushPromises()
    expect(clipboard).toHaveBeenCalledWith(landingExamples[1].code)
    expect(wrapper.get('[role="status"]').text()).toContain(
      'Tripo 原生代码已复制',
    )
    await wrapper.get('#example-tab-meshy').trigger('click')
    expect(wrapper.get('.sh-copy').text()).toBe('复制代码')
    expect(wrapper.get('#example-panel-meshy').text()).toContain(
      '/providers/meshy/openapi/v2/text-to-3d',
    )
    expect(fetch).not.toHaveBeenCalled()
  })
  it('shows a persistent, useful message when clipboard access fails', async () => {
    clipboard.mockRejectedValue(new Error('Permission denied'))
    const wrapper = mountHome()
    await wrapper.get('.sh-copy').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toContain('手动复制')
    expect(wrapper.get('.sh-copy').text()).toBe('复制代码')
  })
  it('does not show stale copy success after changing the example', async () => {
    let complete!: () => void
    clipboard.mockReturnValue(
      new Promise<void>((resolve) => {
        complete = resolve
      }),
    )
    const wrapper = mountHome()
    await wrapper.get('.sh-copy').trigger('click')
    await wrapper.get('#example-tab-meshy').trigger('click')
    complete()
    await flushPromises()
    expect(wrapper.get('.sh-copy').text()).toBe('复制代码')
    expect(wrapper.get('[role="status"]').text()).not.toContain('已复制')
  })
})
