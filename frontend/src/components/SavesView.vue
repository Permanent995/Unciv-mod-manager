<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ScanSaves, DeleteSave, ArchiveSave,
  ListSaveArchives, RestoreSaveArchive, DeleteSaveArchive
} from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'
import IconArchive from './icons/IconArchive.vue'
import IconTrash from './icons/IconTrash.vue'

const saves = ref<app.SaveInfo[]>([])
const loading = ref(false)
const msg = ref('')
const selected = ref<app.SaveInfo | null>(null)

const archives = ref<app.SaveArchive[]>([])
const archivesLoading = ref(false)
const archiving = ref(false)
const deleting = ref(false)
const restoring = ref<string | null>(null)
const deletingArchive = ref<string | null>(null)

onMounted(() => { loadSaves() })

async function loadSaves(): Promise<void> {
  loading.value = true
  msg.value = ''
  selected.value = null
  try {
    saves.value = await ScanSaves()
    if (saves.value.length === 0) {
      msg.value = '未找到存档文件'
    } else {
      selected.value = saves.value[0]
      loadArchives(selected.value.name)
    }
  } catch (e: unknown) {
    msg.value = '读取失败: ' + String(e)
  } finally {
    loading.value = false
  }
}

function selectSave(s: app.SaveInfo): void {
  selected.value = s
  msg.value = ''
  loadArchives(s.name)
}

async function loadArchives(origName: string): Promise<void> {
  archivesLoading.value = true
  archives.value = []
  try {
    const all = await ListSaveArchives()
    archives.value = all.filter(a => a.origName === origName)
  } catch (e: unknown) {
    msg.value = '加载备份失败: ' + String(e)
  } finally {
    archivesLoading.value = false
  }
}

async function deleteSelectedSave(): Promise<void> {
  if (!selected.value) return
  if (!confirm(`确定删除存档 "${selected.value.name}"？此操作不可恢复。`)) return
  deleting.value = true
  const prevIndex = saves.value.findIndex(s => s.path === selected.value!.path)
  const deletedName = selected.value.name
  try {
    await DeleteSave(selected.value.path)
    msg.value = '已删除: ' + deletedName
    saves.value = saves.value.filter(s => s.path !== selected.value!.path)
    if (saves.value.length === 0) {
      selected.value = null
      archives.value = []
    } else {
      const nextIndex = Math.min(prevIndex, saves.value.length - 1)
      selected.value = saves.value[nextIndex]
      loadArchives(selected.value.name)
    }
  } catch (e: unknown) {
    msg.value = '删除失败: ' + String(e)
  } finally {
    deleting.value = false
  }
}

async function archiveSelectedSave(): Promise<void> {
  if (!selected.value) return
  archiving.value = true
  try {
    const result = await ArchiveSave(selected.value.path)
    msg.value = '已备份存档'
    await loadArchives(selected.value.name)
  } catch (e: unknown) {
    msg.value = '备份失败: ' + String(e)
  } finally {
    archiving.value = false
  }
}

async function restoreArchive(a: app.SaveArchive): Promise<void> {
  if (!confirm(`确定从备份 "${formatTimestamp(a.timestamp)}" 还原？当前同名存档将被覆盖。`)) return
  restoring.value = a.path
  try {
    await RestoreSaveArchive(a.path)
    msg.value = '已从备份还原'
    await loadSaves()
    if (selected.value) loadArchives(selected.value.name)
  } catch (e: unknown) {
    msg.value = '还原失败: ' + String(e)
  } finally {
    restoring.value = null
  }
}

async function deleteArchive(a: app.SaveArchive): Promise<void> {
  if (!confirm(`确定删除备份 "${formatTimestamp(a.timestamp)}"？`)) return
  deletingArchive.value = a.path
  try {
    await DeleteSaveArchive(a.path)
    archives.value = archives.value.filter(x => x.path !== a.path)
    msg.value = '已删除备份'
  } catch (e: unknown) {
    msg.value = '删除备份失败: ' + String(e)
  } finally {
    deletingArchive.value = null
  }
}

function formatTimestamp(ts: string): string {
  return ts.replace(/_/g, ' ')
}

