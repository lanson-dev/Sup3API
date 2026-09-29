<template>
  <figure ref="stage" class="sh-mesh-stage" aria-label="三维形态交互预览">
    <div class="sh-mesh-orbit sh-mesh-orbit-one" aria-hidden="true"></div>
    <div class="sh-mesh-orbit sh-mesh-orbit-two" aria-hidden="true"></div>
    <span class="sh-mesh-caption">IMAGINATION, IN EVERY DIMENSION.</span>
    <div
      class="sh-mesh-surface"
      tabindex="0"
      role="group"
      aria-label="旋转三维模型"
      aria-describedby="mesh-instructions"
      @keydown="onKeydown"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerEnd"
      @pointercancel="onPointerEnd"
      @lostpointercapture="onPointerEnd"
    >
      <canvas
        v-show="supported"
        ref="canvas"
        aria-label="薄荷绿色的三叶结三维模型"
        role="img"
      ></canvas>
      <img
        v-if="!supported"
        class="sh-mesh-fallback"
        src="/sup3api-mark.svg"
        width="220"
        height="220"
        alt="Sup3API 三角网格标志"
      />
    </div>
    <div
      v-if="supported"
      class="sh-mesh-label sh-mesh-label-top"
      aria-hidden="true"
    >
      <span class="sh-cross">+</span> IDEAS INTO FORM
    </div>
    <div
      v-if="supported"
      class="sh-mesh-label sh-mesh-label-bottom"
      aria-hidden="true"
    >
      <span class="sh-cross">+</span>
      {{ wireframe ? 'WIREFRAME' : 'SURFACE' }} / 001
    </div>
    <template v-if="supported">
      <button
        class="sh-mesh-icon sh-mesh-turn sh-mesh-turn-left"
        aria-label="向左旋转模型"
        title="向左旋转模型"
        @click="turn(-1)"
      >
        <svg
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          aria-hidden="true"
        >
          <path d="m12 5-5 5 5 5" />
        </svg>
      </button>
      <button
        class="sh-mesh-icon sh-mesh-turn sh-mesh-turn-right"
        aria-label="向右旋转模型"
        title="向右旋转模型"
        @click="turn(1)"
      >
        <svg
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          aria-hidden="true"
        >
          <path d="m8 5 5 5-5 5" />
        </svg>
      </button>
    </template>
    <figcaption id="mesh-instructions">
      {{ supported ? '形态预览 · 拖动或使用方向键旋转' : '形态预览' }}
    </figcaption>
    <div v-if="supported" class="sh-mesh-controls">
      <div class="sh-mesh-segment" role="group" aria-label="模型显示方式">
        <button :aria-pressed="!wireframe" @click="setWireframe(false)">
          实体</button
        ><button :aria-pressed="wireframe" @click="setWireframe(true)">
          线框
        </button>
      </div>
      <button
        class="sh-mesh-icon"
        :aria-label="playing ? '暂停自动旋转' : '播放自动旋转'"
        :title="playing ? '暂停自动旋转' : '播放自动旋转'"
        @click="togglePlaying"
      >
        <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
          <path v-if="playing" d="M5 4h3v12H5zm7 0h3v12h-3Z" />
          <path v-else d="m6 3 10 7-10 7Z" />
        </svg>
      </button>
      <button
        class="sh-mesh-icon"
        aria-label="重置模型视角"
        title="重置模型视角"
        @click="resetView"
      >
        <svg
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          aria-hidden="true"
        >
          <path d="M3 8a7 7 0 1 1 1 7M3 3v5h5" />
        </svg>
      </button>
    </div>
  </figure>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { createKnotMesh, paintKnot } from '@/utils/landingMesh'

const canvas = ref<HTMLCanvasElement>()
const stage = ref<HTMLElement>()
const supported = ref(true)
const wireframe = ref(false)
const playing = ref(false)
const mesh = createKnotMesh()
const initialRotation = { x: -0.48, y: 0.32 }
let rotation = { ...initialRotation }
let context: CanvasRenderingContext2D | null = null
let frame = 0
let lastTime = 0
let visible = true
let width = 0
let height = 0
let pointer: { id: number; x: number; y: number; dragging: boolean } | undefined
let resizeObserver: ResizeObserver | undefined
let intersectionObserver: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined

