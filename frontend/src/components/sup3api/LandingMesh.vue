<template>
  <figure
    ref="stage"
    class="sh-mesh-stage"
    :class="{ compact, moving: playing && visible }"
    aria-label="交互三维模型"
  >
    <canvas
      v-show="supported"
      ref="canvas"
      class="sh-mesh-surface"
      :tabindex="ready ? 0 : -1"
      :aria-busy="!ready"
      role="img"
      aria-label="三维模型，可拖动或用方向键旋转，Home 重置"
      @keydown="onKeydown"
      @webglcontextlost.prevent="onContextLost"
    />
    <img
      v-if="!supported"
      class="sh-mesh-fallback"
      src="/sup3api-mark.svg"
      width="180"
      height="180"
      alt="Sup3API"
    />
    <p v-if="!supported" class="sh-mesh-unavailable" role="status">
      当前浏览器无法显示三维预览
    </p>
    <p v-else-if="!ready" class="sh-mesh-unavailable" role="status">加载三维预览…</p>
    <template v-else>
      <button
        class="sh-mesh-turn sh-mesh-left"
        aria-label="向左旋转模型"
        @click="viewer?.rotate(-0.24)"
      >
        ‹
      </button>
      <button
        class="sh-mesh-turn sh-mesh-right"
        aria-label="向右旋转模型"
        @click="viewer?.rotate(0.24)"
      >
        ›
      </button>
      <div class="sh-mesh-controls">
        <div class="sh-mesh-segment" role="group" aria-label="模型显示方式">
          <button :aria-pressed="!wireframe" @click="setWireframe(false)">
            实体
          </button>
          <button :aria-pressed="wireframe" @click="setWireframe(true)">
            线框
          </button>
        </div>
        <button
          class="sh-mesh-icon"
          :aria-label="playing ? '暂停自动旋转' : '播放自动旋转'"
          @click="setPlaying(!playing)"
        >
          <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path v-if="playing" d="M5 4h3v12H5zm7 0h3v12h-3Z" />
            <path v-else d="m6 3 10 7-10 7Z" />
          </svg>
        </button>
        <button
          class="sh-mesh-icon"
          aria-label="重置模型视角"
          @click="viewer?.reset()"
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
    </template>
  </figure>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { createLandingScene } from '@/utils/landingMesh'

defineProps<{ compact?: boolean }>()
const canvas = ref<HTMLCanvasElement>()
const stage = ref<HTMLElement>()
const supported = ref(true)
const ready = ref(false)
const playing = ref(false)
const visible = ref(false)
const wireframe = ref(false)
let viewer: ReturnType<typeof createLandingScene> | undefined
let resizeObserver: ResizeObserver | undefined
let intersectionObserver: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
let disposed = false
function setPlaying(value: boolean) {
  playing.value = value
  viewer?.setPlaying(value)
}
function setWireframe(value: boolean) {
  wireframe.value = value
  viewer?.setWireframe(value)
}
function syncVisibility() {
  viewer?.setActive(visible.value && !document.hidden)
}
function onMotion(event: MediaQueryListEvent) {
  if (event.matches) setPlaying(false)
}
function onContextLost() {
  supported.value = false
  playing.value = false
  viewer?.dispose()
  viewer = undefined
}
function onKeydown(event: KeyboardEvent) {
  if (event.ctrlKey || event.metaKey || event.altKey) return
  if (
    !['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home'].includes(
      event.key,
    )
  )
    return
  event.preventDefault()
  if (event.key === 'Home') viewer?.reset()
  else
    viewer?.rotate(
      event.key === 'ArrowLeft' ? -0.16 : event.key === 'ArrowRight' ? 0.16 : 0,
      event.key === 'ArrowUp' ? -0.12 : event.key === 'ArrowDown' ? 0.12 : 0,
    )
}
onMounted(async () => {
  try {
    const { createLandingScene } = await import('@/utils/landingMesh')
    if (disposed || !canvas.value) return
    viewer = createLandingScene(canvas.value, () => {
      playing.value = false
    })
    ready.value = true
    motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPlaying(!motionQuery.matches)
    motionQuery.addEventListener('change', onMotion)
    document.addEventListener('visibilitychange', syncVisibility)
    resizeObserver = new ResizeObserver(() => viewer?.resize())
    resizeObserver.observe(canvas.value)
    intersectionObserver = new IntersectionObserver(([entry]) => {
      visible.value = entry.isIntersecting
      viewer?.resize()
      syncVisibility()
    })
    intersectionObserver.observe(stage.value!)
    viewer.resize()
  } catch {
    if (!disposed) onContextLost()
  }
})
onBeforeUnmount(() => {
  disposed = true
  resizeObserver?.disconnect()
  intersectionObserver?.disconnect()
  motionQuery?.removeEventListener('change', onMotion)
  document.removeEventListener('visibilitychange', syncVisibility)
  viewer?.dispose()
})
</script>

