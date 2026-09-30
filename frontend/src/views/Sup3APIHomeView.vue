<template>
  <div class="sh-home" @keydown.esc="closeMenu(true)">
    <a
      href="#home-content"
      class="sh-skip"
      @click.prevent="jumpTo('home-content')"
      >跳至主要内容</a
    >
    <header class="sh-header">
      <div class="sh-nav-wrap">
        <RouterLink to="/home" class="sh-brand" aria-label="Sup3API 首页"
          ><img
            src="/sup3api-mark.svg"
            width="30"
            height="30"
            alt=""
          />Sup3API</RouterLink
        >
        <nav class="sh-desktop-nav" aria-label="主导航">
          <a href="#capabilities" @click.prevent="jumpTo('capabilities')"
            >探索能力</a
          >
          <a href="#developers" @click.prevent="jumpTo('developers')">开发者</a>
          <RouterLink to="/docs">API 文档</RouterLink>
        </nav>
        <div class="sh-nav-actions">
          <RouterLink
            :to="auth.isAuthenticated ? '/dashboard' : '/login'"
            class="sh-console"
            >控制台 <span aria-hidden="true">↗</span></RouterLink
          >
          <button
            ref="menuButton"
            class="sh-menu-button"
            :aria-expanded="menuOpen"
            aria-controls="home-mobile-menu"
            :aria-label="menuOpen ? '关闭导航菜单' : '打开导航菜单'"
            @click="menuOpen = !menuOpen"
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              aria-hidden="true"
            >
              <path
                :d="menuOpen ? 'M6 6l12 12M6 18 18 6' : 'M4 8h16M4 16h16'"
              />
            </svg>
          </button>
        </div>
      </div>
      <nav
        v-if="menuOpen"
        id="home-mobile-menu"
        class="sh-mobile-nav"
        aria-label="移动导航"
      >
        <a href="#capabilities" @click.prevent="jumpTo('capabilities')"
          >探索能力 <span aria-hidden="true">↗</span></a
        >
        <a href="#developers" @click.prevent="jumpTo('developers')"
          >开发者 <span aria-hidden="true">↗</span></a
        >
        <RouterLink to="/docs" @click="closeMenu()"
          >API 文档 <span aria-hidden="true">↗</span></RouterLink
        >
      </nav>
    </header>

    <main id="home-content" tabindex="-1">
      <section class="sh-hero" aria-labelledby="home-title">
        <div class="sh-hero-grid sh-container">
          <div class="sh-hero-copy">
            <h1 id="home-title">让想象，<br /><span>多一个维度。</span></h1>
            <p class="sh-hero-description">
              从文字、图像，到三维世界。<br />一个入口，连接你的应用与生成模型。
            </p>
            <div class="sh-actions">
              <RouterLink to="/connect" class="sh-button sh-button-mint"
                >开始构建 <span aria-hidden="true">↗</span></RouterLink
              ><a
                href="#capabilities"
                class="sh-text-link"
                @click.prevent="jumpTo('capabilities')"
                >探索可能 <span aria-hidden="true">↓</span></a
              >
            </div>
          </div>
          <LandingMesh />
        </div>
        <div class="sh-providers sh-container" aria-label="支持的模型平台">
          <div
            v-for="provider in providers"
            :key="provider.id"
            class="sh-provider"
          >
            <PlatformIcon :platform="provider.id" size="lg" /><span>{{
              provider.name
            }}</span>
          </div>
        </div>
      </section>

      <section
        id="capabilities"
        class="sh-capabilities"
        aria-labelledby="capabilities-title"
        tabindex="-1"
      >
        <div class="sh-container">
          <div class="sh-section-heading">
            <h2 id="capabilities-title">
              想得到。<br /><span>也能创造得到。</span>
            </h2>
            <p>让不同模型各展所长，让你的应用少一些接入工作。</p>
          </div>
          <div class="sh-demo-toolbar">
            <div
              class="sh-capability-tabs"
              role="tablist"
              aria-label="生成能力"
            >
              <button
                v-for="(item, index) in capabilities"
                :id="`capability-tab-${item.id}`"
                :key="item.id"
                role="tab"
                :aria-selected="capabilityIndex === index"
                :tabindex="capabilityIndex === index ? 0 : -1"
                :aria-controls="`capability-panel-${item.id}`"
                @click="capabilityIndex = index"
                @keydown="
                  selectTab($event, index, capabilities.length, 'capability')
                "
              >
                {{ item.label }}
              </button>
            </div>
            <div class="sh-demo-picker" role="group" aria-label="演示资产">
              <button
                v-for="(asset, index) in demos"
                :key="asset.id"
                :aria-pressed="demoIndex === index"
                @click="demoIndex = index"
              >
                <img :src="asset.image" width="36" height="36" alt="" />{{
                  asset.label
                }}
              </button>
            </div>
          </div>
          <div
            v-for="(item, index) in capabilities"
            v-show="capabilityIndex === index"
            :id="`capability-panel-${item.id}`"
            :key="item.id"
            class="sh-capability-panel"
            :class="`sh-capability-${item.id}`"
            role="tabpanel"
            :aria-labelledby="`capability-tab-${item.id}`"
            tabindex="0"
          >
            <div class="sh-capability-copy">
              <h3>{{ item.title }}</h3>
              <p>{{ item.description }}</p>
              <ul>
                <li v-for="feature in item.features" :key="feature">
                  <span aria-hidden="true">✓</span>{{ feature }}
                </li>
              </ul>
              <RouterLink :to="item.link" class="sh-text-link"
                >{{ item.linkText }}
                <span aria-hidden="true">↗</span></RouterLink
              >
            </div>
            <div class="sh-capability-art">
              <div v-if="item.id === 'text'" class="sh-prompt-demo">
                <h4>{{ demo.name }}</h4>
                <p>{{ demo.summary }}</p>
                <details class="sh-full-prompt">
                  <summary>完整提示词</summary>
                  <p>{{ demo.prompt }}</p>
                </details>
                <div
                  class="sh-model-flow"
                  aria-label="生成流程：Nano Banana 生成原画，再由 Tripo H3.1 图生三维"
                >
                  <span
                    ><PlatformIcon platform="gemini" size="md" />Nano
                    Banana</span
                  >
                  <Icon name="arrowRight" size="sm" aria-hidden="true" />
                  <span><PlatformIcon platform="tripo" size="md" />H3.1</span>
                </div>
              </div>
              <figure v-else-if="item.id === 'image'" class="sh-image-demo">
                <img
                  :src="demo.image"
                  width="864"
                  height="1184"
                  loading="lazy"
                  :alt="demo.name + '生成原画'"
                />
                <figcaption>
                  <PlatformIcon platform="gemini" size="md" />Nano Banana
                </figcaption>
              </figure>
              <CharacterDemo
                v-else-if="capabilityIndex === index"
                :key="demo.id"
                :model-url="demo.model"
                :rotation="demo.rotation"
                :name="demo.name"
              />
            </div>
          </div>
        </div>
      </section>

      <section
        id="developers"
        class="sh-developers sh-container"
        aria-labelledby="developers-title"
        tabindex="-1"
      >
        <div class="sh-developer-intro">
          <h2 id="developers-title">
            你的工作流。<br /><span>依然熟悉。</span>
          </h2>
          <p>
            统一 API，让新应用轻装出发。<br />原生接口，让已有工作流从容接入。
          </p>
          <RouterLink to="/docs/compatibility" class="sh-text-link"
            >查看兼容范围 <span aria-hidden="true">↗</span></RouterLink
          >
        </div>
        <div class="sh-code-window">
          <div class="sh-code-tabs" role="tablist" aria-label="接入方式">
            <button
              v-for="(example, index) in examples"
              :id="`example-tab-${example.id}`"
              :key="example.id"
              role="tab"
              :aria-selected="exampleIndex === index"
              :tabindex="exampleIndex === index ? 0 : -1"
              :aria-controls="`example-panel-${example.id}`"
              @click="selectExample(index)"
              @keydown="selectTab($event, index, examples.length, 'example')"
            >
              {{ example.label }}
            </button>
          </div>
          <div
            v-for="(example, index) in examples"
            v-show="exampleIndex === index"
            :id="`example-panel-${example.id}`"
            :key="example.id"
            role="tabpanel"
            :aria-labelledby="`example-tab-${example.id}`"
            tabindex="0"
            class="sh-code-panel"
          >
            <div class="sh-code-endpoint">
              <span>POST</span>{{ example.endpoint }}
            </div>
            <pre
              tabindex="0"
              :aria-label="`${example.label}请求代码`"
            ><code>{{ example.code }}</code></pre>
          </div>
          <div class="sh-code-footer">
            <span role="status">{{ copyMessage }}</span
            ><button class="sh-copy" @click="copyExample">
              {{ copied ? '已复制 ✓' : '复制代码' }}
            </button>
          </div>
        </div>
      </section>

      <section class="sh-start-section" aria-labelledby="start-title">
        <div class="sh-container">
          <div class="sh-section-heading">
            <h2 id="start-title">下一步，<span>开始创造。</span></h2>
          </div>
          <div class="sh-steps">
            <RouterLink
              v-for="step in steps"
              :key="step.number"
              :to="step.to"
              class="sh-step"
              ><span class="sh-step-number">{{ step.number }}</span>
              <h3>{{ step.title }}<span aria-hidden="true">↗</span></h3>
              <p>{{ step.description }}</p></RouterLink
            >
          </div>
        </div>
      </section>
      <section class="sh-faq sh-container" aria-labelledby="faq-title">
        <h2 id="faq-title">你可能还想了解。</h2>
        <div class="sh-faq-list">
          <details v-for="faq in faqs" :key="faq.question">
            <summary>
              {{ faq.question }}<span aria-hidden="true">+</span>
            </summary>
            <div>
              <p>{{ faq.answer }}</p>
              <RouterLink :to="faq.to" class="sh-text-link"
                >{{ faq.link }} <span aria-hidden="true">↗</span></RouterLink
              >
            </div>
          </details>
        </div>
      </section>
    </main>
    <footer class="sh-footer sh-container">
      <div>
        <RouterLink to="/home" class="sh-brand"
          ><img
            src="/sup3api-mark.svg"
            width="26"
            height="26"
            alt=""
          />Sup3API</RouterLink
        >
      </div>
      <nav aria-label="页脚导航">
        <RouterLink to="/docs">API 文档</RouterLink
        ><RouterLink to="/docs/sources">技术与开源</RouterLink
        ><a
          href="https://github.com/lanson-dev/Sup3API"
          target="_blank"
          rel="noopener noreferrer"
          >GitHub <span class="sh-sr-only">（新窗口）</span
          ><span aria-hidden="true">↗</span></a
        >
      </nav>
      <span class="sh-copyright"
        >© {{ new Date().getFullYear() }} Sup3API</span
      >
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import LandingMesh from '@/components/sup3api/LandingMesh.vue'
import CharacterDemo from '@/components/sup3api/CharacterDemo.vue'
import {
  landingDemos as demos,
  landingCapabilities as capabilities,
  landingExamples as examples,
  landingFaqs as faqs,
  landingSteps as steps,
} from '@/content/sup3api-landing'
import '@/styles/sup3api-landing.css'

