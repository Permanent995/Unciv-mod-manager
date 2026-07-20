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

const currentView = ref('mods')
const zoom = ref(100)
const sidebarPos = ref('left')
const sidebarWidth = ref(220)
const hiddenNav = ref<string[]>([])
const theme = ref('light')

onMounted(loadConfig)

async function loadConfig() {
  const cfg = await GetAppConfig()
  zoom.value = cfg.zoomLevel || 100
  sidebarPos.value = cfg.sidebarPos || 'left'
  sidebarWidth.value = cfg.sidebarWidth || 200
  hiddenNav.value = cfg.hiddenNav || []
  theme.value = cfg.theme || 'light'
  applyTheme()
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
</script>

<template>
  <div class="app-root" :class="'pos-' + (sidebarPos === 'right' ? 'right' : 'left')">
    <!-- Custom title bar (draggable) -->
    <div class="title-bar">
      <span class="title-text">Unciv Mod Manager</span>
      <div class="title-actions">
        <button class="tb-btn theme-btn" @click="toggleTheme" :title="theme === 'dark' ? '切换亮色' : '切换暗色'">
          {{ theme === 'dark' ? '☀' : '🌙' }}
        </button>
        <button class="tb-btn" @click="WindowMinimise" title="最小化">—</button>
        <button class="tb-btn" @click="WindowToggleMaximise" title="最大化">□</button>
        <button class="tb-btn tb-close" @click="Quit" title="关闭">×</button>
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
  --accent: #4f46e5;
  --accent-hover: #6366f1;
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;

  --bg-primary: #ffffff;
  --bg-secondary: #f8fafc;
  --bg-sidebar: #f1f5f9;
  --bg-card: #f1f5f9;
  --bg-input: #ffffff;
  --bg-hover: #e2e8f0;
  --bg-active: #cbd5e1;
  --sidebar-active: #e2e8f0;

  --text-primary: #1e293b;
  --text-secondary: #475569;
  --text-muted: #94a3b8;

  --border-color: rgba(203,213,225,0.5);
  --card-shadow: 0 1px 2px 0 rgba(0,0,0,0.05);
}

[data-theme="dark"] {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-sidebar: #1e293b;
  --bg-card: #1e293b;
  --bg-input: #1e293b;
  --bg-hover: #334155;
  --bg-active: #475569;
  --sidebar-active: #334155;

  --border-color: rgba(51,65,85,0.5);
  --card-shadow: none;
}

* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: 'Microsoft YaHei', '微软雅黑', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 15px;
  background: var(--bg-primary);
  color: var(--text-primary);
  /* Only title bar is draggable; everything else is no-drag by default */
  --wails-draggable: no-drag;
}
/* Only the title bar region allows window dragging */
.title-bar {
  --wails-draggable: drag;
  -webkit-app-region: drag;
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
  -webkit-app-region: drag;
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
  -webkit-app-region: no-drag;
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
.main-content { flex: 1; overflow: hidden; padding: 20px; background: var(--bg-primary); --wails-draggable: no-drag; min-height: 0; }
</style>
