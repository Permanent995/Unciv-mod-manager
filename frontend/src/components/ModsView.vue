<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ScanMods, GetAppConfig } from '../../wailsjs/go/app/App'
import { ReadModReadme, ReadModPreview } from '../../wailsjs/go/app/App'
import { ListBackups, RestoreBackup, DeleteBackup } from '../../wailsjs/go/app/App'
import { BackupMod } from '../../wailsjs/go/app/App'
import { TranslateText } from '../../wailsjs/go/app/App'
import { marked } from 'marked'
import { CheckModUpdates, DownloadAllUpdates } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'

type ModInfo = app.ModInfo

const mods = ref<ModInfo[]>([])
const loading = ref(false)
const uncivPath = ref('')
const sortMode = ref<'name' | 'category' | 'size'>('name')
const selected = ref<ModInfo | null>(null)
const readme = ref('')
const renderedReadme = computed(() => readme.value ? marked(readme.value) : '')
const readmeLoading = ref(false)
const translated = ref('')
const translating = ref(false)
const previewSrc = ref('')
const previewLoading = ref(false)
const backups = ref<any[]>([])
const backupsLoading = ref(false)
const backupMsg = ref('')
const restoring = ref(false)
const deleting = ref(false)

const delMsg = ref('')
const scanErr = ref('')

// ── Update check ──
const updates = ref<{ folder: string; name: string; currentVer: string; latestVer: string; modUrl: string; hasUpdate: boolean }[]>([])
const checkingUpdates = ref(false)
const updatingAll = ref(false)
const updateMsg = ref('')
const updateMsgType = ref<'ok' | 'info' | ''>('')

async function checkUpdates() {
  checkingUpdates.value = true
  updateMsg.value = ''
  updateMsgType.value = ''
  try {
    updates.value = await CheckModUpdates()
    const total = updates.value.length
    const updatable = updates.value.filter(u => u.hasUpdate).length
    if (total === 0) {
      updateMsg.value = '未找到可检查的模组（模组需有 GitHub 地址并已收录在 ModListCache 中）'
      updateMsgType.value = 'info'
    } else if (updatable === 0) {
      updateMsg.value = '✅ 所有模组已是最新'
      updateMsgType.value = 'ok'
    } else {
      updateMsg.value = `发现 ${updatable}/${total} 个模组可更新`
      updateMsgType.value = ''
    }
  } catch (e: any) {
    updateMsg.value = '检查失败: ' + e
    updateMsgType.value = ''
  } finally {
    checkingUpdates.value = false
  }
}

async function updateAll() {
  const pending = updates.value.filter(u => u.hasUpdate)
  if (pending.length === 0) return
  updatingAll.value = true
  try {
    await DownloadAllUpdates(pending)
    updateMsg.value = `${pending.length} 个更新已加入下载队列，请前往「下载」页面查看进度`
  } catch (e: any) {
    updateMsg.value = '批量下载失败: ' + e
  } finally {
    updatingAll.value = false
  }
}

async function deleteMod() {
  if (!selected.value) return
  if (!confirm(`确认删除 ${selected.value.name}？将先备份到 umm_backups/`)) return
  deleting.value = true
  try {
    await BackupMod(selected.value.folder, '')
    const { DeleteMod } = await import('../../wailsjs/go/app/App')
    await DeleteMod(selected.value.folder)
    delMsg.value = `${selected.value.name} 已备份并删除`
    selected.value = null
    await loadMods()
  } catch (e: any) { delMsg.value = '删除失败: ' + e }
  finally { deleting.value = false }
}

async function loadBackups() {
  backupsLoading.value = true; backups.value = []
  try {
    const all = await ListBackups()
    backups.value = all.filter((b: any) => b.folder === selected.value?.folder)
  } catch { }
  finally { backupsLoading.value = false }
}

async function doRestore(b: any) {
  if (!confirm(`还原 ${b.timestamp} 版本的 ${selected.value?.name}？`)) return
  restoring.value = true
  try {
    await RestoreBackup(b.path)
    backups.value = backups.value.filter((x: any) => x.path !== b.path)
    backupMsg.value = '已还原，刷新模组列表'
    await loadMods()
  } catch (e: any) { backupMsg.value = '还原失败: ' + e }
  finally { restoring.value = false }
}

