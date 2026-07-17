<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { LaunchUnciv, IsUncivRunning, GetUncivVersion } from '../../wailsjs/go/app/App'

const props = defineProps<{
  currentView: string
  hiddenNav: string[]
  sidebarWidth: number
}>()

const emit = defineEmits<{
  (e: 'update:view', view: string): void
  (e: 'resize', width: number): void
}>()

const menuItems = [
  { id: 'mods', label: '模组库', icon: '📦' },
  { id: 'maps', label: '地图', icon: '🗺️' },
  { id: 'saves', label: '存档', icon: '💾' },
  { id: 'multiplayer', label: '联机检查', icon: '🔗' },
  { id: 'downloads', label: '下载', icon: '⬇️' },
  { id: 'browse', label: '模组发现', icon: '🔍' },
  { id: 'toolbox', label: '工具箱', icon: '🧰' },
  { id: 'help', label: '帮助', icon: '📖' },
  { id: 'settings', label: '设置', icon: '⚙️' },
  { id: 'about', label: '关于', icon: 'ℹ️' },
]

const launching = ref(false)
const dragging = ref(false)
const gameVersion = ref('')

onMounted(async () => {
  try {
    gameVersion.value = await GetUncivVersion()
  } catch {
    gameVersion.value = ''
  }
})

function selectView(id: string) {
  emit('update:view', id)
}

async function launchUnciv() {
  launching.value = true
  try {
    if (await IsUncivRunning()) {
      alert('Unciv 已在运行中')
      return
    }
    await LaunchUnciv()
  } catch (e: any) {
    alert('启动失败: ' + e)
  } finally {
    launching.value = false
  }
}

function startDrag(e: MouseEvent) {
  dragging.value = true
  const startX = e.clientX
  const startW = props.sidebarWidth

  function onMove(ev: MouseEvent) {
    const diff = ev.clientX - startX
    const newW = Math.max(120, Math.min(400, startW + diff))
    emit('resize', newW)
  }

  function onUp() {
    dragging.value = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}
</script>

<template>
  <div class="sidebar" :style="{ width: sidebarWidth + 'px' }">
    <div class="sidebar-header"><h2>Unciv Mod Manager</h2></div>
    <nav class="sidebar-nav">
      <div v-for="item in menuItems" :key="item.id"
        v-show="!(hiddenNav || []).includes(item.id)"
        class="nav-item" :class="{ active: currentView === item.id }"
        @click="selectView(item.id)">
        <span class="nav-icon">{{ item.icon }}</span>
        <span class="nav-label">{{ item.label }}</span>
      </div>
    </nav>
    <div class="sidebar-footer">
      <button class="launch-btn" :disabled="launching" @click="launchUnciv">
        {{ launching ? '启动中...' : '▶ 启动 Unciv' }}
      </button>
      <div v-if="gameVersion" class="game-version">🕹️ Unciv {{ gameVersion }}</div>
    </div>
    <div class="drag-handle" @mousedown="startDrag" :class="{ active: dragging }"></div>
  </div>
</template>

<style scoped>
.sidebar { position: relative; background: var(--bg-sidebar); color: var(--text-primary); display: flex; flex-direction: column; height: 100%; flex-shrink: 0; overflow: hidden; --wails-draggable: no-drag; }
.sidebar-header { padding: 20px; border-bottom: 1px solid var(--border-color); }
.sidebar-header h2 { margin: 0; font-size: 18px; font-weight: 600; }
.sidebar-nav { flex: 1; padding: 10px 0; overflow-y: auto; }
.nav-item { display: flex; align-items: center; padding: 12px 20px; cursor: pointer; transition: background 0.2s; }
.nav-item:hover { background: var(--border-color); }
.nav-item.active { background: var(--sidebar-active); border-left: 3px solid var(--accent); }
.nav-icon { font-size: 20px; margin-right: 12px; }
.nav-label { font-size: 14px; }
.sidebar-footer { padding: 20px; border-top: 1px solid var(--border-color); }
.launch-btn { width: 100%; padding: 10px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
.launch-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.launch-btn:hover:not(:disabled) { background: var(--accent-hover); }
.game-version { text-align: center; font-size: 12px; color: var(--text-muted); margin-top: 8px; white-space: nowrap; }
.drag-handle { position: absolute; top: 0; right: 0; width: 4px; height: 100%; cursor: col-resize; background: transparent; z-index: 10; }
.drag-handle:hover, .drag-handle.active { background: var(--accent); }
</style>
