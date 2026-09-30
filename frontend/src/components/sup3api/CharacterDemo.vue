<template>
  <figure ref="stage" class="character-demo" :aria-label="name + '交互演示'">
    <div class="character-modes" role="group" aria-label="模型显示方式">
      <button
        v-for="item in visibleModes"
        :key="item.id"
        :aria-pressed="mode === item.id"
        :disabled="!ready"
        :aria-label="item.label"
        :title="item.id === 'skeleton' ? `骨骼 · ${joints} 个关节` : item.label"
        @click="selectMode(item.id)"
      >
        <svg
          v-if="item.id === 'skeleton'"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          aria-hidden="true"
        >
          <circle cx="12" cy="4" r="2" />
          <path
            d="M12 6v8m-7-5 7-2 7 2M5 9l-2 5m16-5 2 5m-9 0-4 7m4-7 4 7"
            stroke-linecap="round"
          />
          <circle cx="12" cy="14" r="1.5" />
        </svg>
        <Icon
          v-else
          :name="item.id === 'model' ? 'cube' : 'grid'"
          size="lg"
          aria-hidden="true"
        />
      </button>
    </div>
    <div class="character-viewport">
      <canvas
        v-show="ready && !error"
        ref="canvas"
        tabindex="0"
        role="img"
        :aria-label="name + '，可拖动、缩放，左右方向键旋转'"
        @keydown.left.prevent="viewer?.rotate(-0.15)"
        @keydown.right.prevent="viewer?.rotate(0.15)"
        @webglcontextlost.prevent="fail"
      />
      <p v-if="!ready || error" class="character-status" role="status">
        {{ error || '正在载入…' }}
      </p>
    </div>
    <div v-if="ready && mode === 'materials'" class="character-materials">
      <div class="character-material-select" role="group" aria-label="材质选择">
        <button
          v-for="(material, index) in materials"
          :key="index"
          :aria-pressed="selected === index"
          @click="selectMaterial(index)"
        >
          {{ material.meshes.join('、') }} → {{ material.name }}
        </button>
      </div>
      <div class="character-textures">
        <button
          v-for="texture in materials[selected]?.textures"
          :key="texture.label"
          :aria-label="`查看${texture.label}贴图`"
          @click="openTexture(texture)"
        >
          <img
            :src="texture.image"
            :alt="`${texture.label}贴图`"
            width="88"
            height="88"
          />
          <span>{{ texture.label }}</span>
        </button>
        <span v-if="!materials[selected]?.textures.length"
          >此材质使用纯色参数</span
        >
      </div>
      <p
        v-if="!materials[selected]?.textures.some(t => t.label === '粗糙度' || t.label === '金属度')"
        class="character-parameters"
      >
        粗糙度 {{ materials[selected]?.roughness.toFixed(2) }} · 金属度
        {{ materials[selected]?.metalness.toFixed(2) }}
      </p>
    </div>
    <dialog
      ref="textureDialog"
      class="texture-dialog"
      :aria-label="`${expandedTexture?.label}贴图`"
      @click="closeBackdrop"
    >
      <header>
        <span>{{ expandedTexture?.label }}</span>
        <button aria-label="关闭贴图" @click="textureDialog?.close()">
          <Icon name="x" size="lg" aria-hidden="true" />
        </button>
      </header>
      <img
        v-if="expandedTexture"
        :src="expandedTexture.image"
        :alt="`${expandedTexture.label}贴图`"
        width="512"
        height="512"
      />
    </dialog>
    <figcaption class="character-credit">
      <span
        class="character-provider"
        aria-label="Tripo H3.1"
        title="Tripo H3.1"
        ><PlatformIcon platform="tripo" size="lg" />H3.1</span
      >
      <a
        :href="modelUrl"
        download
        :aria-label="`下载${name} GLB`"
        title="下载 GLB"
        ><Icon name="download" size="lg" aria-hidden="true"
      /></a>
    </figcaption>
  </figure>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type {
  createCharacterPreview,
  MaterialInfo,
  PreviewMode,
} from '@/utils/characterPreview'
const props = defineProps<{
  modelUrl: string
  rotation: number
  name: string
}>()
const modes: { id: PreviewMode; label: string }[] = [
  { id: 'model', label: '模型' },
  { id: 'skeleton', label: '骨骼' },
  { id: 'materials', label: '材质' },
]
const canvas = ref<HTMLCanvasElement>(),
  stage = ref<HTMLElement>()
const mode = ref<PreviewMode>('model'),
  selected = ref(0),
  ready = ref(false),
  error = ref(''),
  joints = ref(0)
const materials = ref<MaterialInfo[]>([])
const textureDialog = ref<HTMLDialogElement>()
const expandedTexture = ref<MaterialInfo['textures'][number]>()
async function openTexture(texture: MaterialInfo['textures'][number]) {
  expandedTexture.value = texture
  await nextTick()
  textureDialog.value?.showModal()
}
function closeBackdrop(event: MouseEvent) {
  if (event.target !== textureDialog.value) return
  const bounds = textureDialog.value!.getBoundingClientRect()
  if (
    event.clientX < bounds.left || event.clientX > bounds.right ||
    event.clientY < bounds.top || event.clientY > bounds.bottom
  ) textureDialog.value?.close()
}
const visibleModes = computed(() =>
  modes.filter((item) => item.id !== 'skeleton' || joints.value > 0),
)
let viewer: ReturnType<typeof createCharacterPreview> | undefined
let resize: ResizeObserver | undefined,
  intersection: IntersectionObserver | undefined
