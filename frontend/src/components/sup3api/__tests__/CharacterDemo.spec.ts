import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import CharacterDemo from '../CharacterDemo.vue'

const viewer = vi.hoisted(() => ({
  load: vi.fn(),
  setMode: vi.fn(),
  setActive: vi.fn(),
  rotate: vi.fn(),
  resize: vi.fn(),
  dispose: vi.fn(),
}))
vi.mock('@/utils/characterPreview', () => ({
  createCharacterPreview: () => viewer,
}))
let wrapper: VueWrapper | undefined
let intersect: (entries: { isIntersecting: boolean }[]) => void
beforeEach(() => {
  vi.clearAllMocks()
  viewer.load.mockResolvedValue({
    joints: 65,
    materials: [
      {
        name: '材质 1',
        meshes: ['网格 1'],
        roughness: 0.9,
        metalness: 0,
        textures: [{ label: '颜色', image: 'data:image/png;base64,AA==' }],
      },
    ],
  })
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      disconnect() {}
    },
  )
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(cb: typeof intersect) {
        intersect = cb
      }
      observe() {}
      disconnect() {}
    },
  )
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.unstubAllGlobals()
})
it('shows loaded joint and material information and supports keyboard rotation', async () => {
  wrapper = mount(CharacterDemo, {
    props: { modelUrl: '/test.glb', rotation: 0, name: '角色' },
  })
  intersect([{ isIntersecting: true }])
  await flushPromises()
  await wrapper.findAll('.character-modes button')[1].trigger('click')
  expect(viewer.setMode).toHaveBeenLastCalledWith('skeleton', 0)
  expect(wrapper.get('[aria-label="骨骼"]').attributes('title')).toContain(
    '65 个关节',
  )
  await wrapper.findAll('.character-modes button')[2].trigger('click')
  expect(wrapper.text()).toContain('网格 1 → 材质 1')
  expect(wrapper.text()).toContain('粗糙度 0.90')
  expect(wrapper.get('img').attributes('alt')).toBe('颜色贴图')
  await wrapper.get('canvas').trigger('keydown', { key: 'ArrowRight' })
  expect(viewer.rotate).toHaveBeenCalledWith(0.15)
  intersect([{ isIntersecting: false }])
  expect(viewer.setActive).toHaveBeenLastCalledWith(false)
})
it('keeps the actual asset download available when preview loading fails', async () => {
  viewer.load.mockRejectedValue(new Error('Invalid GLB'))
  wrapper = mount(CharacterDemo, {
    props: { modelUrl: '/test.glb', rotation: 0, name: '角色' },
  })
  intersect([{ isIntersecting: true }])
  await flushPromises()
  expect(wrapper.get('[role="status"]').text()).toContain('预览暂不可用')
  expect(wrapper.get('a[download]').attributes('href')).toBe(
    '/test.glb',
  )
  expect(
    wrapper
      .findAll('.character-modes button')
      .every((button) => button.attributes('disabled') !== undefined),
  ).toBe(true)
  expect(viewer.dispose).toHaveBeenCalledOnce()
})
it('disposes the renderer if unmounted while the asset is loading', async () => {
  let finish!: (value: null) => void
  viewer.load.mockReturnValue(
    new Promise((resolve) => {
      finish = resolve
    }),
  )
  wrapper = mount(CharacterDemo, {
    props: { modelUrl: '/test.glb', rotation: 0, name: '角色' },
  })
  intersect([{ isIntersecting: true }])
  await flushPromises()
  wrapper.unmount()
  wrapper = undefined
  finish(null)
  await flushPromises()
  expect(viewer.dispose).toHaveBeenCalledOnce()
})

it('waits until visible to load and hides skeleton tools for scene props', async () => {
  viewer.load.mockResolvedValue({ joints: 0, materials: [] })
  wrapper = mount(CharacterDemo, { props: { modelUrl: '/wings.glb', rotation: 0, name: '天使双翼' } })
  await flushPromises()
  expect(viewer.load).not.toHaveBeenCalled()
  intersect([{ isIntersecting: true }])
  await flushPromises()
  expect(viewer.load).toHaveBeenCalledWith('/wings.glb', 0)
  expect(wrapper.find('[aria-label="骨骼"]').exists()).toBe(false)
})

it('opens actual PBR maps and does not show texture multipliers as surface values', async () => {
  viewer.load.mockResolvedValue({ joints: 65, materials: [{
    name: '材质 1', meshes: ['网格 1'], roughness: 1, metalness: 1,
    textures: ['颜色', '法线', '粗糙度', '金属度'].map(label => ({ label, image: `data:image/png;base64,${label}` })),
  }] })
  wrapper = mount(CharacterDemo, { props: { modelUrl: '/pbr.glb', rotation: 0, name: '角色' } })
  intersect([{ isIntersecting: true }])
  await flushPromises()
  await wrapper.get('[aria-label="材质"]').trigger('click')
  expect(wrapper.find('.character-parameters').exists()).toBe(false)
  const dialog = wrapper.get('dialog').element as HTMLDialogElement
  dialog.showModal = vi.fn()
  dialog.close = vi.fn()
  await wrapper.get('[aria-label="查看法线贴图"]').trigger('click')
  await flushPromises()
  expect(dialog.showModal).toHaveBeenCalledOnce()
  expect(wrapper.get('dialog img').attributes('src')).toBe('data:image/png;base64,法线')
  await wrapper.get('[aria-label="关闭贴图"]').trigger('click')
  expect(dialog.close).toHaveBeenCalledOnce()
})
