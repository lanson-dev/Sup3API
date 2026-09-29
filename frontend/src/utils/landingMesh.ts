import {
  ACESFilmicToneMapping,
  Color,
  DirectionalLight,
  Mesh,
  MeshPhysicalMaterial,
  PerspectiveCamera,
  PMREMGenerator,
  Scene,
  TorusKnotGeometry,
  Vector3,
  WebGLRenderer,
} from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js'

/** A local studio-lit object; no supplier calls or remote model downloads. */
export function createLandingScene(
  canvas: HTMLCanvasElement,
  onInteract: () => void,
) {
  const renderer = new WebGLRenderer({
    canvas,
    alpha: true,
    antialias: true,
    powerPreference: 'low-power',
  })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
  renderer.toneMapping = ACESFilmicToneMapping
  renderer.toneMappingExposure = 1.05
  const scene = new Scene()
  const camera = new PerspectiveCamera(38, 1, 0.1, 50)
  const initialPosition = new Vector3(0, 0.65, 7.3)
  camera.position.copy(initialPosition)
  const environment = new RoomEnvironment()
  const generator = new PMREMGenerator(renderer)
  const environmentMap = generator.fromScene(environment, 0.04)
  scene.environment = environmentMap.texture
  environment.dispose()
  generator.dispose()
  const geometry = new TorusKnotGeometry(1.22, 0.43, 192, 32)
  const material = new MeshPhysicalMaterial({
    color: '#a3e6d2',
    metalness: 0.86,
    roughness: 0.24,
    clearcoat: 1,
    clearcoatRoughness: 0.18,
    envMapIntensity: 1.45,
  })
  const model = new Mesh(geometry, material)
  model.rotation.set(0.3, 0, -0.35)
  scene.add(model)
  const keyLight = new DirectionalLight('#eafff9', 3.5)
  keyLight.position.set(-3, 5, 4)
  const rimLight = new DirectionalLight('#63aaff', 3)
  rimLight.position.set(4, 1, -2)
  scene.add(keyLight, rimLight)
  const controls = new OrbitControls(camera, canvas)
  controls.enableDamping = true
  controls.dampingFactor = 0.075
  controls.rotateSpeed = 0.55
  controls.enablePan = false
  controls.enableZoom = false
  controls.autoRotateSpeed = 0.65
  controls.minPolarAngle = 0.3
  controls.maxPolarAngle = Math.PI - 0.3
  // Vertical touch gestures still scroll the page; horizontal drags rotate.
  canvas.style.touchAction = 'pan-y'
  controls.update()
  controls.saveState()
  let active = false
  let frame = 0
  let lastTime = 0
  function wake() {
    if (active && !frame) frame = requestAnimationFrame(draw)
  }
  function draw(time: number) {
    frame = 0
    const delta = lastTime ? Math.min((time - lastTime) / 1000, 0.05) : 1 / 60
    lastTime = time
    const moving = controls.update(delta)
    renderer.render(scene, camera)
    if (moving || controls.autoRotate) wake()
  }
  function stop() {
    cancelAnimationFrame(frame)
    frame = 0
    lastTime = 0
  }
  function interact() {
    controls.autoRotate = false
    onInteract()
    wake()
  }
  controls.addEventListener('start', interact)
  controls.addEventListener('change', wake)
  return {
    setActive(value: boolean) {
      active = value
      if (active) wake()
      else stop()
    },
    resize() {
      const { width, height } = canvas.getBoundingClientRect()
      if (!width || !height) return
      renderer.setSize(width, height, false)
      camera.aspect = width / height
      camera.updateProjectionMatrix()
      wake()
    },
    setPlaying(value: boolean) {
      controls.autoRotate = value
      wake()
    },
    setWireframe(value: boolean) {
      material.wireframe = value
      material.color = new Color(value ? '#75d4bd' : '#a3e6d2')
      wake()
    },
    rotate(horizontal: number, vertical = 0) {
      interact()
      const offset = camera.position.clone().sub(controls.target)
      offset.applyAxisAngle(new Vector3(0, 1, 0), horizontal)
      offset.applyAxisAngle(
        new Vector3().crossVectors(camera.up, offset).normalize(),
        vertical,
      )
      camera.position.copy(controls.target).add(offset)
      wake()
    },
    reset() {
      interact()
      // Flush remaining damping before restoring the saved camera position.
      const damping = controls.enableDamping
      controls.enableDamping = false
      controls.update()
      controls.reset()
      controls.enableDamping = damping
      wake()
    },
    dispose() {
      active = false
      stop()
      controls.dispose()
      geometry.dispose()
      material.dispose()
      environmentMap.dispose()
      renderer.dispose()
    },
  }
}
