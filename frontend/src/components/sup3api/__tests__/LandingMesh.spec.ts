import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import LandingMesh from '../LandingMesh.vue'

const scene = vi.hoisted(() => ({
  setActive: vi.fn(),
  resize: vi.fn(),
  setPlaying: vi.fn(),
  setWireframe: vi.fn(),
  rotate: vi.fn(),
  reset: vi.fn(),
  dispose: vi.fn(),
}))
const createScene = vi.hoisted(() => vi.fn())
vi.mock('@/utils/landingMesh', () => ({ createLandingScene: createScene }))
let reduced = false
let intersect: (entries: { isIntersecting: boolean }[]) => void
let motion: (event: { matches: boolean }) => void
let interact: () => void
const disconnectResize = vi.fn(),
  disconnectIntersection = vi.fn()
let wrapper: VueWrapper | undefined
beforeEach(() => {
  vi.clearAllMocks()
  reduced = false
  createScene.mockImplementation(
    (_: HTMLCanvasElement, callback: () => void) => {
      interact = callback
      return scene
    },
  )
  vi.stubGlobal('matchMedia', () => ({
    matches: reduced,
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

it('honors reduced motion and permits explicit play and pause', async () => {
  reduced = true
  const view = await open()
  expect(scene.setPlaying).toHaveBeenLastCalledWith(false)
  await view.get('[aria-label="播放自动旋转"]').trigger('click')
  expect(scene.setPlaying).toHaveBeenLastCalledWith(true)
  motion({ matches: true })
  await flushPromises()
  expect(scene.setPlaying).toHaveBeenLastCalledWith(false)
})
it('exposes real canvas controls for keyboard, rotation, reset and wireframe', async () => {
  const view = await open()
  expect(view.find('canvas[tabindex="0"]').exists()).toBe(true)
  await view.get('canvas').trigger('keydown', { key: 'Home', ctrlKey: true })
  expect(scene.reset).not.toHaveBeenCalled()
  await view.get('canvas').trigger('keydown', { key: 'ArrowRight' })
  expect(scene.rotate).toHaveBeenLastCalledWith(0.16, 0)
  await view.get('[aria-label="向左旋转模型"]').trigger('click')
  expect(scene.rotate).toHaveBeenLastCalledWith(-0.24)
  await view.get('[aria-label="重置模型视角"]').trigger('click')
  expect(scene.reset).toHaveBeenCalledOnce()
  await view.findAll('.sh-mesh-segment button')[1].trigger('click')
  expect(scene.setWireframe).toHaveBeenLastCalledWith(true)
  interact()
  await flushPromises()
  expect(view.find('[aria-label="播放自动旋转"]').exists()).toBe(true)
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
  expect(view.find('.sh-mesh-fallback').exists()).toBe(true)
  expect(view.find('.sh-mesh-controls').exists()).toBe(false)
  view.unmount()
  wrapper = undefined
  const live = await open()
  await live.get('canvas').trigger('webglcontextlost')
  expect(scene.dispose).toHaveBeenCalledOnce()
  expect(live.find('.sh-mesh-fallback').exists()).toBe(true)
})
it('does not create a renderer if unmounted while the module loads', async () => {
  wrapper = mount(LandingMesh)
  expect(wrapper.find('.sh-mesh-controls').exists()).toBe(false)
  wrapper.unmount()
  wrapper = undefined
  await flushPromises()
  expect(createScene).not.toHaveBeenCalled()
})
