import {
  ACESFilmicToneMapping, AdditiveBlending, BufferGeometry, CapsuleGeometry,
  DoubleSide, Float32BufferAttribute, Group, HemisphereLight, Mesh, MeshBasicMaterial,
  MeshPhysicalMaterial, MeshStandardMaterial, PerspectiveCamera,
  PMREMGenerator, Scene, SphereGeometry, Vector3, WebGLRenderer,
} from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js'

export type NodeProjection = { x: number; y: number; depth: number }

export function createLandingScene(
  canvas: HTMLCanvasElement,
  onProject: (points: NodeProjection[]) => void,
) {
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
  renderer.toneMapping = ACESFilmicToneMapping
  const scene = new Scene()
  const studio = new RoomEnvironment()
  const pmrem = new PMREMGenerator(renderer)
  const environment = pmrem.fromScene(studio, 0.06)
  scene.environment = environment.texture
  scene.environmentIntensity = 0.65
  studio.dispose()
  pmrem.dispose()
  scene.add(new HemisphereLight('#e7f7e7', '#183c32', 1.4))

  const camera = new PerspectiveCamera(37, 1, 0.1, 30)
  camera.position.set(3, 1.8, 5.1)
  const model = new Group()
  scene.add(model)
  // A regular tetrahedron: three base vertices and one apex, never a flat extrusion.
  const vertices = [
    new Vector3(0, 1.65, 0),
    new Vector3(-1.516, -0.825, 0.875),
    new Vector3(1.516, -0.825, 0.875),
    new Vector3(0, -0.825, -1.75),
  ]
  const center = new Vector3(0, -0.20625, 0)
  const frameMaterial = new MeshStandardMaterial({
    color: '#8d9f90', metalness: 0.65, roughness: 0.36,
  })
  const nodeGeometry = new SphereGeometry(0.125, 32, 20)
  const nodes = vertices.map((position) => {
    const material = new MeshStandardMaterial({
      color: '#c0cdb4', metalness: 0.45, roughness: 0.3,
      emissive: '#b7e77c', emissiveIntensity: 0.03,
    })
    const node = new Mesh(nodeGeometry, material)
    node.position.copy(position)
    model.add(node)
    return node
  })
  const direction = new Vector3(0, 1, 0)
  const edgeGeometry = new CapsuleGeometry(0.045, vertices[0].distanceTo(vertices[1]) - 0.09, 5, 16)
  for (let a = 0; a < vertices.length; a++) {
    for (let b = a + 1; b < vertices.length; b++) {
      const edge = new Mesh(edgeGeometry, frameMaterial)
      edge.position.copy(vertices[a]).add(vertices[b]).multiplyScalar(0.5)
      edge.quaternion.setFromUnitVectors(direction, vertices[b].clone().sub(vertices[a]).normalize())
      model.add(edge)
    }
  }
  const panelGeometry = new BufferGeometry()
  const panelVertices = vertices.map(v => v.clone().sub(center).multiplyScalar(0.88).add(center))
  panelGeometry.setAttribute('position', new Float32BufferAttribute(
    [0, 1, 2, 0, 2, 3, 0, 3, 1, 1, 3, 2].flatMap(i => panelVertices[i].toArray()), 3,
  ))
  panelGeometry.computeVertexNormals()
  const panelMaterial = new MeshStandardMaterial({
    color: '#a7c8a8', metalness: 0.2, roughness: 0.4,
    transparent: true, opacity: 0.055, side: DoubleSide, depthWrite: false,
  })
  model.add(new Mesh(panelGeometry, panelMaterial))
  const coreMaterial = new MeshPhysicalMaterial({
    color: '#bfdd79', emissive: '#a8ce64', emissiveIntensity: 0.2,
    metalness: 0.12, roughness: 0.36, clearcoat: 0.5, clearcoatRoughness: 0.32,
  })
  const core = new Mesh(new SphereGeometry(0.46, 48, 32), coreMaterial)
  core.position.copy(center)
  model.add(core)
  const pathGeometry = new CapsuleGeometry(0.008, center.distanceTo(vertices[0]) - 0.016, 2, 6)
  const paths = vertices.map((position) => {
    const path = new Mesh(pathGeometry, new MeshBasicMaterial({
      color: '#cdecad', transparent: true, opacity: 0.09, depthWrite: false,
    }))
    path.position.copy(center).add(position).multiplyScalar(0.5)
    path.quaternion.setFromUnitVectors(direction, position.clone().sub(center).normalize())
    model.add(path)
    return path
  })
  const pulseGeometry = new SphereGeometry(0.038, 10, 8)
  const pulses = Array.from({ length: 12 }, () => {
    const pulse = new Mesh(pulseGeometry, new MeshBasicMaterial({
      color: '#e8ffc3', transparent: true, blending: AdditiveBlending, depthWrite: false,
    }))
    pulse.visible = false
    model.add(pulse)
    return pulse
  })
  const controls = new OrbitControls(camera, canvas)
  controls.target.copy(center)
  controls.enableDamping = true
  controls.dampingFactor = 0.075
  controls.rotateSpeed = 0.55
  controls.enablePan = controls.enableZoom = false
  controls.autoRotateSpeed = 0.4
  controls.minPolarAngle = 0.45
  controls.maxPolarAngle = Math.PI - 0.45
  canvas.style.touchAction = 'pan-y'
  let active = false, playing = false, disposed = false
  let frame = 0, lastTime = 0, elapsed = 0, selected: number | null = null
  const screen = new Vector3()
  function wake() {
    if (active && !disposed && !frame) frame = requestAnimationFrame(draw)
  }
  function draw(time: number) {
    frame = 0
    const delta = lastTime ? Math.min((time - lastTime) / 1000, 0.05) : 1 / 60
    lastTime = time
    if (playing) elapsed += delta
    const moving = controls.update(delta)
    if (playing) core.rotation.y = elapsed * 0.16
    core.scale.setScalar(selected === null ? 1 : 1.025 + (playing ? Math.sin(elapsed * 3) * 0.015 : 0))
    pulses.forEach((pulse, i) => {
      pulse.visible = selected !== null
      if (selected === null) return
      const t = ((playing ? elapsed * 0.48 : 0.55) + Math.floor(i / 6) * 0.5 - (i % 6) * 0.025 + 1) % 1
      pulse.position.lerpVectors(center, vertices[selected], t)
      pulse.scale.setScalar(1 - (i % 6) * 0.12)
      pulse.material.opacity = (1 - (i % 6) / 6) * Math.sin(t * Math.PI)
    })
    renderer.render(scene, camera)
    onProject(vertices.map((position) => {
      screen.copy(position).project(camera)
      return { x: (screen.x + 1) * 50, y: (1 - screen.y) * 50, depth: screen.z }
    }))
    if (moving || playing) wake()
  }
  function stop() {
    cancelAnimationFrame(frame)
    frame = lastTime = 0
  }
  controls.addEventListener('change', wake)
  return {
    setNode(index: number | null) {
      selected = index
      controls.autoRotateSpeed = index === null ? 0.4 : 0.1
      nodes.forEach((node, i) => {
        node.material.emissiveIntensity = i === index ? 0.65 : 0.03
        node.material.color.set(i === index ? '#daf6ad' : '#c0cdb4')
        paths[i].material.opacity = i === index ? 0.35 : 0.09
      })
      wake()
    },
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
      camera.zoom = Math.min(1, camera.aspect / 0.95)
      camera.updateProjectionMatrix()
      wake()
    },
    setPlaying(value: boolean) {
      playing = controls.autoRotate = value
      wake()
    },
    dispose() {
      disposed = true
      active = false
      stop()
      controls.dispose()
      for (const geometry of [nodeGeometry, edgeGeometry, panelGeometry, core.geometry, pathGeometry, pulseGeometry]) geometry.dispose()
      for (const material of [frameMaterial, panelMaterial, coreMaterial, ...nodes.map(n => n.material), ...paths.map(p => p.material), ...pulses.map(p => p.material)]) material.dispose()
      environment.dispose()
      renderer.dispose()
    },
  }
}