function formatSize(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(0) + ' KB'
  return (n / 1024 / 1024).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="saves-view view-card">
    <div class="view-header">
      <h1><IconArchive :size="24" /> 存档</h1>
      <button class="btn-refresh" :disabled="loading" @click="loadSaves">
        {{ loading ? '加载中...' : '刷新' }}
      </button>
    </div>

    <div v-if="msg" class="toast" :class="{ ok: msg.startsWith('已') }">{{ msg }}</div>

    <div v-if="loading && saves.length === 0" class="loading">加载中...</div>

    <template v-else-if="saves.length > 0">
      <div class="master-detail">
        <div class="save-list">
          <div
            v-for="s in saves" :key="s.name"
            class="save-card" :class="{ active: selected?.name === s.name }"
            @click="selectSave(s)"
          >
            <div class="save-info">
              <div class="save-name">{{ s.name }}</div>
              <div class="save-meta">
                <span v-if="s.civName" class="badge-civ">{{ s.civName }}</span>
                <span v-if="s.turn" class="badge-turn">回合 {{ s.turn }}</span>
                <span v-if="s.version" class="badge-ver">v{{ s.version }}</span>
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
          <div class="detail-grid">
            <div class="dr"><span class="dk">文明</span><span class="dv">{{ selected.civName || '未知' }}</span></div>
            <div class="dr"><span class="dk">回合</span><span class="dv">{{ selected.turn || '-' }}</span></div>
            <div class="dr"><span class="dk">版本</span><span class="dv">{{ selected.version || '未知' }}</span></div>
            <div class="dr"><span class="dk">修改时间</span><span class="dv">{{ selected.modifiedAt }}</span></div>
            <div class="dr"><span class="dk">大小</span><span class="dv">{{ formatSize(selected.fileSize) }}</span></div>
          </div>

          <div class="mods-section" v-if="selected.mods && selected.mods.length">
            <h3>使用的模组</h3>
            <div class="mods-list">
              <span v-for="m in selected.mods" :key="m" class="mod-tag">{{ m }}</span>
            </div>
          </div>

          <div class="actions">
            <button class="btn btn-danger" :disabled="deleting" @click="deleteSelectedSave">
              <IconTrash :size="16" /> {{ deleting ? '删除中...' : '删除存档' }}
            </button>
            <button class="btn btn-archive" :disabled="archiving" @click="archiveSelectedSave">
              <IconArchive :size="16" /> {{ archiving ? '备份中...' : '备份存档' }}
            </button>
          </div>

          <details class="archive-section" @toggle="(e: Event) => { if ((e.target as HTMLDetailsElement).open && selected) loadArchives(selected.name) }">
            <summary>📋 备份历史</summary>
            <div v-if="archivesLoading" class="loading-sm">加载中...</div>
            <div v-else-if="archives.length === 0" class="archive-empty">暂无备份</div>
            <div v-else class="archive-list">
              <div v-for="a in archives" :key="a.path" class="archive-row">
                <div class="archive-info">
                  <span class="archive-ts">{{ formatTimestamp(a.timestamp) }}</span>
                  <span class="archive-size">{{ formatSize(a.fileSize) }}</span>
                </div>
                <div class="archive-actions">
                  <button class="btn-sm" :disabled="restoring === a.path" @click="restoreArchive(a)">
                    {{ restoring === a.path ? '还原中...' : '还原' }}
                  </button>
                  <button class="btn-sm danger" :disabled="deletingArchive === a.path" @click="deleteArchive(a)">
                    {{ deletingArchive === a.path ? '删除中...' : '删除' }}
                  </button>
                </div>
              </div>
            </div>
          </details>
        </div>
      </div>
    </template>

    <div v-else-if="!loading" class="empty">
      <p>未找到存档文件</p>
      <p class="hint">存档位于 Unciv/SaveFiles/ 目录</p>
    </div>
  </div>
</template>

<style scoped>
.saves-view { height: 100%; display: flex; flex-direction: column; }

.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; flex-shrink: 0; }
.view-header h1 { font-size: 22px; font-weight: 700; color: var(--text-primary); }
.btn-refresh { padding: 6px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-refresh:disabled { opacity: 0.5; }

.toast { padding: 10px 14px; border-radius: 4px; margin-bottom: 12px; font-size: 13px; background: rgba(78,205,196,0.1); color: #4ecdc4; flex-shrink: 0; }
.toast.ok { background: rgba(78,205,196,0.12); color: #2dd4bf; }

.loading, .empty { padding: 40px; text-align: center; color: var(--text-muted); }
.hint { font-size: 13px; margin-top: 6px; }

.master-detail { display: flex; gap: 14px; flex: 1; overflow: hidden; }

.save-list { width: 270px; flex-shrink: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 3px; }
.save-card { display: flex; padding: 8px 10px; border-radius: 6px; cursor: pointer; background: var(--bg-card); border: 1px solid transparent; transition: all 0.15s; }
.save-card:hover { border-color: var(--accent); }
.save-card.active { border-color: var(--accent); background: var(--bg-active); }
.save-info { flex: 1; min-width: 0; }
.save-name { font-size: 13px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.save-meta { display: flex; gap: 4px; flex-wrap: wrap; margin: 2px 0; }
.badge-civ { font-size: 11px; padding: 1px 5px; background: var(--accent); color: #fff; border-radius: 3px; font-weight: 600; }
.badge-turn { font-size: 11px; color: var(--text-secondary); font-weight: 600; }
.badge-ver { font-size: 10px; color: var(--text-muted); }
.save-sub { font-size: 11px; color: var(--text-muted); display: flex; gap: 10px; }

.save-detail { flex: 1; overflow-y: auto; }
.save-detail h2 { font-size: 18px; font-weight: 700; margin-bottom: 10px; }
.detail-grid { margin-bottom: 12px; }
.dr { display: flex; padding: 4px 0; border-bottom: 1px solid var(--border-color); font-size: 13px; }
.dk { width: 60px; color: var(--text-muted); flex-shrink: 0; }
.dv { color: var(--text-primary); font-weight: 600; }

.mods-section { margin-bottom: 12px; }
.mods-section h3 { font-size: 14px; font-weight: 700; margin-bottom: 6px; }
.mods-list { display: flex; flex-wrap: wrap; gap: 4px; }
.mod-tag { padding: 2px 8px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 4px; font-size: 12px; font-weight: 600; }

.actions { display: flex; gap: 8px; margin-bottom: 12px; }
.btn { padding: 6px 14px; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 600; border: 1px solid var(--danger); color: var(--danger); background: transparent; display: inline-flex; align-items: center; gap: 4px; }
.btn:hover { background: rgba(239,68,68,0.08); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-archive { border-color: var(--accent); color: var(--accent); display: inline-flex; align-items: center; gap: 4px; }
.btn-archive:hover { background: rgba(79,70,229,0.08); }

.archive-section { margin-bottom: 12px; font-size: 13px; }
.archive-section summary { cursor: pointer; color: var(--text-primary); margin-bottom: 6px; font-weight: 600; padding: 4px 0; }
.archive-section summary:hover { color: var(--text-primary); }
.archive-empty { color: var(--text-muted); font-size: 12px; padding: 8px 0; }
.loading-sm { font-size: 12px; color: var(--text-muted); padding: 8px 0; }
.archive-list { display: flex; flex-direction: column; gap: 4px; }
.archive-row { display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; background: var(--bg-card); border-radius: 4px; }
.archive-info { display: flex; gap: 10px; align-items: center; }
.archive-ts { font-size: 12px; font-weight: 500; }
.archive-size { font-size: 11px; color: var(--text-muted); }
.archive-actions { display: flex; gap: 4px; }

.btn-sm { padding: 4px 10px; border-radius: 3px; cursor: pointer; font-size: 11px; font-weight: 600; border: 1px solid var(--accent); color: var(--accent); background: transparent; }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-sm.danger { border-color: var(--danger); color: var(--danger); }
.btn-sm:hover { background: rgba(79,70,229,0.08); }
.btn-sm.danger:hover { background: rgba(239,68,68,0.08); }
</style>
