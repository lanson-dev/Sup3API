import {
  ACESFilmicToneMapping,
  DirectionalLight,
  ExtrudeGeometry,
  HemisphereLight,
  Mesh,
  MeshStandardMaterial,
  Path,
  PerspectiveCamera,
  Scene,
  Shape,
  WebGLRenderer,
} from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'

export function createLandingScene(canvas: HTMLCanvasElement) {
  const renderer = new WebGLRenderer({
    canvas,
    alpha: true,
    antialias: true,
    powerPreference: 'low-power',
  })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
  renderer.toneMapping = ACESFilmicToneMapping
  const scene = new Scene()
  const camera = new PerspectiveCamera(38, 1, 0.1, 50)
  camera.position.set(0, 0.4, 7.2)
  const outline = new Shape()
    .moveTo(0, 1.7)
    .lineTo(-1.65, -1.2)
    .lineTo(1.65, -1.2)
    .closePath()
  outline.holes.push(
    new Path()
      .moveTo(0, 0.85)
      .lineTo(0.82, -0.64)
      .lineTo(-0.82, -0.64)
      .closePath(),
  )
  const geometry = new ExtrudeGeometry(outline, {
    depth: 0.48,
    bevelEnabled: true,
    bevelSize: 0.12,
    bevelThickness: 0.12,
    bevelSegments: 12,
    steps: 1,
  })
  geometry.center()
  const material = new MeshStandardMaterial({
    color: '#a7ceb9',
    metalness: 0,
    roughness: 0.82,
  })
  const model = new Mesh(geometry, material)
  model.rotation.set(-0.12, 0.25, -0.12)
  scene.add(model, new HemisphereLight('#f0fff8', '#233b32', 2.6))
  const light = new DirectionalLight('#fffaf0', 3)
  light.position.set(-3, 5, 4)
  scene.add(light)
  const controls = new OrbitControls(camera, canvas)
  controls.enableDamping = true
  controls.dampingFactor = 0.075
  controls.rotateSpeed = 0.55
  controls.enablePan = controls.enableZoom = false
  controls.autoRotateSpeed = 0.65
  controls.minPolarAngle = 0.3
  controls.maxPolarAngle = Math.PI - 0.3
  canvas.style.touchAction = 'pan-y'
  let active = false,
    frame = 0,
    lastTime = 0
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
    dispose() {
      active = false
      stop()
      controls.dispose()
      geometry.dispose()
      material.dispose()
      renderer.dispose()
    },
  }
}
