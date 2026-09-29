<template>
  <figure ref="stage" class="sh-mesh-stage" aria-label="Sup3API 三维标志">
    <canvas
      v-show="supported"
      ref="canvas"
      class="sh-mesh-surface"
      role="img"
      aria-label="缓慢旋转的哑光三角形"
      @webglcontextlost.prevent="onContextLost"
    />
    <img
      v-if="!supported"
      src="/sup3api-mark.svg"
      width="180"
      height="180"
      alt="Sup3API"
    />
  </figure>
</template>
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { createLandingScene } from '@/utils/landingMesh'
const canvas = ref<HTMLCanvasElement>()
const stage = ref<HTMLElement>()
const supported = ref(true)
let viewer: ReturnType<typeof createLandingScene> | undefined
let resizeObserver: ResizeObserver | undefined
let intersectionObserver: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
let visible = false,
  disposed = false
function syncVisibility() {
  viewer?.setActive(visible && !document.hidden)
}
function onMotion() {
  viewer?.setPlaying(!motionQuery?.matches)
}
function onContextLost() {
  supported.value = false
  viewer?.dispose()
  viewer = undefined
}
onMounted(async () => {
  try {
    const { createLandingScene } = await import('@/utils/landingMesh')
    if (disposed || !canvas.value) return
    viewer = createLandingScene(canvas.value)
    motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    onMotion()
    motionQuery.addEventListener('change', onMotion)
    document.addEventListener('visibilitychange', syncVisibility)
    resizeObserver = new ResizeObserver(() => viewer?.resize())
    resizeObserver.observe(canvas.value)
    intersectionObserver = new IntersectionObserver(([entry]) => {
      visible = entry.isIntersecting
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
  height: 540px;
  min-width: 0;
  display: grid;
  place-items: center;
}
.sh-mesh-surface {
  display: block;
  width: 100%;
  height: 100%;
  cursor: grab;
  touch-action: pan-y;
}
.sh-mesh-surface:active {
  cursor: grabbing;
}
@media (max-width: 760px) {
  .sh-mesh-stage {
    height: 360px;
  }
}
</style>
