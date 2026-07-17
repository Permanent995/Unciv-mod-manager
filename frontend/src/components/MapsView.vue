<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ScanMaps, ImportFile, SaveClipboardMap, DeleteMap, RenameMap } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'
import { OnFileDrop, OnFileDropOff, ClipboardGetText } from '../../wailsjs/runtime/runtime'

type MapInfo = {
  name: string; path: string; source: string; modFolder?: string
}

const maps = ref<MapInfo[]>([])
const msg = ref('')
const dropActive = ref(false)
const deleting = ref<string | null>(null)

// ── Modal state ──
const showModal = ref(false)
const modalTitle = ref('')
const modalInput = ref('')
const modalPlaceholder = ref('')
let modalResolve: ((val: string | null) => void) | null = null

watch(showModal, async (v) => {
  if (v) {
    await nextTick()
    document.querySelector<HTMLInputElement>('.modal-input')?.focus()
  }
})

function openModal(title: string, initial: string, placeholder: string): Promise<string | null> {
  modalTitle.value = title
  modalInput.value = initial
  modalPlaceholder.value = placeholder
  showModal.value = true
  return new Promise(r => { modalResolve = r })
}

function modalConfirm() {
  showModal.value = false
  modalResolve?.(modalInput.value.trim() || null)
  modalResolve = null
}

function modalCancel() {
  showModal.value = false
  modalResolve?.(null)
  modalResolve = null
}

function modalKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') modalConfirm()
  if (e.key === 'Escape') modalCancel()
}

// ── Paste import ──
async function pasteImport() {
  try {
    const text = await ClipboardGetText()
    if (!text) { msg.value = '剪贴板为空'; return }

    const name = await openModal('导入地图', '', '输入地图名称（留空自动生成）')
    if (name === null) return

    const filename = await SaveClipboardMap(text, name || '')
    msg.value = '已导入: ' + filename
    await loadMaps()
  } catch (e: any) { msg.value = '导入失败: ' + e }
}

// ── Delete ──
async function deleteMap(m: MapInfo) {
  if (m.source !== 'maps') {
    msg.value = '模组自带地图请在模组目录中管理'
    return
  }
  if (deleting.value) return
  if (!confirm(`确定删除地图 "${m.name}"？`)) return
  deleting.value = m.path
  try {
    await DeleteMap(m.path)
    msg.value = '已删除: ' + m.name
    maps.value = maps.value.filter(x => x.path !== m.path)
  } catch (e: any) {
    msg.value = '删除失败: ' + e
  } finally {
    deleting.value = null
  }
}

// ── Rename ──
async function renameMap(m: MapInfo) {
  if (m.source !== 'maps') {
    msg.value = '模组自带地图不能重命名'
    return
  }
  const name = await openModal('重命名地图', m.name, '输入新名称')
  if (!name || name === m.name) return

  try {
    const newName = await RenameMap(m.path, name)
    msg.value = '已重命名: ' + newName
    await loadMaps()
  } catch (e: any) {
    msg.value = '重命名失败: ' + e
  }
}

// ── Lifecycle ──
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

onUnmounted(() => { OnFileDropOff() })

async function loadMaps() {
  try { maps.value = await ScanMaps() }
  catch (e) { console.error('ScanMaps:', e) }
}

function sourceLabel(s: string): string {
  return s === 'maps' ? '独立地图' : s === 'mod' ? '模组自带' : s
}
</script>

