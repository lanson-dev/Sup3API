import {
  ACESFilmicToneMapping,
  Box3,
  DirectionalLight,
  HemisphereLight,
  Mesh,
  MeshStandardMaterial,
  PerspectiveCamera,
  Scene,
  SkeletonHelper,
  SkinnedMesh,
  Texture,
  Vector3,
  WebGLRenderer,
} from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { MeshoptDecoder } from 'three/addons/libs/meshopt_decoder.module.js'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'

export type PreviewMode = 'model' | 'skeleton' | 'materials'
export type MaterialInfo = {
  name: string
  meshes: string[]
  roughness: number
  metalness: number
  textures: { label: string; image: string }[]
}

export function createCharacterPreview(canvas: HTMLCanvasElement) {
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
  renderer.toneMapping = ACESFilmicToneMapping
  const scene = new Scene()
  const camera = new PerspectiveCamera(36, 1, 0.01, 100)
  const controls = new OrbitControls(camera, canvas)
  controls.enableDamping = true
  controls.enablePan = false
  controls.rotateSpeed = 0.6
  controls.minDistance = 2.8
  controls.maxDistance = 7
  canvas.style.touchAction = 'pan-y'
  scene.add(new HemisphereLight('#ffffff', '#6e776e', 2.2))
  const key = new DirectionalLight('#fff5e6', 2.4)
  key.position.set(-3, 4, 5)
  scene.add(key)
  const fill = new DirectionalLight('#e1efe9', 1)
  fill.position.set(3, 2, -3)
  scene.add(fill)
  const meshes: Mesh[] = []
  const materials: MeshStandardMaterial[] = []
  const original = new Map<
    MeshStandardMaterial,
    { opacity: number; transparent: boolean; depthWrite: boolean }
  >()
  let skeleton: SkeletonHelper | undefined
  let disposed = false,
    active = false,
    frame = 0
  function wake() {
    if (active && !disposed && !frame) frame = requestAnimationFrame(draw)
  }
  function draw() {
    frame = 0
    const moving = controls.update()
    renderer.render(scene, camera)
    if (moving) wake()
  }
  controls.addEventListener('change', wake)
  function releaseModel() {
    const textures = new Set<Texture>()
    for (const mesh of meshes) {
      mesh.geometry.dispose()
      if (mesh instanceof SkinnedMesh) mesh.skeleton.dispose()
    }
    for (const material of materials) {
      for (const value of Object.values(material))
        if (value instanceof Texture) textures.add(value)
      material.dispose()
    }
    for (const texture of textures) {
      texture.dispose()
      if (
        typeof ImageBitmap !== 'undefined' &&
        texture.image instanceof ImageBitmap
      )
        texture.image.close()
    }
    skeleton?.dispose()
  }
  return {
    async load(url: string, rotation: number) {
      const gltf = await new GLTFLoader()
        .setMeshoptDecoder(MeshoptDecoder)
        .loadAsync(url)
      const root = gltf.scene
      root.traverse((object) => {
        if (!(object instanceof Mesh)) return
        meshes.push(object)
        for (const material of Array.isArray(object.material)
          ? object.material
          : [object.material]) {
          if (
            material instanceof MeshStandardMaterial &&
            !materials.includes(material)
          ) {
            materials.push(material)
            original.set(material, {
              opacity: material.opacity,
              transparent: material.transparent,
              depthWrite: material.depthWrite,
            })
          }
        }
      })
      if (disposed) {
        releaseModel()
        return null
      }
      root.rotation.y += rotation
      const bounds = new Box3().setFromObject(root)
      const size = bounds.getSize(new Vector3())
      const center = bounds.getCenter(new Vector3())
      const scale = 2 / size.y
      root.scale.multiplyScalar(scale)
      root.position.sub(center.multiplyScalar(scale))
      scene.add(root)
      root.updateMatrixWorld(true)
      skeleton = new SkeletonHelper(root)
      skeleton.visible = false
      for (const material of Array.isArray(skeleton.material)
        ? skeleton.material
        : [skeleton.material]) {
        material.depthTest = false
        material.transparent = true
      }
      skeleton.renderOrder = 2
      scene.add(skeleton)
      camera.position.set(0.25, 0.1, 4.2)
      controls.target.set(0, 0, 0)
      const bones = new Set(
        meshes.flatMap((mesh) =>
          mesh instanceof SkinnedMesh ? mesh.skeleton.bones : [],
        ),
      )
      const info: MaterialInfo[] = materials.map((material, index) => ({
        name: `材质 ${index + 1}`,
        roughness: material.roughness,
        metalness: material.metalness,
        meshes: meshes.flatMap((mesh, i) =>
          (Array.isArray(mesh.material)
            ? mesh.material
            : [mesh.material]
          ).includes(material)
            ? [`网格 ${i + 1}`]
            : [],
        ),
        textures: (
          [
            ['颜色', material.map],
            ['法线', material.normalMap],
            ['粗糙度', material.roughnessMap],
            ['金属度', material.metalnessMap],
          ] as const
        ).flatMap(([label, texture]) => {
          if (!texture?.image) return []
          const thumbnail = document.createElement('canvas')
          thumbnail.width = thumbnail.height = 128
          thumbnail.getContext('2d')!.drawImage(texture.image, 0, 0, 128, 128)
          return [{ label, image: thumbnail.toDataURL('image/webp') }]
        }),
      }))
      wake()
      return { joints: bones.size, materials: info }
    },
    setMode(mode: PreviewMode, selected = 0) {
      if (skeleton) skeleton.visible = mode === 'skeleton'
      for (const [index, material] of materials.entries()) {
        const dimmed =
          mode === 'skeleton' || (mode === 'materials' && index !== selected)
        Object.assign(material, original.get(material))
        if (dimmed) {
          material.opacity = mode === 'skeleton' ? 0.18 : 0.12
          material.transparent = true
          material.depthWrite = false
        }
        material.needsUpdate = true
      }
      wake()
    },
    rotate(delta: number) {
      const offset = camera.position.clone().sub(controls.target)
      offset.applyAxisAngle(new Vector3(0, 1, 0), delta)
      camera.position.copy(controls.target).add(offset)
      wake()
    },
    resize() {
      const { width, height } = canvas.getBoundingClientRect()
      if (!width || !height) return
      renderer.setSize(width, height, false)
      camera.aspect = width / height
      camera.updateProjectionMatrix()
      wake()
    },
    setActive(value: boolean) {
      active = value
      if (value) wake()
      else {
        cancelAnimationFrame(frame)
        frame = 0
      }
    },
    dispose() {
      disposed = true
      cancelAnimationFrame(frame)
      controls.dispose()
      releaseModel()
      renderer.dispose()
    },
  }
}
