import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import LandingMesh from '../LandingMesh.vue'

const scene = vi.hoisted(() => ({
  setActive: vi.fn(),
  resize: vi.fn(),
  setPlaying: vi.fn(),
  dispose: vi.fn(),
}))
const createScene = vi.hoisted(() => vi.fn())
vi.mock('@/utils/landingMesh', () => ({ createLandingScene: createScene }))
let reduced = false
let intersect: (entries: { isIntersecting: boolean }[]) => void
let motion: (event: { matches: boolean }) => void
const disconnectResize = vi.fn(),
  disconnectIntersection = vi.fn()
let wrapper: VueWrapper | undefined
beforeEach(() => {
  vi.clearAllMocks()
  reduced = false
  createScene.mockReturnValue(scene)
  vi.stubGlobal('matchMedia', () => ({
    get matches() {
      return reduced
    },
    addEventListener: (_: string, cb: typeof motion) => {
      motion = cb
    },
    removeEventListener: vi.fn(),
  }))
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      disconnect = disconnectResize
    },
  )
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(cb: typeof intersect) {
        intersect = cb
      }
      observe = vi.fn()
      disconnect = disconnectIntersection
    },
  )
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})
async function open() {
  wrapper = mount(LandingMesh)
  await flushPromises()
  return wrapper
}

it('auto-rotates without control buttons and respects reduced motion', async () => {
  const view = await open()
  expect(scene.setPlaying).toHaveBeenLastCalledWith(true)
  expect(view.findAll('button')).toHaveLength(0)
  reduced = true
  motion({ matches: true })
  expect(scene.setPlaying).toHaveBeenLastCalledWith(false)
  reduced = false
  motion({ matches: false })
  expect(scene.setPlaying).toHaveBeenLastCalledWith(true)
})
it('stops hidden scenes, resumes visible scenes and releases GPU resources', async () => {
  await open()
  intersect([{ isIntersecting: true }])
  expect(scene.setActive).toHaveBeenLastCalledWith(true)
  intersect([{ isIntersecting: false }])
  expect(scene.setActive).toHaveBeenLastCalledWith(false)
  intersect([{ isIntersecting: true }])
  vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
  document.dispatchEvent(new Event('visibilitychange'))
  expect(scene.setActive).toHaveBeenLastCalledWith(false)
  wrapper!.unmount()
  wrapper = undefined
  expect(scene.dispose).toHaveBeenCalledOnce()
  expect(disconnectResize).toHaveBeenCalledOnce()
  expect(disconnectIntersection).toHaveBeenCalledOnce()
})
it('falls back when WebGL cannot initialize or its context is lost', async () => {
  createScene.mockImplementationOnce(() => {
    throw new Error('No WebGL')
  })
  const view = await open()
  expect(view.find('img[alt="Sup3API"]').exists()).toBe(true)
  expect(view.find('.sh-mesh-controls').exists()).toBe(false)
  view.unmount()
  wrapper = undefined
  const live = await open()
  await live.get('canvas').trigger('webglcontextlost')
  expect(scene.dispose).toHaveBeenCalledOnce()
  expect(live.find('img[alt="Sup3API"]').exists()).toBe(true)
})
it('does not create a renderer if unmounted while the module loads', async () => {
  wrapper = mount(LandingMesh)
  expect(wrapper.find('.sh-mesh-controls').exists()).toBe(false)
  wrapper.unmount()
  wrapper = undefined
  await flushPromises()
  expect(createScene).not.toHaveBeenCalled()
})
