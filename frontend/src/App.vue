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
:root,
[data-theme="light"] {
  --primary: #4f46e5;
  --primary-light: #6366f1;
  --primary-dark: #3730a3;
  --accent: #06b6d4;
  --accent-light: #22d3ee;
  --accent-dark: #0891b2;
  --success: #10b981;
  --success-light: #34d399;
  --success-dark: #059669;
  --warning: #f59e0b;
  --warning-light: #fbbf24;
  --danger: #ef4444;
  --danger-light: #f87171;
  --danger-dark: #dc2626;

  --gray-50: #f9fafb;
  --gray-100: #f3f4f6;
  --gray-200: #e5e7eb;
  --gray-300: #d1d5db;
  --gray-400: #9ca3af;
  --gray-500: #6b7280;
  --gray-600: #4b5563;
  --gray-700: #374151;
  --gray-800: #1f2937;
  --gray-900: #111827;
  --gray-950: #030712;

  --bg-app: #ffffff;
  --bg-surface: #f8fafc;
  --bg-card: #f1f5f9;
  --bg-hover: #e2e8f0;
  --bg-active: #cbd5e1;
  --bg-overlay: rgba(255,255,255,0.95);

  --text-primary: #1e293b;
  --text-secondary: #475569;
  --text-tertiary: #64748b;
  --text-muted: #94a3b8;
  --text-inverse: #ffffff;

  --border-default: rgba(203,213,225,0.3);
  --border-light: rgba(203,213,225,0.5);
  --border-strong: rgba(148,163,184,0.8);

  --shadow-xs: 0 1px 2px 0 rgba(0,0,0,0.05);
  --shadow-sm: 0 1px 3px 0 rgba(0,0,0,0.1), 0 1px 2px -1px rgba(0,0,0,0.06);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -2px rgba(0,0,0,0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -4px rgba(0,0,0,0.05);
  --shadow-xl: 0 20px 25px -5px rgba(0,0,0,0.1), 0 10px 10px -5px rgba(0,0,0,0.04);

  /* UMM compat aliases */
  --bg-primary: #ffffff;
  --bg-secondary: #f8fafc;
  --bg-sidebar: #f1f5f9;
  --bg-input: #ffffff;
  --bg-card: #f1f5f9;
  --sidebar-active: #e2e8f0;
  --accent: #4f46e5;
  --accent-hover: #6366f1;
  --border-color: rgba(203,213,225,0.5);
  --card-shadow: 0 1px 2px 0 rgba(0,0,0,0.05);
}

[data-theme="dark"] {
  --bg-app: #0f172a;
  --bg-surface: #1e293b;
  --bg-card: #1e293b;
  --bg-hover: #334155;
  --bg-active: #475569;
  --bg-overlay: rgba(15,23,42,0.95);
  --border-default: rgba(51,65,85,0.3);
  --border-light: rgba(51,65,85,0.5);
  --border-strong: rgba(100,116,139,0.8);
  --shadow-xs: none;
  --shadow-sm: none;
  --shadow-md: none;
  --shadow-lg: none;
  --shadow-xl: none;
  --text-inverse: #1e293b;

  /* UMM compat aliases */
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-sidebar: #1e293b;
  --bg-input: #1e293b;
  --sidebar-active: #334155;
  --accent: #4f46e5;
  --accent-hover: #6366f1;
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