const auth = useAuthStore()
const providers = [
  { id: 'openai', name: 'OpenAI' },
  { id: 'anthropic', name: 'Claude' },
  { id: 'tripo', name: 'Tripo' },
  { id: 'meshy', name: 'Meshy' },
] as const
const menuOpen = ref(false)
const menuButton = ref<HTMLButtonElement>()
const capabilityIndex = ref(0)
const demoIndex = ref(0)
const demo = computed(() => demos[demoIndex.value])
const exampleIndex = ref(0)
const copied = ref(false)
const copyMessage = ref('')
let copyTimer: ReturnType<typeof setTimeout> | undefined
let copyRevision = 0
let desktopQuery: MediaQueryList | undefined

function closeMenu(restoreFocus = false) {
  if (!menuOpen.value) return
  menuOpen.value = false
  if (restoreFocus) menuButton.value?.focus()
}
function jumpTo(id: string) {
  closeMenu()
  const target = document.getElementById(id)
  target?.focus({ preventScroll: true })
  target?.scrollIntoView({
    behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches
      ? 'instant'
      : 'smooth',
    block: 'start',
  })
}
function selectExample(index: number) {
  exampleIndex.value = index
  copyRevision++
  clearTimeout(copyTimer)
  copied.value = false
  copyMessage.value = ''
}
async function selectTab(
  event: KeyboardEvent,
  index: number,
  count: number,
  kind: 'capability' | 'example',
) {
  if (event.ctrlKey || event.metaKey || event.altKey) return
  let target: number
  if (event.key === 'ArrowRight') target = (index + 1) % count
  else if (event.key === 'ArrowLeft') target = (index - 1 + count) % count
  else if (event.key === 'Home') target = 0
  else if (event.key === 'End') target = count - 1
  else return
  event.preventDefault()
  if (kind === 'capability') capabilityIndex.value = target
  else selectExample(target)
  await nextTick()
  const item = kind === 'capability' ? capabilities[target] : examples[target]
  document.getElementById(`${kind}-tab-${item.id}`)?.focus()
}
async function copyExample() {
  const revision = ++copyRevision
  const example = examples[exampleIndex.value]
  try {
    await navigator.clipboard.writeText(example.code)
    if (revision !== copyRevision) return
    copied.value = true
    copyMessage.value = `${example.label}代码已复制`
    clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied.value = false
      copyMessage.value = ''
    }, 2200)
  } catch {
    if (revision !== copyRevision) return
    copied.value = false
    copyMessage.value = '未能复制，请选中上方代码手动复制。'
  }
}
function onDesktopChange(event: MediaQueryListEvent) {
  if (event.matches) closeMenu()
}
onMounted(() => {
  desktopQuery = window.matchMedia('(min-width: 761px)')
  desktopQuery.addEventListener('change', onDesktopChange)
})
onBeforeUnmount(() => {
  copyRevision++
  clearTimeout(copyTimer)
  desktopQuery?.removeEventListener('change', onDesktopChange)
})
</script>
