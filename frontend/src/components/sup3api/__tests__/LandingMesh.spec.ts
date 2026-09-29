import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LandingMesh from '../LandingMesh.vue'

const paint = vi.hoisted(() => vi.fn())
vi.mock('@/utils/landingMesh', () => ({
  createKnotMesh: () => ({}),
  paintKnot: paint,
}))
let reduced = false
let onMotion: (event: { matches: boolean }) => void
let onIntersection: (entries: { isIntersecting: boolean }[]) => void
const disconnectResize = vi.fn(),
  disconnectIntersection = vi.fn(),
  removeMotion = vi.fn()
let wrapper: VueWrapper | undefined
let nextFrame = 0
const frames = new Map<number, FrameRequestCallback>()

beforeEach(() => {
  reduced = false
  nextFrame = 0
  frames.clear()
  vi.clearAllMocks()
  vi.stubGlobal(
    'requestAnimationFrame',
    vi.fn((callback: FrameRequestCallback) => {
      frames.set(++nextFrame, callback)
      return nextFrame
    }),
  )
  vi.stubGlobal(
    'cancelAnimationFrame',
    vi.fn((id: number) => frames.delete(id)),
  )
  vi.stubGlobal('matchMedia', () => ({
    matches: reduced,
    addEventListener: (_: string, callback: typeof onMotion) => {
      onMotion = callback
    },
    removeEventListener: removeMotion,
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
      constructor(callback: typeof onIntersection) {
        onIntersection = callback
      }
      observe = vi.fn()
      disconnect = disconnectIntersection
    },
  )
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    setTransform: vi.fn(),
  } as unknown as CanvasRenderingContext2D)
  vi.spyOn(
    HTMLCanvasElement.prototype,
    'getBoundingClientRect',
  ).mockReturnValue({ width: 500, height: 450 } as DOMRect)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('landing mesh motion and accessibility', () => {
  it('renders a static frame and does not autoplay with reduced motion', () => {
    reduced = true
    wrapper = mount(LandingMesh)
    expect(paint).toHaveBeenCalledOnce()
    expect(frames.size).toBe(0)
    expect(wrapper.find('[aria-label="播放自动旋转"]').exists()).toBe(true)
  })
  it('allows explicit play, pause, wireframe and reset', async () => {
    reduced = true
    wrapper = mount(LandingMesh)
    await wrapper.get('[aria-label="播放自动旋转"]').trigger('click')
    expect(frames.size).toBe(1)
    await wrapper.get('[aria-label="暂停自动旋转"]').trigger('click')
    expect(frames.size).toBe(0)
    await wrapper.findAll('.sh-mesh-segment button')[1].trigger('click')
    expect(paint.mock.calls.at(-1)?.at(-1)).toBe(true)
    await wrapper.get('[aria-label="重置模型视角"]').trigger('click')
    expect(frames.size).toBe(0)
    expect(paint.mock.calls.at(-1)?.[4]).toEqual({ x: -0.48, y: 0.32 })
  })
  it('pauses automatic movement for keyboard manipulation and honors reset', async () => {
    wrapper = mount(LandingMesh)
    expect(frames.size).toBe(1)
    await wrapper
      .get('.sh-mesh-surface')
      .trigger('keydown', { key: 'ArrowRight' })
    expect(frames.size).toBe(0)
    expect(paint.mock.calls.at(-1)?.[4].y).toBeCloseTo(0.48)
    await wrapper.get('.sh-mesh-surface').trigger('keydown', { key: 'Home' })
    expect(paint.mock.calls.at(-1)?.[4].y).toBe(0.32)
  })
  it('offers single-click alternatives to dragging in both directions', async () => {
    wrapper = mount(LandingMesh)
    await wrapper.get('[aria-label="向左旋转模型"]').trigger('click')
    expect(paint.mock.calls.at(-1)?.[4].y).toBeCloseTo(0.08)
    expect(frames.size).toBe(0)
    await wrapper.get('[aria-label="向右旋转模型"]').trigger('click')
    expect(paint.mock.calls.at(-1)?.[4].y).toBeCloseTo(0.32)
  })
  it('suspends animation outside the viewport, and never overrides a manual pause', async () => {
    wrapper = mount(LandingMesh)
    onIntersection([{ isIntersecting: false }])
    expect(frames.size).toBe(0)
    onIntersection([{ isIntersecting: true }])
    await wrapper.vm.$nextTick()
    expect(frames.size).toBe(1)
    await wrapper.get('[aria-label="暂停自动旋转"]').trigger('click')
    onIntersection([{ isIntersecting: false }])
    onIntersection([{ isIntersecting: true }])
    expect(frames.size).toBe(0)
  })
  it('stops on a live reduced-motion preference change and releases observers on unmount', () => {
    wrapper = mount(LandingMesh)
    onMotion({ matches: true })
    expect(frames.size).toBe(0)
    wrapper.unmount()
    wrapper = undefined
    expect(disconnectResize).toHaveBeenCalledOnce()
    expect(disconnectIntersection).toHaveBeenCalledOnce()
    expect(removeMotion).toHaveBeenCalledOnce()
  })
  it('leaves vertical touch gestures to page scrolling and rotates on horizontal drag', async () => {
    wrapper = mount(LandingMesh)
    const surface = wrapper.get('.sh-mesh-surface')
    const capture = vi.fn(),
      release = vi.fn()
    Object.assign(surface.element, {
      setPointerCapture: capture,
      hasPointerCapture: () => true,
      releasePointerCapture: release,
    })
    const pointer = {
      pointerId: 1,
      isPrimary: true,
      button: 0,
      pointerType: 'touch',
    }
    await surface.trigger('pointerdown', {
      ...pointer,
      clientX: 100,
      clientY: 100,
    })
    await surface.trigger('pointermove', {
      ...pointer,
      clientX: 101,
      clientY: 140,
    })
    expect(capture).not.toHaveBeenCalled()
    expect(frames.size).toBe(1)
    await surface.trigger('pointerdown', {
      ...pointer,
      clientX: 100,
      clientY: 100,
    })
    await surface.trigger('pointermove', {
      ...pointer,
      clientX: 150,
      clientY: 102,
    })
    expect(capture).toHaveBeenCalledWith(1)
    expect(frames.size).toBe(0)
    expect(paint.mock.calls.at(-1)?.[4].y).toBeCloseTo(0.72)
    await surface.trigger('pointercancel', pointer)
    expect(release).toHaveBeenCalledWith(1)
  })
  it('suspends in a hidden tab and resumes only while playing', async () => {
    wrapper = mount(LandingMesh)
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
    document.dispatchEvent(new Event('visibilitychange'))
    expect(frames.size).toBe(0)
    hidden.mockReturnValue(false)
    document.dispatchEvent(new Event('visibilitychange'))
    expect(frames.size).toBe(1)
  })
  it('offers a static branded fallback when canvas is unavailable', () => {
    vi.mocked(HTMLCanvasElement.prototype.getContext).mockReturnValue(null)
    wrapper = mount(LandingMesh)
    return wrapper.vm.$nextTick().then(() => {
      expect(wrapper.find('.sh-mesh-fallback').exists()).toBe(true)
      expect(wrapper.find('.sh-mesh-controls').exists()).toBe(false)
      expect(frames.size).toBe(0)
    })
  })
})
