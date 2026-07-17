<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ScanSaves, DeleteSave, ArchiveSave, ListSaveArchives, RestoreSaveArchive, DeleteSaveArchive } from '../../wailsjs/go/app/App'

type SaveInfo = {
  name: string; path: string; fileSize: number
  modifiedAt: string; civName?: string; turn?: number; version?: string
  mods?: string[]
}

type SaveArchive = {
  name: string; origName: string; timestamp: string
  path: string; fileSize: number; modifiedAt: string
}

const saves = ref<SaveInfo[]>([])
const loading = ref(false)
const msg = ref('')
const selected = ref<SaveInfo | null>(null)
const deleting = ref(false)

// Backup state
const archives = ref<SaveArchive[]>([])
const archiving = ref(false)
const restoring = ref<SaveArchive | null>(null)

onMounted(() => { loadSaves(); loadArchives() })

async function loadSaves() {
  loading.value = true
  msg.value = ''
  selected.value = null
  try {
    saves.value = await ScanSaves()
    if (saves.value.length === 0) msg.value = '未找到存档文件'
  } catch (e: any) {
    msg.value = '读取失败: ' + e
  } finally {
    loading.value = false
  }
}

async function loadArchives() {
  try {
    archives.value = await ListSaveArchives()
  } catch (_) { /* ignore */ }
}

function archivesForCurrent(): SaveArchive[] {
  if (!selected.value) return []
  return archives.value.filter(a => a.origName === selected.value!.name)
}

async function archiveCurrent() {
  if (!selected.value || archiving.value) return
  archiving.value = true
  msg.value = ''
  try {
    await ArchiveSave(selected.value.path)
    msg.value = '已备份: ' + selected.value.name
    await loadArchives()
  } catch (e: any) {
    msg.value = '备份失败: ' + e
  } finally {
    archiving.value = false
  }
}

async function restoreArchive(a: SaveArchive) {
  if (restoring.value) return
  restoring.value = a
  msg.value = ''
  try {
    const dst = await RestoreSaveArchive(a.path)
    msg.value = '已恢复到: ' + dst
    await loadSaves()
  } catch (e: any) {
    msg.value = '恢复失败: ' + e
  } finally {
    restoring.value = null
  }
}

async function deleteArchive(a: SaveArchive) {
  if (!confirm(`确定删除备份 "${a.name}"？`)) return
  try {
    await DeleteSaveArchive(a.path)
    msg.value = '已删除备份: ' + a.name
    await loadArchives()
  } catch (e: any) {
    msg.value = '删除备份失败: ' + e
  }
}

async function deleteSave(s: SaveInfo) {
  if (deleting.value) return
  if (!confirm(`确定删除存档 "${s.name}"？\n此操作不可恢复`)) return
  deleting.value = true
  msg.value = ''
  try {
    await DeleteSave(s.path)
    msg.value = '已删除: ' + s.name
    saves.value = saves.value.filter(x => x.name !== s.name)
    if (selected.value?.name === s.name) selected.value = null
  } catch (e: any) {
    msg.value = '删除失败: ' + e
  } finally {
    deleting.value = false
  }
}

function selectSave(s: SaveInfo) {
  selected.value = s
  msg.value = ''
}

