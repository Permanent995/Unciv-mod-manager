<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { GetAppConfig, SaveAppConfig } from '../wailsjs/go/app/App'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../wailsjs/runtime/runtime'
import Sidebar from './components/Sidebar.vue'
import ModsView from './components/ModsView.vue'
import MapsView from './components/MapsView.vue'
import DownloadsView from './components/DownloadsView.vue'
import BrowseView from './components/BrowseView.vue'
import MultiplayerView from './components/MultiplayerView.vue'
import ToolboxView from './components/ToolboxView.vue'
import SavesView from './components/SavesView.vue'
import HelpView from './components/HelpView.vue'
import SettingsView from './components/SettingsView.vue'
import AboutView from './components/AboutView.vue'
import IconMoon from './components/icons/IconMoon.vue'
import IconSun from './components/icons/IconSun.vue'

const currentView = ref('mods')
const zoom = ref(100)
const sidebarPos = ref('left')
const sidebarWidth = ref(220)
const hiddenNav = ref<string[]>([])
const theme = ref('light')
const themeVariant = ref('pure')

onMounted(loadConfig)

async function loadConfig() {
  const cfg = await GetAppConfig()
  zoom.value = cfg.zoomLevel || 100
  sidebarPos.value = cfg.sidebarPos || 'left'
  sidebarWidth.value = cfg.sidebarWidth || 200
  hiddenNav.value = cfg.hiddenNav || []
  theme.value = cfg.theme || 'light'
  themeVariant.value = cfg.themeVariant || 'pure'
  applyTheme()
  applyThemeVariant()
}

const zoomStyle = computed(() => ({ zoom: `${(zoom.value || 100) / 100}` }))

function updateView(view: string) {
  currentView.value = view
  if (view !== 'settings') loadConfig()
}

async function onResize(w: number) {
  sidebarWidth.value = w
  const cfg = await GetAppConfig()
  cfg.sidebarWidth = w
  await SaveAppConfig(cfg)
}

async function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  applyTheme()
  const cfg = await GetAppConfig()
  cfg.theme = theme.value
  await SaveAppConfig(cfg)
}

function applyTheme() {
  document.documentElement.setAttribute('data-theme', theme.value)
}

function applyThemeVariant() {
  document.documentElement.setAttribute('data-theme-variant', themeVariant.value)
}

async function setThemeVariant(v: string) {
  themeVariant.value = v
  applyThemeVariant()
  const cfg = await GetAppConfig()
  cfg.themeVariant = v
  await SaveAppConfig(cfg)
}
</script>

<template>
  <div class="app-root" :class="'pos-' + (sidebarPos === 'right' ? 'right' : 'left')">
    <!-- Custom title bar (draggable) -->
    <div class="title-bar">
      <span class="title-text">Unciv Mod Manager</span>
      <div class="title-actions">
        <button class="tb-btn theme-btn" @click="toggleTheme" :title="theme === 'dark' ? '切换亮色' : '切换暗色'" aria-label="切换主题">
          <IconSun v-if="theme === 'dark'" :size="16" />
          <IconMoon v-else :size="16" />
        </button>
        <button class="tb-btn" @click="WindowMinimise" title="最小化" aria-label="最小化">—</button>
        <button class="tb-btn" @click="WindowToggleMaximise" title="最大化" aria-label="最大化">□</button>
        <button class="tb-btn tb-close" @click="Quit" title="关闭" aria-label="关闭">×</button>
      </div>
    </div>

    <div class="app-body" :style="zoomStyle">
      <Sidebar :currentView="currentView" :hiddenNav="hiddenNav" :sidebarWidth="sidebarWidth" @update:view="updateView" @resize="onResize" />
      <main class="main-content">
        <ModsView v-if="currentView === 'mods'" />
        <MapsView v-else-if="currentView === 'maps'" />
        <SavesView v-else-if="currentView === 'saves'" />
        <MultiplayerView v-else-if="currentView === 'multiplayer'" />
        <DownloadsView v-else-if="currentView === 'downloads'" />
        <BrowseView v-else-if="currentView === 'browse'" />
        <ToolboxView v-else-if="currentView === 'toolbox'" />
        <HelpView v-else-if="currentView === 'help'" />
        <SettingsView v-else-if="currentView === 'settings'" />
        <AboutView v-else-if="currentView === 'about'" />
      </main>
    </div>
  </div>
</template>

<style>
/* ══ Theme variables ══ */
/* Only variables actually referenced by components are defined here.
   Previously there was a redundant LYT-style set + UMM compat aliases layer;
   the LYT set (`--primary-*`, `--gray-*`, `--shadow-*`, etc.) was unused and has been removed. */
