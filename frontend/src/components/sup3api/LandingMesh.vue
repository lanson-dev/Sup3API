<template>
  <figure ref="stage" class="sh-mesh-stage" aria-label="Sup3API 三维能力核心" @keydown.esc="dismiss">
    <div class="sh-core-aura" :class="{ 'is-active': selected !== null }" aria-hidden="true"></div>
    <canvas
      v-show="supported"
      ref="canvas"
      class="sh-mesh-surface"
      role="img"
      aria-label="持续旋转的三棱锥与悬浮核心，可拖动查看"
      @pointerdown="dismiss"
      @webglcontextlost.prevent="onContextLost"
    />
    <template v-if="supported">
      <button
        v-for="(point, index) in points"
        :key="index"
        class="sh-capability-node"
        :class="{ 'is-active': selected === index }"
        :style="{ left: `${point.x}%`, top: `${point.y}%`, zIndex: Math.round((1 - point.depth) * 1000) }"
        :aria-label="capabilities[index].title"
        :aria-describedby="selected === index ? 'hero-node-tooltip' : undefined"
        @pointerenter="hover(index, $event)"
        @pointerleave="hovered = null"
        @focus="focused = index"
        @blur="focused = null"
        @click="($event.currentTarget as HTMLButtonElement).focus()"
      ><span aria-hidden="true"></span></button>
      <Transition name="node-tip">
        <div
          v-if="selected !== null && points[selected]"
          id="hero-node-tooltip"
          class="sh-node-tooltip"
          :class="{ 'is-below': points[selected].y < 25 }"
          role="tooltip"
          :style="tooltipStyle"
        >
          <Icon :name="capabilities[selected].icon" size="lg" aria-hidden="true" />
          <div>
            <strong>{{ capabilities[selected].title }}</strong>
            <span v-if="capabilities[selected].detail">{{ capabilities[selected].detail }}</span>
          </div>
        </div>
      </Transition>
    </template>
    <img v-else src="/sup3api-mark.svg" width="180" height="180" alt="Sup3API" />
  </figure>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { createLandingScene, NodeProjection } from '@/utils/landingMesh'
const capabilities = [
  { title: '更多能力', detail: '骨骼 · 动画 · 重拓扑', icon: 'grid' },
  { title: '文本生成', detail: '', icon: 'chat' },
  { title: '图像生成', detail: '', icon: 'sparkles' },
  { title: '三维生成', detail: '', icon: 'cube' },
] as const
const canvas = ref<HTMLCanvasElement>()
const stage = ref<HTMLElement>()
const supported = ref(true)
const points = ref<NodeProjection[]>([])
const hovered = ref<number | null>(null)
const focused = ref<number | null>(null)
const selected = computed(() => hovered.value ?? focused.value)
const tooltipStyle = computed(() => {
  const point = selected.value === null ? undefined : points.value[selected.value]
  return point ? {
    left: `${Math.max(22, Math.min(78, point.x))}%`,
    top: `${point.y}%`,
  } : undefined
})
let viewer: ReturnType<typeof createLandingScene> | undefined
let resizeObserver: ResizeObserver | undefined
let intersectionObserver: IntersectionObserver | undefined
let motionQuery: MediaQueryList | undefined
let visible = false, disposed = false
watch(selected, value => viewer?.setNode(value))
function hover(index: number, event: PointerEvent) {
  if (event.pointerType !== 'touch') hovered.value = index
}
function dismiss() {
  hovered.value = focused.value = null
  const element = document.activeElement
  if (element instanceof HTMLButtonElement && stage.value?.contains(element)) element.blur()
}
function syncVisibility() {
  viewer?.setActive(visible && !document.hidden)
}
function onMotion() {
  viewer?.setPlaying(!motionQuery?.matches)
}
function onContextLost() {
  supported.value = false
  dismiss()
  viewer?.dispose()
  viewer = undefined
}
onMounted(async () => {
  try {
    const { createLandingScene } = await import('@/utils/landingMesh')
    if (disposed || !canvas.value) return
    viewer = createLandingScene(canvas.value, value => { points.value = value })
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
  height: 580px;
  min-width: 0;
  display: grid;
  place-items: center;
  isolation: isolate;
}
.sh-core-aura {
  position: absolute;
  inset: 10%;
  border-radius: 50%;
  background: radial-gradient(ellipse, #c3e89220, #a6d9ac07 40%, transparent 68%);
  opacity: .7;
  transform: scale(.9);
  transition: opacity .7s, transform .9s;
  pointer-events: none;
  z-index: -1;
}
.sh-core-aura.is-active { opacity: 1; transform: scale(1.14); }
.sh-mesh-surface {
  display: block;
  width: 100%;
  height: 100%;
  cursor: grab;
  touch-action: pan-y;
}
.sh-mesh-surface:active { cursor: grabbing; }
.sh-capability-node {
  position: absolute;
  width: 52px;
  height: 52px;
  transform: translate(-50%, -50%);
  border: 0;
  border-radius: 50%;
  padding: 0;
  background: transparent;
  display: grid;
  place-items: center;
}
.sh-capability-node > span {
  width: 46px;
  height: 46px;
  border: 1px solid #d6efaa;
  border-radius: 50%;
  box-shadow: 0 0 28px #d3f79e26, inset 0 0 15px #cdeb961a;
  opacity: 0;
  transform: scale(.7);
  transition: opacity .2s, transform .3s;
}
.sh-capability-node.is-active > span { opacity: .8; transform: scale(1); }
.sh-node-tooltip {
  position: absolute;
  z-index: 100;
  transform: translate(-50%, calc(-100% - 34px));
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border: 1px solid #dcf0c629;
  border-radius: 16px;
  color: #eef5e7;
  background: #15241de8;
  box-shadow: 0 16px 48px #0003;
  backdrop-filter: blur(18px);
  pointer-events: none;
  white-space: nowrap;
}
.sh-node-tooltip > svg { color: #d4edaa; flex-shrink: 0; }
.sh-node-tooltip.is-below { transform: translate(-50%, 34px); }
.sh-node-tooltip strong { font-size: 15px; font-weight: 550; }
.sh-node-tooltip span { display: block; margin-top: 4px; font-size: 12px; color: #b4c4b5; }
.node-tip-enter-active, .node-tip-leave-active { transition: opacity .18s; }
.node-tip-enter-from, .node-tip-leave-to { opacity: 0; }
@media (max-width: 760px) {
  .sh-mesh-stage { height: 410px; }
  .sh-node-tooltip { padding: 12px 15px; }
}
@media (prefers-reduced-motion: reduce) {
  .sh-core-aura, .sh-capability-node > span, .node-tip-enter-active, .node-tip-leave-active { transition: none; }
}
</style>