<template>
  <div class="maps-view" @dragenter.prevent="dropActive = true" @dragleave.prevent="dropActive = false" @drop.prevent="dropActive = false">
    <div class="view-header">
      <h1>🗺️ 地图</h1>
      <div class="header-right">
        <button class="btn-paste" @click="pasteImport">📋 粘贴导入</button>
        <button class="btn-refresh" @click="loadMaps">刷新</button>
      </div>
    </div>

    <div v-if="msg" class="toast" :class="{ ok: msg.startsWith('已导入') || msg.startsWith('已删除') || msg.startsWith('已重命名') }">{{ msg }}</div>

    <div class="drop-zone" :class="{ active: dropActive }">
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
        <div v-if="m.source === 'maps'" class="map-actions">
          <button class="btn-icon" @click="renameMap(m)" title="重命名">✏️</button>
          <button class="btn-icon del" :disabled="deleting === m.path" @click="deleteMap(m)" title="删除">🗑</button>
        </div>
      </div>
    </div>

    <!-- ── Centered modal ── -->
    <Teleport to="body">
      <div v-if="showModal" class="modal-overlay" @click.self="modalCancel">
        <div class="modal-dialog" @keydown="modalKeydown">
          <div class="modal-header">{{ modalTitle }}</div>
          <input
            ref="modalInputRef"
            v-model="modalInput"
            class="modal-input"
            :placeholder="modalPlaceholder"
            autofocus
            @keydown="modalKeydown"
          />
          <div class="modal-buttons">
            <button class="modal-btn" @click="modalCancel">取消</button>
            <button class="modal-btn primary" @click="modalConfirm">确定</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>


<style scoped>
.maps-view { height: 100%; min-height: 0; max-width: 800px; overflow-y: auto; }
.maps-view::-webkit-scrollbar { width: 6px; }
.maps-view::-webkit-scrollbar-thumb { background: var(--border-color); border-radius: 3px; }
.maps-view::-webkit-scrollbar-thumb:hover { background: var(--text-muted); }

.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-shrink: 0; }
.view-header h1 { font-size: 24px; font-weight: 600; }
.header-right { display: flex; gap: 8px; }
.btn-refresh { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-paste { padding: 8px 14px; background: var(--bg-card); color: var(--text-secondary); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-paste:hover { color: var(--text-primary); }

.toast { padding: 10px 14px; border-radius: 4px; margin-bottom: 12px; font-size: 13px; background: rgba(78,205,196,0.1); color: #4ecdc4; flex-shrink: 0; }
.toast.ok { background: rgba(78,205,196,0.12); color: #2dd4bf; }

.drop-zone { border: 2px dashed var(--border-color); border-radius: 8px; padding: 40px; text-align: center; color: var(--text-muted); font-size: 14px; transition: all 0.2s; margin-bottom: 16px; flex-shrink: 0; }
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

.map-actions { display: flex; gap: 2px; }
.btn-icon { background: none; border: none; cursor: pointer; font-size: 15px; padding: 4px 6px; border-radius: 4px; opacity: 0.4; transition: all 0.15s; }
.btn-icon:hover { opacity: 1; background: rgba(100,116,139,0.1); }
.btn-icon.del:hover { background: rgba(255,77,79,0.1); }
.btn-icon:disabled { opacity: 0.2; cursor: default; }
</style>

<style>
/* Unscoped: modal over body */
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex; align-items: center; justify-content: center;
  z-index: 9999;
}
.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
  min-width: 320px;
  max-width: 420px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.2);
}
.modal-header {
  font-size: 16px; font-weight: 700;
  margin-bottom: 14px;
  color: var(--text-primary);
}
.modal-input {
  width: 100%; box-sizing: border-box;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--bg-input);
  color: var(--text-primary);
  outline: none;
  margin-bottom: 16px;
}
.modal-input:focus { border-color: var(--accent); }
.modal-buttons {
  display: flex; justify-content: flex-end; gap: 8px;
}
.modal-btn {
  padding: 8px 18px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px; font-weight: 600;
  cursor: pointer;
  background: var(--bg-card);
  color: var(--text-primary);
}
.modal-btn:hover { border-color: var(--accent); }
.modal-btn.primary {
  background: var(--accent); color: #fff; border-color: var(--accent);
}
.modal-btn.primary:hover { opacity: 0.9; }
</style>