let disposed = false,
  visible = false,
  started = false
function syncVisibility() {
  viewer?.setActive(visible && !document.hidden)
}
function selectMode(value: PreviewMode) {
  mode.value = value
  viewer?.setMode(value, selected.value)
}
function selectMaterial(index: number) {
  selected.value = index
  viewer?.setMode(mode.value, index)
}
function fail() {
  error.value = '预览暂不可用，可下载 GLB 查看。'
  ready.value = false
  viewer?.dispose()
  viewer = undefined
}
async function startViewer() {
  if (started) return
  started = true
  try {
    const { createCharacterPreview } = await import('@/utils/characterPreview')
    if (disposed || !canvas.value) return
    viewer = createCharacterPreview(canvas.value)
    syncVisibility()
    resize = new ResizeObserver(() => viewer?.resize())
    resize.observe(canvas.value)
    const info = await viewer.load(props.modelUrl, props.rotation)
    if (disposed || !info) return
    materials.value = info.materials
    joints.value = info.joints
    ready.value = true
  } catch {
    if (!disposed) fail()
  }
}
onMounted(() => {
  document.addEventListener('visibilitychange', syncVisibility)
  intersection = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting
    if (visible) void startViewer()
    syncVisibility()
  })
  intersection.observe(stage.value!)
})
onBeforeUnmount(() => {
  disposed = true
  resize?.disconnect()
  intersection?.disconnect()
  document.removeEventListener('visibilitychange', syncVisibility)
  viewer?.dispose()
})
</script>
<style scoped>
.character-demo {
  width: 100%;
  margin: 0;
  padding: 24px;
  color: #d9e6dd;
  background: radial-gradient(ellipse at 50% 35%, #28392e 0%, #1b2921 48%, #131e18 100%);
}
.character-modes {
  display: flex;
  gap: 4px;
  width: fit-content;
  padding: 4px;
  border: 1px solid #ffffff20;
  border-radius: 24px;
  background: #111a16cc;
  margin-inline: auto;
}
.character-modes button {
  min-height: 44px;
  width: 44px;
  display: grid;
  place-items: center;
  padding: 10px;
  border-radius: 20px;
  font-size: 13px;
}
button[aria-pressed='true'] {
  background: #365340;
  color: #fff;
}
button:disabled {
  opacity: 0.5;
  cursor: default;
}
button:focus-visible,
canvas:focus-visible,
a:focus-visible {
  outline: 2px solid #98c5a5;
  outline-offset: 3px;
}
.character-viewport {
  position: relative;
  height: 400px;
}
canvas {
  width: 100%;
  height: 100%;
  display: block;
  cursor: grab;
  touch-action: pan-y;
}
canvas:active {
  cursor: grabbing;
}
.character-status {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 14px;
}
.character-credit {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
}
.character-provider {
  display: flex;
  align-items: center;
  gap: 8px;
}
.character-credit a {
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border: 1px solid #ffffff20;
  border-radius: 50%;
  background: #ffffff08;
  transition: background 0.2s;
}
.character-credit a:hover {
  background: #263b2f;
}
.character-modes svg {
  width: 22px;
  height: 22px;
}
.character-materials {
  padding: 16px;
  margin-bottom: 10px;
  border: 1px solid #ffffff20;
  border-radius: 14px;
  background: #111a16dd;
}
.character-material-select {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.character-material-select button {
  padding: 8px 12px;
  min-height: 40px;
  border-radius: 8px;
  font-size: 12px;
}
.character-textures {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  padding-top: 12px;
}
.character-textures button {
  min-width: 0;
  padding: 6px;
  border: 1px solid #ffffff20;
  border-radius: 10px;
  background: #263b2f;
  text-align: center;
  font-size: 12px;
  transition: border-color .2s, transform .2s;
}
.character-textures button:hover {
  border-color: #98c5a5;
  transform: translateY(-2px);
}
.character-textures img {
  border-radius: 6px;
  margin-bottom: 8px;
  width: 100%;
  height: auto;
  aspect-ratio: 1;
}
.texture-dialog {
  padding: 20px;
  width: min(552px, calc(100vw - 32px));
  max-height: calc(100dvh - 32px);
  overflow: auto;
  margin: auto;
  border: 1px solid #ffffff20;
  border-radius: 20px;
  background: #16221b;
  color: #d9e6dd;
  box-shadow: 0 30px 100px #08180f40;
}
.texture-dialog::backdrop { background: #09191099; backdrop-filter: blur(8px); }
.texture-dialog header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.texture-dialog header button { display: grid; place-items: center; width: 44px; height: 44px; border-radius: 50%; background: #293b30; }
.texture-dialog > img { width: 100%; height: auto; border-radius: 10px; }
@media (prefers-reduced-motion: reduce) {
  .character-textures button { transition: none; }
}
.character-parameters {
  margin: 10px 0 0;
  font-size: 12px;
  color: #aabeb0;
}
@media (max-width: 480px) {
  .character-demo {
    padding: 16px;
  }
  .character-viewport {
    height: 330px;
  }
  .character-credit {
    gap: 8px;
  }
}
</style>
