<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ScanMaps, ImportFile, SaveClipboardAsMap } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'
import { OnFileDrop, OnFileDropOff, ClipboardGetText } from '../../wailsjs/runtime/runtime'

type MapInfo = {
  name: string; path: string; source: string; modFolder?: string
}

const maps = ref<MapInfo[]>([])
const msg = ref('')
const dropActive = ref(false)

onMounted(async () => {
  await loadMaps()
  OnFileDrop(async (x: number, y: number, paths: string[]) => {
    dropActive.value = false
    for (const p of paths) {
      try {
        const result = await ImportFile(p)
        msg.value = result
        await loadMaps()
      } catch (e: any) {
        msg.value = '导入失败: ' + e
      }
    }
  }, true)
})

onUnmounted(() => {
  OnFileDropOff()
})

async function loadMaps() {
  try {
    maps.value = await ScanMaps()
  } catch (e) {
    console.error('ScanMaps:', e)
  }
}

async function pasteImport() {
  try {
    const text = await ClipboardGetText()
    if (!text) { msg.value = '剪贴板为空'; return }
    await SaveClipboardAsMap(text)
    msg.value = '已从剪贴板导入地图'
    await loadMaps()
  } catch (e: any) { msg.value = '导入失败: ' + e }
}

function sourceLabel(s: string): string {
  return s === 'maps' ? '独立地图' : s === 'mod' ? '模组自带' : s
}
</script>

<template>
  <div
    class="maps-view"
    :class="{ 'drop-active': dropActive }"
    @dragenter.prevent="dropActive = true"
    @dragleave.prevent="dropActive = false"
    @drop.prevent="dropActive = false"
  >
    <div class="view-header">
      <h1>🗺️ 地图</h1>
      <button class="btn-refresh" @click="loadMaps">刷新</button>
      <button class="btn-paste" @click="pasteImport">📋 粘贴导入</button>
    </div>

    <div v-if="msg" class="toast">{{ msg }}</div>

    <div
      class="drop-zone"
      :class="{ active: dropActive }"
      @dragenter.prevent="dropActive = true"
    >
      {{ dropActive ? '📥 释放以导入' : '拖拽 ZIP 或 .civ5map 文件到此处导入' }}
    </div>

    <div v-if="maps.length === 0" class="placeholder">
      <p>未找到地图文件</p>
      <p class="hint">将 .civ5map 文件拖入窗口，或放入 maps/ 目录</p>
    </div>

    <div v-else class="map-list">
      <div v-for="m in maps" :key="m.path" class="map-card">
        <div class="map-icon">🗺️</div>
        <div class="map-info">
          <div class="map-name">{{ m.name }}</div>
          <div class="map-meta">
            <span class="source-tag" :class="m.source">{{ sourceLabel(m.source) }}</span>
            <span v-if="m.modFolder" class="mod-folder">{{ m.modFolder }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.maps-view { height: 100%; max-width: 800px; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 600; }
.btn-refresh { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-paste { padding: 8px 14px; background: var(--bg-card); color: var(--text-secondary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-paste:hover { color: var(--text-primary); }

.toast { background: rgba(78,205,196,0.1); color: #4ecdc4; padding: 10px 14px; border-radius: 4px; margin-bottom: 12px; font-size: 13px; }

.drop-zone { border: 2px dashed var(--border-color); border-radius: 8px; padding: 40px; text-align: center; color: var(--text-muted); font-size: 14px; transition: all 0.2s; margin-bottom: 16px; }
.drop-zone.active { border-color: var(--accent); color: var(--accent); background: rgba(74,158,255,0.05); }

.placeholder { text-align: center; padding: 60px; color: var(--text-muted); }
.hint { font-size: 13px; margin-top: 8px; }

.map-list { display: flex; flex-direction: column; gap: 8px; }
.map-card { display: flex; align-items: center; gap: 12px; background: var(--bg-card); border-radius: 6px; padding: 12px 16px; box-shadow: var(--card-shadow); }
.map-icon { font-size: 24px; }
.map-info { flex: 1; }
.map-name { font-size: 14px; font-weight: 500; margin-bottom: 2px; }
.map-meta { display: flex; gap: 8px; align-items: center; }
.source-tag { padding: 1px 6px; border-radius: 3px; font-size: 11px; }
.source-tag.maps { background: rgba(74,158,255,0.15); color: var(--accent); }
.source-tag.mod { background: rgba(180,78,255,0.15); color: #b44eff; }
.mod-folder { font-size: 11px; color: var(--text-muted); }
</style>