function formatSize(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(0) + ' KB'
  return (n / 1024 / 1024).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="saves-view">
    <div class="view-header">
      <h1>💾 存档</h1>
      <button class="btn-refresh" :disabled="loading" @click="loadSaves">{{ loading ? '加载中...' : '刷新' }}</button>
    </div>

    <div v-if="msg" class="msg" :class="{ ok: msg.startsWith('已删除') || msg.startsWith('已备份') || msg.startsWith('已恢复') }">{{ msg }}</div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="saves.length > 0" class="master-detail">
      <div class="save-list">
        <div
          v-for="s in saves"
          :key="s.name"
          class="save-card"
          :class="{ active: selected?.name === s.name }"
          @click="selectSave(s)"
        >
          <div class="save-icon">💾</div>
          <div class="save-info">
            <div class="save-name">{{ s.name }}</div>
            <div class="save-meta">
              <span v-if="s.civName" class="save-civ">{{ s.civName }}</span>
              <span v-if="s.turn" class="save-turn">回合 {{ s.turn }}</span>
              <span class="save-ver" v-if="s.version">v{{ s.version }}</span>
            </div>
            <div class="save-sub">
              <span>{{ s.modifiedAt }}</span>
              <span>{{ formatSize(s.fileSize) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-if="selected" class="save-detail">
        <h2>{{ selected.name }}</h2>
        <div class="detail-meta">
          <div class="detail-row"><span class="dk">文明</span><span class="dv">{{ selected.civName || '未知' }}</span></div>
          <div class="detail-row"><span class="dk">回合</span><span class="dv">{{ selected.turn || '-' }}</span></div>
          <div class="detail-row"><span class="dk">版本</span><span class="dv">{{ selected.version || '未知' }}</span></div>
          <div class="detail-row"><span class="dk">修改时间</span><span class="dv">{{ selected.modifiedAt }}</span></div>
          <div class="detail-row"><span class="dk">大小</span><span class="dv">{{ formatSize(selected.fileSize) }}</span></div>
        </div>

        <div v-if="selected.mods && selected.mods.length > 0" class="mods-section">
          <h3>使用的模组</h3>
          <div class="mods-list">
            <span v-for="m in selected.mods" :key="m" class="mod-tag">{{ m }}</span>
          </div>
        </div>
        <div v-else class="mods-section">
          <h3>使用的模组</h3>
          <p class="no-mods">无扩展模组</p>
        </div>

        <div class="detail-actions">
          <button class="btn-archive" :disabled="archiving" @click="archiveCurrent">
            {{ archiving ? '备份中...' : '📦 备份存档' }}
          </button>
          <button class="btn-delete" :disabled="deleting" @click="deleteSave(selected)">
            {{ deleting ? '删除中...' : '🗑 删除存档' }}
          </button>
        </div>

        <!-- Backup history -->
        <div v-if="archivesForCurrent().length > 0" class="archive-section">
          <h3>备份历史</h3>
          <div class="archive-list">
            <div v-for="a in archivesForCurrent()" :key="a.name" class="archive-item">
              <div class="archive-info">
                <span class="archive-ts">{{ a.timestamp.replace('_', ' ') }}</span>
                <span class="archive-size">{{ formatSize(a.fileSize) }}</span>
              </div>
              <div class="archive-actions">
                <button class="btn-restore" :disabled="restoring === a" @click="restoreArchive(a)">
                  {{ restoring === a ? '恢复中...' : '恢复' }}
                </button>
                <button class="btn-del-arc" @click="deleteArchive(a)">删除</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="!loading" class="empty">
      <p>未找到存档</p>
      <p class="hint">存档位于 Unciv/SaveFiles/ 目录</p>
    </div>
  </div>
</template>

<style scoped>
.saves-view { height: 100%; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 700; }
.btn-refresh { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
.btn-refresh:disabled { opacity: 0.6; cursor: not-allowed; }
.msg { padding: 10px 14px; border-radius: 4px; margin-bottom: 12px; font-size: 14px; background: rgba(74,158,255,0.08); color: var(--accent); }
.msg.ok { background: rgba(34,197,94,0.08); color: var(--success); }
.loading, .empty { text-align: center; padding: 60px; color: var(--text-muted); }
.hint { font-size: 13px; margin-top: 8px; color: var(--text-muted); }

.master-detail { display: flex; gap: 16px; height: calc(100% - 80px); }
.save-list { width: 300px; flex-shrink: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; }
.save-card {
  display: flex; align-items: center; gap: 12px;
  background: var(--bg-card); border: 1px solid transparent;
  border-radius: 8px; padding: 12px 14px;
  cursor: pointer; transition: all 0.15s;
  box-shadow: var(--card-shadow);
}
.save-card:hover { border-color: var(--accent); }
.save-card.active { border-color: var(--accent); background: var(--bg-active); }
.save-icon { font-size: 24px; }
.save-info { flex: 1; min-width: 0; }
.save-name { font-size: 14px; font-weight: 600; margin-bottom: 2px; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.save-meta { display: flex; gap: 6px; align-items: center; margin-bottom: 1px; flex-wrap: wrap; }
.save-civ { font-size: 12px; padding: 1px 6px; background: var(--accent); color: #fff; border-radius: 3px; font-weight: 600; }
.save-turn { font-size: 12px; color: var(--text-secondary); font-weight: 600; }
.save-ver { font-size: 11px; color: var(--text-muted); }
.save-sub { display: flex; gap: 12px; font-size: 11px; color: var(--text-muted); }

.save-detail { flex: 1; overflow-y: auto; }
.save-detail h2 { font-size: 20px; font-weight: 700; margin-bottom: 14px; }
.detail-meta { margin-bottom: 16px; }
.detail-row { display: flex; padding: 6px 0; border-bottom: 1px solid var(--border-color); font-size: 14px; }
.dk { width: 70px; color: var(--text-muted); flex-shrink: 0; }
.dv { color: var(--text-primary); font-weight: 600; }

.mods-section { margin-bottom: 16px; }
.mods-section h3 { font-size: 15px; font-weight: 700; margin-bottom: 8px; }
.mods-list { display: flex; flex-wrap: wrap; gap: 6px; }
.mod-tag { padding: 3px 10px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 4px; font-size: 13px; color: var(--text-primary); font-weight: 600; }
.no-mods { font-size: 13px; color: var(--text-muted); }

.detail-actions { display: flex; gap: 8px; margin-bottom: 16px; }
.btn-archive { padding: 8px 18px; background: rgba(34,197,94,0.08); color: var(--success); border: 1px solid var(--success); border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
.btn-archive:hover { background: rgba(34,197,94,0.18); }
.btn-archive:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-delete { padding: 8px 18px; background: rgba(255,77,79,0.1); color: var(--danger); border: 1px solid var(--danger); border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
.btn-delete:hover { background: rgba(255,77,79,0.2); }
.btn-delete:disabled { opacity: 0.5; cursor: not-allowed; }

.archive-section { margin-bottom: 16px; }
.archive-section h3 { font-size: 15px; font-weight: 700; margin-bottom: 8px; }
.archive-list { display: flex; flex-direction: column; gap: 4px; }
.archive-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 12px; background: var(--bg-card); border-radius: 6px;
  border: 1px solid var(--border-color);
}
.archive-info { display: flex; gap: 12px; align-items: center; font-size: 13px; }
.archive-ts { font-weight: 600; color: var(--text-primary); }
.archive-size { color: var(--text-muted); }
.archive-actions { display: flex; gap: 6px; }
.btn-restore { padding: 3px 10px; background: rgba(74,158,255,0.08); color: var(--accent); border: 1px solid var(--accent); border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.btn-restore:hover { background: rgba(74,158,255,0.18); }
.btn-restore:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-del-arc { padding: 3px 10px; background: rgba(255,77,79,0.08); color: var(--danger); border: 1px solid var(--danger); border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.btn-del-arc:hover { background: rgba(255,77,79,0.18); }
</style>
