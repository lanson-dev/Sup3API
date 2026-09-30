import {
  BufferGeometry, Float32BufferAttribute, PerspectiveCamera, Points,
  Scene, ShaderMaterial, Vector2, Vector3, Vector4, WebGLRenderer,
} from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'

export type NodeProjection = { x: number; y: number; depth: number }

export function createLandingScene(
  canvas: HTMLCanvasElement,
  onProject: (points: NodeProjection[]) => void,
) {
  const stage = canvas.parentElement!
  const destination = stage.closest('main')?.querySelector<HTMLElement>('#capabilities')
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: false })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.75))
  const scene = new Scene()
  const camera = new PerspectiveCamera(37, 1, 0.1, 30)
  camera.position.set(3.3, 1.7, 5.5)
  const center = new Vector3(0, -0.20625, 0)
  const vertices = [
    new Vector3(0, 1.65, 0), new Vector3(-1.516, -.825, .875),
    new Vector3(1.516, -.825, .875), new Vector3(0, -.825, -1.75),
  ]
  const edges = [[0, 1], [0, 2], [0, 3], [1, 2], [2, 3], [3, 1]]
  const faces = [[0, 1, 2], [0, 2, 3], [0, 3, 1], [1, 3, 2]]
  const count = window.innerWidth < 760 ? 8500 : 16000
  const positions = new Float32Array(count * 3)
  const seeds = new Float32Array(count * 3)
  const ids = new Float32Array(count)
  // Stable sampling keeps the same grains through gathering, scrolling and resizing.
  let randomState = 73
  const random = () => ((randomState = (Math.imul(randomState, 1664525) + 1013904223) >>> 0) / 4294967296)
  const point = new Vector3()
  for (let i = 0; i < count; i++) {
    const kind = random(), a = random(), b = random(), c = random()
    if (kind < .32) {
      const edge = edges[Math.floor(a * edges.length)]
      point.lerpVectors(vertices[edge[0]], vertices[edge[1]], b)
      point.add(new Vector3(random() - .5, random() - .5, random() - .5).multiplyScalar(.07))
    } else if (kind < .61) {
      const face = faces[Math.floor(a * faces.length)], root = Math.sqrt(b)
      point.copy(vertices[face[0]]).multiplyScalar(1 - root)
        .addScaledVector(vertices[face[1]], root * (1 - c))
        .addScaledVector(vertices[face[2]], root * c)
    } else if (kind > .93 && kind < .96) {
      point.lerpVectors(center, vertices[Math.floor(a * 4)], b)
      point.add(new Vector3(random() - .5, random() - .5, random() - .5).multiplyScalar(.06))
    } else {
      const radius = kind < .93 ? .44 * (.72 + c * .28) : 2.3 * c
      const angle = a * Math.PI * 2, y = 2 * b - 1, ring = Math.sqrt(1 - y * y)
      point.set(Math.cos(angle) * ring, y, Math.sin(angle) * ring).multiplyScalar(radius).add(center)
    }
    point.toArray(positions, i * 3)
    seeds.set([random(), random(), kind], i * 3)
    ids[i] = i
  }
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new Float32BufferAttribute(positions, 3))
  geometry.setAttribute('seed', new Float32BufferAttribute(seeds, 3))
  geometry.setAttribute('indexId', new Float32BufferAttribute(ids, 1))
  const uniforms = {
    time: { value: 0 }, intro: { value: 0 }, morph: { value: 0 }, motion: { value: 1 },
    viewport: { value: new Vector2() }, rect: { value: new Vector4() },
    grid: { value: new Vector3() }, pixelRatio: { value: renderer.getPixelRatio() },
    pointer: { value: new Vector2(10, 10) }, node: { value: new Vector3() },
    highlight: { value: 0 },
  }
  const material = new ShaderMaterial({
    uniforms, transparent: true, depthWrite: false, depthTest: false,
    vertexShader: `
      attribute vec3 seed;
      attribute float indexId;
      uniform float time, intro, morph, motion, pixelRatio, highlight;
      uniform vec2 viewport, pointer;
      uniform vec4 rect;
      uniform vec3 grid, node;
      varying float alpha, tone, star;
      varying vec3 grainColor;
      void main() {
        float pi = 3.14159265;
        float assembly = smoothstep(seed.x * .22, 1., intro);
        vec3 p = position;
        float core = step(.61, seed.z) * (1. - step(.93, seed.z));
        float angle = time * .11 * core + (1. - assembly) * (1.5 + seed.x);
        p.xz = mat2(cos(angle), -sin(angle), sin(angle), cos(angle)) * p.xz;
        p += motion * .012 * sin(time * .6 + seed * 20.);
        p += (1. - assembly) * vec3(sin(seed.x * 40.), cos(seed.y * 31.), sin(seed.y * 53.)) * 2.6;
        vec4 projected = projectionMatrix * modelViewMatrix * vec4(p, 1.);
        vec2 source = projected.xy / projected.w * rect.zw + rect.xy;
        float t = smoothstep(seed.y * .12, .87 + seed.x * .13, morph);
        float cellId = mod(indexId, grid.x * grid.y);
        vec2 cell = vec2(mod(cellId, grid.x), floor(cellId / grid.x));
        vec2 target = vec2((cell.x + .5) * 32., grid.z + (cell.y + .5) * 32.);
        target = target / viewport * vec2(2., -2.) + vec2(-1., 1.);
        vec2 bend = vec2(sin(seed.x * 6.28), cos(seed.y * 6.28)) * .16;
        vec2 screen = mix(source, target, t) + bend * sin(t * pi) * motion;
        vec2 away = screen - pointer;
        screen += away * exp(-dot(away, away) * 38.) * .13 * (1. - t) * motion;
        float survives = 1. - step(grid.x * grid.y, indexId);
        float surface = step(.32, seed.z) * (1. - step(.61, seed.z));
        float dust = step(.93, seed.z);
        float brightness = mix(.6, .12, surface) * mix(1., .22, dust);
        vec3 fromCore = p - vec3(0., -.20625, 0.);
        vec3 ray = normalize(node - vec3(0., -.20625, 0.));
        float along = dot(fromCore, ray);
        float distanceToRay = length(fromCore - ray * along);
        float stream = exp(-distanceToRay * 16.) * step(0., along)
          * pow(.5 + .5 * cos(along * 12. - time * 4.), 6.) * highlight;
        star = step(.976, seed.y) * (1. - t);
        alpha = mix(brightness + stream * .8 + star * .35, .3 * survives, t) * smoothstep(0., .3, assembly);
        tone = t;
        grainColor = mix(vec3(.7, .85, .93), vec3(.89, .93, .83), seed.x);
        grainColor = mix(grainColor, vec3(.92, .74, .52), step(.9, seed.x));
        gl_PointSize = mix((1.2 + seed.x * .9 + stream) * (1. + star * 4.), 2.2, t) * pixelRatio;
        gl_Position = vec4(screen, 0., 1.);
      }
    `,
    fragmentShader: `
      varying float alpha, tone, star;
      varying vec3 grainColor;
      void main() {
        float radius = length(gl_PointCoord - .5);
        float edge = mix(1. - smoothstep(.18, .5, radius), exp(-radius * radius * 80.) + .13 * exp(-radius * radius * 8.), star);
        vec3 color = mix(grainColor, vec3(.28, .38, .32), tone);
        gl_FragColor = vec4(color, alpha * edge);
      }
    `,
  })
  const particles = new Points(geometry, material)
  particles.frustumCulled = false
  scene.add(particles)
  const controls = new OrbitControls(camera, stage)
  controls.target.copy(center)
  controls.enableDamping = true
  controls.dampingFactor = .055
  controls.rotateSpeed = .4
  controls.enablePan = controls.enableZoom = false
  controls.autoRotateSpeed = .24
  controls.minPolarAngle = .5
  controls.maxPolarAngle = Math.PI - .5
  stage.style.touchAction = 'pan-y'
  let active = false, playing = false, disposed = false, frame = 0, lastTime = 0
  let targetMorph = 0, elapsed = 0, selected: number | null = null
  const screen = new Vector3()
  function wake() {
    if (active && !disposed && !frame) frame = requestAnimationFrame(draw)
  }
  function layout() {
    const r = stage.getBoundingClientRect(), width = window.innerWidth, height = window.innerHeight
    const section = destination?.getBoundingClientRect()
    uniforms.rect.value.set((r.left + r.width / 2) / width * 2 - 1,
      1 - (r.top + r.height / 2) / height * 2, r.width / width, r.height / height)
    uniforms.grid.value.set(Math.ceil(width / 32), Math.ceil((section?.height || height) / 32), section?.top || 0)
    const end = (section?.top ?? height) + window.scrollY
    targetMorph = Math.max(0, Math.min(1, window.scrollY / Math.max(end - height * .2, 1)))
    controls.enabled = targetMorph < .2
    wake()
  }
  function draw(time: number) {
    frame = 0
    const delta = lastTime ? Math.min((time - lastTime) / 1000, .05) : 1 / 60
    lastTime = time
    const settling = Math.abs(targetMorph - uniforms.morph.value) > .0001
    uniforms.morph.value = playing && settling
      ? uniforms.morph.value + (targetMorph - uniforms.morph.value) * (1 - Math.exp(-delta * 12))
      : targetMorph
    if (playing) elapsed += delta
    const highlight = selected === null ? 0 : 1
    uniforms.highlight.value = playing
      ? uniforms.highlight.value + (highlight - uniforms.highlight.value) * (1 - Math.exp(-delta * 10))
      : highlight
    uniforms.time.value = elapsed
    uniforms.intro.value = playing ? Math.min(1, elapsed / 2.2) : 1
    controls.autoRotate = playing && uniforms.morph.value < .99
    const moving = controls.update(delta)
    camera.updateMatrixWorld()
    renderer.render(scene, camera)
    onProject(uniforms.morph.value < .18 ? vertices.map(position => {
      screen.copy(position).project(camera)
      return { x: (screen.x + 1) * 50, y: (1 - screen.y) * 50, depth: screen.z }
    }) : [])
    if (moving || settling || (playing && uniforms.morph.value < .999)) wake()
  }
  function stop() { cancelAnimationFrame(frame); frame = lastTime = 0 }
  function pointer(event: PointerEvent) {
    if (event.pointerType === 'touch') return
    uniforms.pointer.value.set(event.clientX / window.innerWidth * 2 - 1, 1 - event.clientY / window.innerHeight * 2)
    wake()
  }
  function leave() { uniforms.pointer.value.set(10, 10); wake() }
  controls.addEventListener('change', wake)
  window.addEventListener('scroll', layout, { passive: true })
  stage.addEventListener('pointermove', pointer)
  stage.addEventListener('pointerleave', leave)
  return {
    setNode(index: number | null) {
      selected = index
      uniforms.node.value.copy(vertices[selected ?? 0])
      controls.autoRotateSpeed = selected === null ? .24 : .06
      wake()
    },
    setActive(value: boolean) { active = value; if (active) { layout(); wake() } else stop() },
    resize() {
      const { width, height } = stage.getBoundingClientRect()
      if (!width || !height) return
      renderer.setSize(window.innerWidth, window.innerHeight, false)
      uniforms.viewport.value.set(window.innerWidth, window.innerHeight)
      camera.aspect = width / height
      camera.zoom = Math.min(1, camera.aspect / .95)
      camera.updateProjectionMatrix()
      layout()
    },
    setPlaying(value: boolean) { playing = value; uniforms.motion.value = value ? 1 : 0; wake() },
    dispose() {
      disposed = true; active = false; stop()
      window.removeEventListener('scroll', layout)
      stage.removeEventListener('pointermove', pointer)
      stage.removeEventListener('pointerleave', leave)
      controls.dispose(); geometry.dispose(); material.dispose(); renderer.dispose()
    },
  }
}
