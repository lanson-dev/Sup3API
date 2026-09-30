import {
  ACESFilmicToneMapping,
  Box3,
  DirectionalLight,
  HemisphereLight,
  Mesh,
  MeshStandardMaterial,
  PerspectiveCamera,
  PlaneGeometry,
  PMREMGenerator,
  Scene,
  ShadowMaterial,
  SkeletonHelper,
  SkinnedMesh,
  Texture,
  Vector3,
  VSMShadowMap,
  WebGLRenderer,
} from 'three'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { MeshoptDecoder } from 'three/addons/libs/meshopt_decoder.module.js'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js'

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
  renderer.shadowMap.enabled = true
  renderer.shadowMap.type = VSMShadowMap
  renderer.shadowMap.autoUpdate = false
  const scene = new Scene()
  const studio = new RoomEnvironment()
  const pmrem = new PMREMGenerator(renderer)
  const environment = pmrem.fromScene(studio, 0.04)
  scene.environment = environment.texture
  scene.environmentIntensity = 0.3
  studio.dispose()
  pmrem.dispose()
  const camera = new PerspectiveCamera(36, 1, 0.01, 100)
  const controls = new OrbitControls(camera, canvas)
  controls.enableDamping = true
  controls.enablePan = false
  controls.rotateSpeed = 0.6
  controls.minDistance = 2.8
  controls.maxDistance = 7
  canvas.style.touchAction = 'pan-y'
  scene.add(new HemisphereLight('#ffffff', '#7b827c', 1.5))
  const key = new DirectionalLight('#fff5e6', 2)
  key.position.set(-2, 8, 4)
  key.castShadow = true
  key.shadow.mapSize.set(512, 512)
  Object.assign(key.shadow.camera, {
    left: -2, right: 2, top: 2, bottom: -2, near: 0.1, far: 15,
  })
  key.shadow.normalBias = 0.025
  key.shadow.radius = 12
  key.shadow.blurSamples = 8
  scene.add(key)
  const fill = new DirectionalLight('#e1efe9', 0.8)
  fill.position.set(3, 2, -3)
  scene.add(fill)
  const ground = new Mesh(
    new PlaneGeometry(20, 20),
    new ShadowMaterial({ opacity: 0.12 }),
  )
  ground.rotation.x = -Math.PI / 2
  ground.position.y = -1.01
  ground.receiveShadow = true
  scene.add(ground)
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
        object.castShadow = true
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
      camera.position.set(0.15, 0.18, 3.7)
      renderer.shadowMap.needsUpdate = true
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
            ['颜色', material.map, -1],
            ['法线', material.normalMap, -1],
            ['粗糙度', material.roughnessMap, 1],
            ['金属度', material.metalnessMap, 2],
            ['遮蔽', material.aoMap, 0],
            ['自发光', material.emissiveMap, -1],
          ] as const
        ).flatMap(([label, texture, channel]) => {
          if (!texture?.image) return []
          const thumbnail = document.createElement('canvas')
          thumbnail.width = thumbnail.height = 512
          const context = thumbnail.getContext('2d')!
          context.drawImage(texture.image, 0, 0, 512, 512)
          // glTF packs roughness in G, metalness in B and occlusion in R.
          if (channel >= 0) {
            const pixels = context.getImageData(0, 0, 512, 512)
            for (let i = 0; i < pixels.data.length; i += 4) {
              const value = pixels.data[i + channel]
              pixels.data[i] = pixels.data[i + 1] = pixels.data[i + 2] = value
            }
            context.putImageData(pixels, 0, 0)
          }
          return [{ label, image: thumbnail.toDataURL('image/png') }]
        }),
      }))
      wake()
      return { joints: bones.size, materials: info }
    },
    setMode(mode: PreviewMode, selected = 0) {
      if (skeleton) skeleton.visible = mode === 'skeleton'
      ground.visible = mode !== 'skeleton'
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
      ground.geometry.dispose()
      ground.material.dispose()
      key.shadow.dispose()
      environment.dispose()
      renderer.dispose()
    },
  }
}