:root,
[data-theme="light"] {
  /* ── 主题色 ── */
  --accent: #4f46e5;        /* 主色调：按钮、链接、焦点环 */
  --accent-hover: #6366f1;  /* 主色调悬停 */
  --success: #10b981;       /* 成功/绿色：toast、完成状态 */
  --warning: #f59e0b;       /* 警告/黄色：更新提示、排序星标 */
  --danger: #ef4444;         /* 危险/红色：删除按钮、错误提示 */

  /* ── 背景色 ── */
  --bg-primary: #ffffff;       /* 页面主背景 */
  --bg-secondary: #f8fafc46;   /* 次级背景：代码块、列表项交替 */
  --bg-sidebar: #c4d2e1;       /* 侧栏背景 */
  --bg-card: #97b8d9;          /* 卡片背景：.view-card 容器 */
  --bg-input: #ffffff;         /* 输入框背景 */
  --bg-hover: #96a4b7;         /* 可交互元素悬停背景 */
  --bg-active: #c1d2e7;        /* 激活/选中状态背景 */
  --sidebar-active: #acb8c8;   /* 侧栏导航项悬停背景 */

  /* ── 文字色 ── */
  --text-primary: #1e293b;    /* 主文字色：标题、正文 */
  --text-secondary: #475569;  /* 次级文字色：描述、标签、按钮 */
  --text-muted: #a4b1c7;      /* 弱化文字色：占位符、辅助信息 */

  /* ── 边框与阴影 ── */
  --border-color: rgba(203,213,225,0.5);  /* 通用边框、分隔线 */
  --card-shadow: 0 1px 2px 0 rgba(0,0,0,0.05);  /* 卡片阴影 */

  /* ── 层级 ── */
  --z-modal: 50;    /* 模态框叠层 */
  --z-drag: 10;     /* 拖拽手柄 */

  /* ── 代码块（跨主题固定深色） ── */
  --code-bg: #1a1a24;      /* 代码块背景 */
  --code-border: #333;      /* 代码块边框 */
  --code-text: #eedfdf;     /* 代码块文字 */
}

[data-theme="dark"] {
  /* ── 背景色（暗色） ── */
  --bg-primary: #c9cdd7;
  --bg-secondary: #a6b0c2;
  --bg-sidebar: #1e293b;
  --bg-card: #b2bccc;
  --bg-input: #1e293b;
  --bg-hover: #334155;
  --bg-active: #475569;
  --sidebar-active: #334155;

  /* ── 文字色（暗色） ── */
  --text-primary: #f1f5f9;
  --text-secondary: #cbd5e1;
  --text-muted: #64748b;

  /* ── 边框与阴影（暗色） ── */
  --border-color: rgba(71,85,105,0.6);
  --card-shadow: none;

  /* 层级（跨主题不变） */
  --z-modal: 50;
  --z-drag: 10;

  /* 代码块（跨主题固定，引用亮色组的值） */
}

/* ── 浅色主题变体 ── */
[data-theme="light"][data-theme-variant="warm"] {
  --bg-primary: #fefaf5;
  --bg-sidebar: #f5e6d3;
  --bg-card: #faebd7;
  --bg-input: #fffaf5;
}
[data-theme="light"][data-theme-variant="blue"] {
  --bg-primary: #f4f8fb;
  --bg-sidebar: #d6e5f3;
  --bg-card: #e4eef7;
  --bg-input: #f8fafc;
}
[data-theme="light"][data-theme-variant="green"] {
  --bg-primary: #f2f9f5;
  --bg-sidebar: #c8e6d4;
  --bg-card: #d9efe2;
  --bg-input: #f6fbf8;
}

* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: 'Microsoft YaHei', '微软雅黑', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 15px;
  background: var(--bg-primary);
  color: var(--text-primary);
}
/* ══ Title bar ══ */
.title-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 32px;
  background: var(--bg-sidebar);
  border-bottom: 1px solid var(--border-color);
  padding: 0 8px;
  user-select: none;
  --wails-draggable: drag;
}
.title-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  padding-left: 8px;
}
.title-actions {
  display: flex;
  gap: 2px;
  --wails-draggable: no-drag;
}
.tb-btn {
  width: 36px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tb-btn:hover { background: var(--border-color); }
.tb-close:hover { background: #e81123; color: #fff; }
.theme-btn { font-size: 13px; }

/* ══ Layout ══ */
.app-root { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
.app-root.pos-right { direction: rtl; }
.app-root.pos-right .main-content { direction: ltr; }
.app-body { display: flex; flex: 1; overflow: hidden; width: 100%; min-height: 0; }
.main-content { flex: 1; overflow: hidden; padding: 20px; background: var(--bg-primary); min-height: 0; }

/* ══ Focus ring (global) ══ */
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
:focus:not(:focus-visible) { outline: none; }

/* ══ Respect reduced motion ══ */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}

/* ══ View card container — wraps each page's content ══ */
.view-card {
  background: var(--bg-card);
  border-radius: 8px;
  border: 1px solid var(--border-color);
  padding: 16px;
  height: 100%;
  overflow-y: auto;
}
</style>