function paint() {
  if (context && width && height)
    paintKnot(context, mesh, width, height, rotation, wireframe.value)
}
function cancelFrame() {
  cancelAnimationFrame(frame)
  frame = 0
  lastTime = 0
}
function animate(time: number) {
  frame = 0
  if (!playing.value || !visible || document.hidden || !supported.value) return
  // Cap expensive canvas painting at 30 fps; rotation speed stays time based.
  if (!lastTime || time - lastTime >= 32) {
    rotation.y += lastTime ? Math.min(time - lastTime, 80) * 0.00019 : 0
    lastTime = time
    paint()
  }
  frame = requestAnimationFrame(animate)
}
function syncAnimation() {
  cancelFrame()
  if (playing.value && visible && !document.hidden && supported.value)
    frame = requestAnimationFrame(animate)
}
function togglePlaying() {
  playing.value = !playing.value
  syncAnimation()
}
function pause() {
  playing.value = false
  syncAnimation()
}
function setWireframe(value: boolean) {
  wireframe.value = value
  paint()
}
function resetView() {
  pause()
  rotation = { ...initialRotation }
  paint()
}
function turn(direction: number) {
  pause()
  rotation.y += direction * 0.24
  paint()
}
function onKeydown(event: KeyboardEvent) {
  if (
    !['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home'].includes(
      event.key,
    )
  )
    return
  event.preventDefault()
  pause()
  if (event.key === 'Home') rotation = { ...initialRotation }
  else if (event.key === 'ArrowLeft') rotation.y -= 0.16
  else if (event.key === 'ArrowRight') rotation.y += 0.16
  else
    rotation.x = Math.max(
      -1.2,
      Math.min(1.2, rotation.x + (event.key === 'ArrowUp' ? -0.12 : 0.12)),
    )
  paint()
}
function onPointerDown(event: PointerEvent) {
  if (!event.isPrimary || event.button !== 0) return
  pointer = {
    id: event.pointerId,
    x: event.clientX,
    y: event.clientY,
    dragging: false,
  }
}
function onPointerMove(event: PointerEvent) {
  if (!pointer || pointer.id !== event.pointerId) return
  const dx = event.clientX - pointer.x
  const dy = event.clientY - pointer.y
  if (!pointer.dragging) {
    if (Math.abs(dx) + Math.abs(dy) < 5) return
    // Leave vertical touch gestures to the browser's page scrolling.
    if (event.pointerType === 'touch' && Math.abs(dy) > Math.abs(dx)) {
      pointer = undefined
      return
    }
    pointer.dragging = true
    ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
    pause()
  }
  rotation.y += dx * 0.008
  rotation.x = Math.max(-1.2, Math.min(1.2, rotation.x + dy * 0.005))
  pointer.x = event.clientX
  pointer.y = event.clientY
  paint()
}
function onPointerEnd(event: PointerEvent) {
  if (pointer?.id !== event.pointerId) return
  pointer = undefined
  const element = event.currentTarget as HTMLElement
  if (element.hasPointerCapture(event.pointerId))
    element.releasePointerCapture(event.pointerId)
}
function resize() {
  const element = canvas.value
  if (!element || !context) return
  const rect = element.getBoundingClientRect()
  width = rect.width
  height = rect.height
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  element.width = Math.round(width * ratio)
  element.height = Math.round(height * ratio)
  context.setTransform(ratio, 0, 0, ratio, 0, 0)
  paint()
}
function onMotionPreference(event: MediaQueryListEvent) {
  if (event.matches) pause()
}
onMounted(() => {
  try {
    context = canvas.value?.getContext('2d', { alpha: true }) ?? null
  } catch {
    context = null
  }
  supported.value = !!context
  if (!context) return
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  playing.value = !motionQuery.matches
  motionQuery.addEventListener('change', onMotionPreference)
  document.addEventListener('visibilitychange', syncAnimation)
  resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(canvas.value!)
  intersectionObserver = new IntersectionObserver(
    ([entry]) => {
      visible = entry.isIntersecting
      syncAnimation()
    },
    { threshold: 0.05 },
  )
  intersectionObserver.observe(stage.value!)
  resize()
  syncAnimation()
})
onBeforeUnmount(() => {
  cancelFrame()
  resizeObserver?.disconnect()
  intersectionObserver?.disconnect()
  motionQuery?.removeEventListener('change', onMotionPreference)
  document.removeEventListener('visibilitychange', syncAnimation)
})
</script>