<style scoped>
.sh-mesh-stage {
  position: relative;
  height: 560px;
  min-width: 0;
  isolation: isolate;
}
.sh-mesh-stage::before {
  content: '';
  position: absolute;
  inset: 8% -8% 15%;
  z-index: -1;
  border-radius: 50%;
  background: radial-gradient(
    ellipse,
    #65d2af30,
    #5186af0c 48%,
    transparent 70%
  );
  filter: blur(25px);
  pointer-events: none;
}
.sh-mesh-stage.moving::before {
  animation: mesh-light 9s ease-in-out infinite alternate;
}
.sh-mesh-stage::after {
  content: '';
  position: absolute;
  inset: auto 23% 14%;
  height: 20px;
  background: #0005;
  filter: blur(16px);
  border-radius: 50%;
  z-index: -1;
}
.sh-mesh-surface {
  display: block;
  width: 100%;
  height: calc(100% - 72px);
  cursor: grab;
  touch-action: pan-y;
}
.sh-mesh-surface:active {
  cursor: grabbing;
}
.sh-mesh-controls {
  position: absolute;
  inset: auto 0 8px;
  display: flex;
  justify-content: center;
  gap: 10px;
  align-items: center;
}
.sh-mesh-stage button {
  flex-shrink: 0;
  border: 1px solid #ffffff26;
  background: #142420e6;
  color: #d8e8e0;
  border-radius: 50%;
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  transition:
    background 180ms,
    transform 180ms;
}
.sh-mesh-stage button:hover {
  background: #2d4b40;
}
.sh-mesh-stage button:active {
  transform: scale(0.95);
}
.sh-mesh-segment {
  display: flex;
  padding: 4px;
  background: #142420e6;
  border: 1px solid #ffffff26;
  border-radius: 40px;
}
.sh-mesh-segment button {
  width: 66px;
  border: 0;
  border-radius: 40px;
  background: transparent;
  font-size: 14px;
}
.sh-mesh-segment button[aria-pressed='true'] {
  background: #e7f4ed;
  color: #18362b;
}
.sh-mesh-icon svg {
  width: 18px;
  height: 18px;
}
.sh-mesh-turn {
  position: absolute;
  top: 43%;
  font-size: 26px !important;
}
.sh-mesh-left {
  left: 0;
}
.sh-mesh-right {
  right: 0;
}
.sh-mesh-fallback {
  position: absolute;
  left: 50%;
  top: 42%;
  transform: translate(-50%, -50%);
}
.sh-mesh-unavailable {
  position: absolute;
  bottom: 20%;
  width: 100%;
  text-align: center;
  font-size: 14px;
}
.compact {
  height: 420px;
  width: 100%;
}
.compact .sh-mesh-controls {
  gap: 6px;
}
.compact .sh-mesh-segment button {
  width: 56px;
}
@keyframes mesh-light {
  to {
    transform: translate(-4%, 5%) scale(1.12);
    opacity: 0.65;
  }
}
@media (max-width: 760px) {
  .sh-mesh-stage {
    height: 400px;
  }
  .compact {
    height: 350px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sh-mesh-stage::before {
    animation: none !important;
  }
  .sh-mesh-stage button {
    transition: none;
  }
}
</style>