async function doDeleteBackup(b: any) {
  if (!confirm(`删除 ${b.timestamp} 备份？`)) return
  try {
    await DeleteBackup(b.path)
    backups.value = backups.value.filter((x: any) => x.path !== b.path)
  } catch (e: any) { backupMsg.value = '删除失败: ' + e }
}

function formatSize(bytes: number): string {
  if (!bytes) return ''
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatTs(ts: string): string { return ts.replace(/_/g, ' ') }

const categoryOrder = ['ruleset', 'expansion', 'graphics', 'audio', 'map', 'fun', 'unclassified']

onMounted(async () => {
  try {
    const config = await GetAppConfig()
    uncivPath.value = config.uncivPath || ''
    await loadMods()
  } catch (e: any) {
    scanErr.value = '加载失败: ' + e
  }
})

async function loadMods() {
  loading.value = true
  scanErr.value = ''
  try {
    mods.value = await ScanMods()
    if (mods.value.length > 0 && !selected.value) selected.value = mods.value[0]
  } catch (e: any) {
    scanErr.value = '扫描模组失败: ' + e
  } finally { loading.value = false }
}

const sortedMods = computed(() => {
  const list = [...mods.value]
  if (sortMode.value === 'name')
    return list.sort((a, b) => (a.name || a.folder).localeCompare(b.name || b.folder))
  if (sortMode.value === 'size')
    return list.sort((a, b) => (b.modSize || 0) - (a.modSize || 0))
  return list.sort((a, b) => {
    const ai = categoryOrder.indexOf(a.category), bi = categoryOrder.indexOf(b.category)
    return (ai >= 0 ? ai : 99) - (bi >= 0 ? bi : 99) || (a.name || a.folder).localeCompare(b.name || b.folder)
  })
})

async function translateReadme() {
  if (!readme.value) return
  translating.value = true
  translated.value = ''
  try { translated.value = await TranslateText(readme.value) }
  catch (e: any) { translated.value = '翻译失败: ' + e }
  finally { translating.value = false }
}

async function selectMod(m: ModInfo) {
  selected.value = m
  readme.value = ''
  translated.value = ''
  previewSrc.value = ''
  if (m.hasReadme) {
    readmeLoading.value = true
    try { readme.value = await ReadModReadme(m.folder) }
    catch { readme.value = '' }
    finally { readmeLoading.value = false }
  }
  if (m.hasPreview) {
    previewLoading.value = true
    try { previewSrc.value = await ReadModPreview(m.folder) }
    catch { previewSrc.value = '' }
    finally { previewLoading.value = false }
  }
}

function catLabel(c: string): string {
  const m: Record<string, string> = { ruleset: '规则集', expansion: '扩展', graphics: '图形', audio: '音频', map: '地图', fun: '趣味', unclassified: '未分类' }
  return m[c] || c
}
function catColor(c: string): string {
  const m: Record<string, string> = { ruleset: '#ff6b6b', expansion: '#4ecdc4', graphics: '#45b7d1', audio: '#f7b731', map: '#5f27cd', fun: '#e67e22', unclassified: '#95a5a6' }
  return m[c] || '#95a5a6'
}

</script>

<template>
  <div class="mods-view view-card">
    <div class="view-header">
      <h1>模组库</h1>
      <div class="header-right">
        <div class="sort-bar">
          <button class="sort-btn" :class="{ active: sortMode === 'name' }" @click="sortMode = 'name'">A-Z</button>
          <button class="sort-btn" :class="{ active: sortMode === 'category' }" @click="sortMode = 'category'">类型</button>
          <button class="sort-btn" :class="{ active: sortMode === 'size' }" @click="sortMode = 'size'">大小</button>
        </div>
        <button class="refresh-btn" @click="loadMods" :disabled="loading">{{ loading ? '扫描中...' : '刷新' }}</button>
        <button class="update-btn" :disabled="checkingUpdates" @click="checkUpdates">{{ checkingUpdates ? '检查中...' : '🔄 检查更新' }}</button>
      </div>
    </div>

    <!-- Update notification -->
    <div v-if="scanErr" class="scan-error">{{ scanErr }}</div>
    <div v-if="delMsg" class="toast">{{ delMsg }}</div>
    <div v-if="updateMsg" class="update-banner" :class="updateMsgType">
      <span>{{ updateMsg }}</span>
      <div v-if="updates.some(u => u.hasUpdate)" class="update-actions">
        <button class="btn-update-all" :disabled="updatingAll" @click="updateAll">{{ updatingAll ? '添加中...' : '⬇ 一键更新全部' }}</button>
        <span class="update-hint">已添加到下载队列，解压后将覆盖旧版本</span>
      </div>
    </div>

    <!-- Update list detail -->
    <div v-if="updates.filter(u => u.hasUpdate).length > 0 && !updatingAll" class="update-list">
      <div v-for="u in updates.filter(u => u.hasUpdate)" :key="u.folder" class="update-row">
        <span class="u-name">{{ u.name }}</span>
        <span class="u-old">本地: {{ u.currentVer || '未知' }}</span>
        <span class="u-arrow">→</span>
        <span class="u-new">远端: {{ u.latestVer }}</span>
        <span class="u-badge">可更新</span>
      </div>
    </div>

    <div v-if="!uncivPath" class="no-path-warning">⚠️ 未设置 Unciv 路径，请先在设置中配置</div>

    <div v-if="loading" class="loading">正在扫描模组...</div>

    <div v-else-if="mods.length === 0" class="empty-state">未找到模组</div>

    <div v-else class="master-detail">
      <!-- Left: compact list -->
      <div class="mod-list">
        <div
          v-for="mod in sortedMods" :key="mod.folder"
          class="mod-item" :class="{ active: selected?.folder === mod.folder }"
          @click="selectMod(mod)"
        >
          <div class="item-main">
            <span class="item-name">{{ mod.name || mod.folder }}</span>
            <span v-if="mod.modSize" class="item-size">{{ formatSize(mod.modSize) }}</span>
          </div>
          <span class="item-cat" :style="{ background: catColor(mod.category) }">{{ catLabel(mod.category) }}</span>
        </div>
      </div>

      <!-- Right: detail -->
      <div class="mod-detail" v-if="selected">
        <h2>{{ selected.name || selected.folder }}</h2>
        <div class="detail-meta">
          <span class="cat-badge" :style="{ background: catColor(selected.category) }">{{ catLabel(selected.category) }}</span>
          <span v-if="selected.isBaseRuleset" class="ruleset-badge">基础规则集</span>
          <span v-if="selected.modSize" class="size-text">{{ formatSize(selected.modSize) }}</span>
        </div>
        <div v-if="selected.hasPreview" class="preview-section">
          <div v-if="previewLoading" class="preview-loading">加载预览图...</div>
          <img v-else-if="previewSrc" :src="previewSrc" class="preview-img" />
        </div>

        <div class="detail-table">
          <div class="row"><span class="k">文件夹</span><span class="v">{{ selected.folder }}</span></div>
          <div class="row"><span class="k">作者</span><span class="v">{{ selected.author || '未提供' }}</span></div>
          <div class="row" v-if="selected.modUrl"><span class="k">链接</span><span class="v">{{ selected.modUrl }}</span></div>
          <div class="row" v-if="selected.lastUpdated"><span class="k">更新</span><span class="v">{{ selected.lastUpdated }}</span></div>
          <div class="row"><span class="k">说明</span><span class="v">{{ selected.hasReadme ? '有 README' : '无说明文档' }}</span></div>
          <div class="row" v-if="selected.isIncomplete"><span class="k">注意</span><span class="v warn">ModOptions.json 不完整</span></div>
        </div>
        <div class="detail-actions-row">
          <button class="btn-del" :disabled="deleting" @click="deleteMod">{{ deleting ? '删除中...' : '🗑 删除（备份后删）' }}</button>
        </div>

        <!-- Backups -->
        <details class="backup-section" @toggle="(e: any) => { if ((e.target as HTMLElement).getAttribute('open') !== null) loadBackups() }">
          <summary>📦 备份管理</summary>
          <div v-if="backupsLoading" class="loading-sm">加载中...</div>
          <div v-else-if="backups.length === 0" class="backup-empty">暂无备份</div>
          <div v-else>
            <div v-for="b in backups" :key="b.path" class="backup-row">
              <div class="backup-info">
                <span class="backup-ts">{{ formatTs(b.timestamp) }}</span>
                <span v-if="b.version" class="backup-ver">v{{ b.version }}</span>
                <span v-if="b.size" class="backup-size">{{ formatSize(b.size) }}</span>
              </div>
              <div class="backup-actions">
                <button class="btn-sm" :disabled="restoring" @click="doRestore(b)">还原</button>
                <button class="btn-sm danger" @click="doDeleteBackup(b)">删</button>
              </div>
            </div>
          </div>
        </details>
        <div v-if="backupMsg" class="toast">{{ backupMsg }}</div>

        <div v-if="selected.topics && selected.topics.length" class="topics">
          <span v-for="t in selected.topics" :key="t" class="topic-tag">{{ t }}</span>
        </div>

        <div v-if="selected.hasReadme" class="readme-section">
          <div class="readme-header">
            <h3>📖 说明文档</h3>
            <button class="btn-trans" :disabled="translating || !readme" @click="translateReadme">
              {{ translating ? '翻译中...' : '🌐 翻译' }}
            </button>
          </div>
          <div v-if="readmeLoading" class="readme-loading">加载中...</div>
          <div v-else class="readme-text" v-html="renderedReadme"></div>
          <div v-if="translated" class="translated-section">
            <h3>📝 中文翻译</h3>
            <div class="readme-text" v-html="translated"></div>
          </div>
        </div>
      </div>
      <div class="mod-detail empty-detail" v-else>
        <p>选择左侧模组查看详情</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mods-view { height: 100%; display: flex; flex-direction: column; min-height: 0; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; flex-shrink: 0; }
.view-header h1 { font-size: 24px; font-weight: 600; color: var(--text-primary); }
.header-right { display: flex; align-items: center; gap: 12px; }

.sort-bar { display: flex; gap: 2px; }
.sort-btn { padding: 4px 10px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-secondary); cursor: pointer; font-size: 12px; }
.sort-btn:hover { color: var(--text-primary); }
.sort-btn.active { background: var(--accent); color: #fff; border-color: var(--accent); }
.refresh-btn { padding: 6px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.6; cursor: not-allowed; }

.no-path-warning { background: var(--danger); color: #fff; padding: 10px; border-radius: 4px; margin-bottom: 12px; flex-shrink: 0; }
.loading, .empty-state { text-align: center; padding: 40px; color: var(--text-muted); }

.master-detail { display: flex; flex: 1; overflow: hidden; gap: 0; min-height: 0; }

/* Left list */
.mod-list { width: 260px; flex-shrink: 0; overflow-y: auto; background: var(--bg-card); border-radius: 6px; }
.mod-list::-webkit-scrollbar { width: 6px; }
.mod-list::-webkit-scrollbar-thumb { background: var(--border-color); border-radius: 3px; }
.mod-list::-webkit-scrollbar-thumb:hover { background: var(--text-muted); }
.mod-item { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; cursor: pointer; border-bottom: 1px solid var(--border-color); transition: background 0.12s; gap: 6px; color: var(--text-primary); }
.mod-item:last-child { border-bottom: none; }
.mod-item:hover { background: var(--sidebar-active); }
.mod-item.active { background: var(--accent); color: #fff; }
.mod-item.active .item-cat { opacity: 1; }
.mod-item.active .item-size { color: rgba(255,255,255,0.7); }
.item-main { display: flex; flex-direction: column; flex: 1; min-width: 0; }
.item-name { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--text-primary); }
.item-size { font-size: 10px; color: var(--text-muted); margin-top: 1px; }
.item-cat { padding: 1px 6px; border-radius: 3px; font-size: 10px; color: #fff; flex-shrink: 0; opacity: 0.8; }

/* Right detail */
.mod-detail { flex: 1; overflow-y: auto; padding: 0 20px; }
.mod-detail h2 { font-size: 22px; font-weight: 600; margin-bottom: 10px; color: var(--text-primary); }
.empty-detail { display: flex; align-items: center; justify-content: center; color: var(--text-muted); }

.detail-meta { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }

.preview-section { margin-bottom: 14px; }
.preview-loading { font-size: 12px; color: var(--text-muted); }
.preview-img { max-width: 100%; max-height: 180px; border-radius: 6px; border: 1px solid var(--border-color); object-fit: contain; }
.cat-badge { padding: 3px 10px; border-radius: 4px; font-size: 13px; color: #fff; }
.ruleset-badge { background: var(--danger); color: #fff; padding: 3px 10px; border-radius: 4px; font-size: 12px; }
.size-text { color: var(--text-muted); font-size: 13px; }

.detail-table { margin-bottom: 16px; }
.detail-table .row { display: flex; padding: 6px 0; border-bottom: 1px solid var(--border-color); font-size: 13px; }
.detail-table .row:last-child { border-bottom: none; }
.detail-table .k { width: 60px; color: var(--text-muted); flex-shrink: 0; }
.detail-table .v { color: var(--text-primary); word-break: break-all; }
.detail-table .v.warn { color: var(--warning); }

.topics { display: flex; flex-wrap: wrap; gap: 4px; }
.topic-tag { background: var(--bg-secondary); padding: 2px 8px; border-radius: 4px; font-size: 11px; color: var(--text-secondary); }

.detail-actions-row { margin-bottom: 12px; }
.btn-del { padding: 6px 14px; background: rgba(255,77,79,0.1); color: var(--danger); border: 1px solid var(--danger); border-radius: 4px; cursor: pointer; font-size: 12px; }
.btn-del:hover { background: rgba(255,77,79,0.2); }
.backup-section { margin: 12px 0; font-size: 13px; }
.backup-section summary { cursor: pointer; color: var(--text-primary); margin-bottom: 6px; }
.backup-empty { color: var(--text-muted); font-size: 12px; padding: 8px 0; }
.backup-row { display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; background: var(--bg-card); border-radius: 4px; margin-bottom: 4px; }
.backup-info { display: flex; gap: 10px; align-items: center; }
.backup-ts { font-size: 12px; }
.backup-ver { font-size: 11px; background: var(--accent); color: #fff; padding: 1px 6px; border-radius: 3px; }
.backup-size { font-size: 11px; color: var(--text-muted); }
.backup-actions { display: flex; gap: 4px; }
.loading-sm { font-size: 12px; color: var(--text-muted); padding: 8px 0; }
.toast { font-size: 12px; color: var(--success); padding: 6px 0; }
.scan-error { font-size: 13px; color: var(--danger); padding: 6px 0; }

.readme-section { margin-top: 20px; }
.readme-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.readme-header h3 { font-size: 15px; margin: 0; color: var(--text-primary); }
.btn-trans { padding: 4px 12px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; }
.btn-trans:disabled { opacity: 0.5; cursor: not-allowed; }
.translated-section { margin-top: 16px; }
.translated-section h3 { font-size: 14px; margin-bottom: 8px; color: var(--text-primary); }
.readme-loading { color: var(--text-muted); font-size: 13px; }
.readme-text { background: var(--code-bg); border: 1px solid var(--code-border); border-radius: 6px; padding: 14px; font-size: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-word; color: var(--code-text); font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }

.update-btn { padding: 6px 14px; background: var(--warning); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; }
.update-btn:disabled { opacity: 0.6; cursor: not-allowed; }

.update-banner { padding: 10px 14px; border-radius: 4px; margin-bottom: 10px; font-size: 13px; background: rgba(255,107,107,0.08); color: var(--danger); display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; }
.update-banner.ok { background: rgba(82,196,26,0.08); color: var(--success); }
.update-banner.info { background: rgba(74,158,255,0.08); color: var(--accent); }
.update-actions { display: flex; align-items: center; gap: 10px; }
.btn-update-all { padding: 4px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.btn-update-all:disabled { opacity: 0.6; cursor: not-allowed; }
.update-hint { font-size: 11px; color: var(--text-muted); }

.update-list { display: flex; flex-direction: column; gap: 3px; margin-bottom: 10px; flex-shrink: 0; max-height: 150px; overflow-y: auto; }
.update-row { display: flex; align-items: center; gap: 8px; padding: 6px 10px; background: var(--bg-card); border-radius: 4px; font-size: 12px; border-left: 3px solid var(--warning); }
.u-name { font-weight: 600; min-width: 120px; color: var(--text-primary); }
.u-old { color: var(--text-muted); }
.u-arrow { color: var(--text-muted); }
.u-new { color: var(--text-secondary); }
.u-badge { margin-left: auto; padding: 1px 8px; background: var(--warning); color: #fff; border-radius: 3px; font-size: 11px; }
</style>
