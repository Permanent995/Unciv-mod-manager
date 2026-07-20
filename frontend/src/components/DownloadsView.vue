<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  StartDownloadWithMirror, PauseDownload, ResumeDownload, CancelDownload,
  GetDownloadList, ExtractMod, CleanupTempFile, GetAppConfig, GetMirrorHealth,
  RemoveDownload, RetryDownload, ImportFile, BackupMod,
} from '../../wailsjs/go/app/App'
import { FetchReleases } from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

type DownloadTask = {
  id: string; url: string; filename: string; status: string
  totalSize: number; downloaded: number; percent: number
  speed: string; error?: string
}
type Release = {
  tag_name: string; name: string; published_at: string
  zipball_url: string; assets: { name: string; size: number; browser_download_url: string }[]
}
type Mirror = { url: string; latency: number; label: string }

const tasks = ref<DownloadTask[]>([])
const repoUrl = ref('')
const customName = ref('')
const releases = ref<Release[]>([])
const loadingReleases = ref(false)
const releaseError = ref('')
const busy = ref(false)
const msg = ref('')
const mirrors = ref<Mirror[]>([])
const selectedMirror = ref('direct')
const testingMirrors = ref(false)
const mirrorMode = ref('auto')
let unsubProgress: (() => void) | null = null
let unsubComplete: (() => void) | null = null

onMounted(async () => {
  const cfg = await GetAppConfig()
  mirrorMode.value = cfg.mirrorMode || 'auto'
  if (mirrorMode.value === 'manual' && cfg.selectedMirror) {
    selectedMirror.value = cfg.selectedMirror
  }
  tasks.value = await GetDownloadList()
  testMirrors()
  unsubProgress = EventsOn('download:progress', (data: any) => {
    const idx = tasks.value.findIndex(t => t.id === data.id)
    if (idx >= 0) tasks.value[idx] = { ...tasks.value[idx], ...data }
    else tasks.value.push(data as DownloadTask)
  })
  unsubComplete = EventsOn('download:complete', async (data: any) => {
    msg.value = `下载完成: ${data.filename}，正在解包...`
    try {
      const cfg = await GetAppConfig()
      await ExtractMod(data.filePath, (cfg.uncivPath || '') + '\\mods')
      await CleanupTempFile(data.filePath)
      msg.value = `模组 ${data.filename} 已安装`
    } catch (e: any) { msg.value = '解包失败: ' + e }
  })
})
onUnmounted(() => { unsubProgress?.(); unsubComplete?.() })

async function testMirrors() {
  testingMirrors.value = true
  try {
    const health = await GetMirrorHealth()
    const list: Mirror[] = [{ url: 'direct', latency: 0, label: '直连' }]
    for (const m of health) {
      if (m.alive) list.push({ url: m.url, latency: m.latency, label: m.label })
    }
    list.sort((a, b) => a.url === 'direct' ? 1 : b.url === 'direct' ? -1 : (a.latency || 9999) - (b.latency || 9999))
    mirrors.value = list
    const fastest = list.find(m => m.url !== 'direct' && m.latency > 0)
    selectedMirror.value = fastest ? fastest.url : 'direct'
  } catch { mirrors.value = [{ url: 'direct', latency: 0, label: '直连' }] }
  finally { testingMirrors.value = false }
}

const ghAuthor = computed(() => { const m = repoUrl.value.match(/github\.com\/([^\/]+)\//); return m ? m[1] : '' })
function buildArchiveUrl(): string {
  const u = repoUrl.value.replace(/^https?:\/\//, '').replace(/^github\.com\//, '').replace(/\/$/, '')
  return `https://github.com/${u}/archive/refs/`
}

async function downloadLatest() {
  if (!repoUrl.value) return
  busy.value = true; releaseError.value = ''
  try {
    const rels = await FetchReleases(repoUrl.value)
    if (rels.length === 0) { releaseError.value = '该仓库没有 Release'; return }
    const latest = rels[0]
    let dlUrl = ''; let fname = latest.tag_name + '.zip'
    for (const a of latest.assets || []) { if (a.name.toLowerCase().endsWith('.zip')) { dlUrl = a.browser_download_url; fname = a.name; break } }
    if (!dlUrl) dlUrl = buildArchiveUrl() + 'tags/' + latest.tag_name + '.zip'
    if (customName.value) fname = customName.value.endsWith('.zip') ? customName.value : customName.value + '.zip'
    await StartDownloadWithMirror(dlUrl, fname, selectedMirror.value)
    tasks.value = await GetDownloadList(); repoUrl.value = ''; customName.value = ''
  } catch (e: any) { releaseError.value = '下载失败: ' + e } finally { busy.value = false }
}

async function browseReleases() {
  if (!repoUrl.value) return
  loadingReleases.value = true; releaseError.value = ''; releases.value = []
  try { releases.value = await FetchReleases(repoUrl.value) }
  catch (e: any) { releaseError.value = e } finally { loadingReleases.value = false }
}

async function downloadRelease(rel: Release) {
  busy.value = true
  let dlUrl = buildArchiveUrl() + 'tags/' + rel.tag_name + '.zip'
  let fname = rel.tag_name + '.zip'
  for (const a of rel.assets || []) { if (a.name.toLowerCase().endsWith('.zip')) { dlUrl = a.browser_download_url; fname = a.name; break } }
  if (customName.value) fname = customName.value.endsWith('.zip') ? customName.value : customName.value + '.zip'
  try {
    await StartDownloadWithMirror(dlUrl, fname, selectedMirror.value)
    tasks.value = await GetDownloadList(); releases.value = []; repoUrl.value = ''; customName.value = ''
  } catch (e: any) { msg.value = '下载失败: ' + e } finally { busy.value = false }
}

async function downloadDirect() {
  if (!repoUrl.value) return; busy.value = true
  try { await StartDownloadWithMirror(repoUrl.value, repoUrl.value.split('/').pop() || 'mod.zip', selectedMirror.value); tasks.value = await GetDownloadList(); repoUrl.value = '' }
  catch (e: any) { msg.value = '下载失败: ' + e } finally { busy.value = false }
}

async function doPause(id: string) { await PauseDownload(id); tasks.value = await GetDownloadList() }
async function doResume(id: string) { await ResumeDownload(id); tasks.value = await GetDownloadList() }
async function doCancel(id: string) { await CancelDownload(id); tasks.value = await GetDownloadList() }
async function doRetry(id: string) { await RetryDownload(id); tasks.value = await GetDownloadList() }
async function doRemove(id: string) { await RemoveDownload(id); tasks.value = await GetDownloadList() }
async function doClearCompleted() {
  for (const t of completed()) await RemoveDownload(t.id)
  tasks.value = await GetDownloadList()
}

const fileInput = ref<HTMLInputElement | null>(null)
function pickLocalFile() { fileInput.value?.click() }
async function onLocalFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    // Copy via Wails — the path is available through the file object
    const path = (file as any).path || file.name
    if (path) {
      msg.value = '正在导入: ' + file.name
      await ImportFile(path)
      msg.value = file.name + ' 已导入'
    }
  } catch (err: any) { msg.value = '导入失败: ' + err }
  input.value = '' // reset so same file can be re-selected
}

const downloading = () => tasks.value.filter(t => t.status === 'downloading' || t.status === 'queued')
const paused = () => tasks.value.filter(t => t.status === 'paused')
const completed = () => tasks.value.filter(t => t.status === 'completed')
const failed = () => tasks.value.filter(t => t.status === 'failed')

function mirrorLabel(m: Mirror): string { return m.url === 'direct' ? '直连' : `${m.label} (${m.latency > 0 ? m.latency + 'ms' : '超时'})` }
function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '?'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="downloads-view">
    <div class="view-header"><h1>⬇ 下载</h1></div>

    <div class="mirror-bar">
      <span class="mirror-label">线路：</span>
      <span v-if="testingMirrors" class="mirror-testing">⏳ 测速中...</span>
      <span v-else-if="mirrorMode === 'auto'" class="mirror-auto">自动（故障切换）</span>
      <select v-else-if="mirrorMode !== 'auto'" v-model="selectedMirror" class="mirror-select"><option v-for="m in mirrors" :key="m.url" :value="m.url">{{ mirrorLabel(m) }}</option></select>
      <button class="btn-retest" @click="testMirrors" :disabled="testingMirrors">🔄</button>
    </div>

    <div class="add-bar">
      <input v-model="repoUrl" placeholder="GitHub 仓库地址" @keyup.enter="downloadLatest" />
      <input v-model="customName" placeholder="保存名称（可选）" class="name-input" />
      <button class="btn-go" :disabled="busy || loadingReleases || !repoUrl" @click="downloadLatest">{{ busy ? '下载中...' : '下载最新' }}</button>
      <button class="btn-go outline" :disabled="loadingReleases || !repoUrl" @click="browseReleases">{{ loadingReleases ? '...' : '版本' }}</button>
      <button class="btn-link" :disabled="busy || !repoUrl" @click="downloadDirect">直链</button>
      <button class="btn-local" @click="pickLocalFile">📁 本地</button>
      <input ref="fileInput" type="file" accept=".zip,.civ5map,.map" style="display:none" @change="onLocalFile" />
    </div>

    <div v-if="ghAuthor" class="author-hint">👤 作者: {{ ghAuthor }}</div>
    <div v-if="msg" class="toast">{{ msg }}</div>
    <div v-if="releaseError" class="error-toast">{{ releaseError }}</div>

    <div v-if="releases.length > 0" class="release-section">
      <h2>选择版本（共 {{ releases.length }} 个）</h2>
      <div v-for="(r, idx) in releases" :key="r.tag_name" class="release-row" @click="downloadRelease(r)">
        <div class="rel-info"><span class="rel-tag">{{ r.tag_name }}</span><span v-if="idx === 0" class="latest-badge">最新</span><span v-if="r.name && r.name !== r.tag_name" class="rel-name">{{ r.name }}</span></div>
        <span class="rel-date">{{ (r.published_at || '').slice(0, 10) }}</span>
      </div>
    </div>

    <div v-if="tasks.length > 0" class="queue">
      <div v-if="downloading().length" class="section"><h2>下载中</h2>
        <div v-for="t in downloading()" :key="t.id" class="task-row" :class="t.status === 'queued' ? 'queued' : 'active'">
          <div class="task-info"><span class="fname">{{ t.filename }}</span><span class="meta">{{ t.status === 'queued' ? '⏳ 排队中' : formatSize(t.downloaded) + ' / ' + formatSize(t.totalSize) + ' · ' + (t.speed||'...') + ' · ' + (t.percent||0).toFixed(0) + '%' }}</span></div>
          <div class="progress-bar"><div class="fill" :class="{ paused: t.status === 'queued' }" :style="{ width: Math.max(t.percent || 0, 2) + '%' }"></div></div>
          <div class="actions"><button class="btn-sm" @click="doPause(t.id)" v-if="t.status === 'downloading'">暂停</button><button class="btn-sm danger" @click="doCancel(t.id)">取消</button></div>
        </div>
      </div>
      <div v-if="paused().length" class="section"><h2>已暂停</h2>
        <div v-for="t in paused()" :key="t.id" class="task-row">
          <div class="task-info"><span class="fname">{{ t.filename }}</span><span class="meta">{{ formatSize(t.downloaded) }} / {{ formatSize(t.totalSize) }}</span></div>
          <div class="progress-bar"><div class="fill paused" :style="{ width: Math.max(t.percent || 0, 2) + '%' }"></div></div>
          <div class="actions"><button class="btn-sm" @click="doResume(t.id)">继续</button><button class="btn-sm danger" @click="doCancel(t.id)">取消</button></div>
        </div>
      </div>
      <div v-if="completed().length" class="section">
        <div class="section-hdr"><h2>已完成</h2><button class="btn-sm danger" @click="doClearCompleted">清空已完成</button></div>
        <div v-for="t in completed()" :key="t.id" class="task-row done"><span class="fname">{{ t.filename }}</span><div><span class="done-badge">✅ 已安装</span><button class="btn-sm" style="margin-left:8px" @click="doRemove(t.id)">✕</button></div></div>
      </div>
      <div v-if="failed().length" class="section"><h2>失败</h2>
        <div v-for="t in failed()" :key="t.id" class="task-row failed">
          <div class="task-info"><span class="fname">{{ t.filename }}</span><span class="err-msg">{{ t.error }}</span></div>
          <div class="actions"><button class="btn-sm" @click="doRetry(t.id)">重试</button><button class="btn-sm danger" @click="doRemove(t.id)">删除</button></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.downloads-view { height: 100%; max-width: 800px; }
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 12px; }
.mirror-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 10px; font-size: 13px; }
.mirror-label { color: var(--text-muted); }
.mirror-auto { color: var(--success); font-weight: 600; font-size: 13px; }
.mirror-testing { color: var(--warning); }
.mirror-select { padding: 4px 8px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; min-width: 200px; }
.btn-retest { padding: 4px 6px; background: transparent; border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-muted); cursor: pointer; font-size: 12px; }
.add-bar { display: flex; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.add-bar input { flex: 1; padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 14px; }
.name-input { max-width: 160px; }
.btn-go { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; white-space: nowrap; }
.btn-go:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-go.outline { background: transparent; border: 1px solid var(--accent); color: var(--accent); }
.btn-link { padding: 8px 12px; background: var(--border-color); color: var(--text-secondary); border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-local { padding: 8px 12px; background: var(--bg-card); color: var(--text-secondary); border: 1px dashed var(--border-color); border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-local:hover { border-color: var(--accent); color: var(--accent); }
.toast { background: rgba(78,205,196,0.1); color: #4ecdc4; padding: 10px 14px; border-radius: 4px; margin-bottom: 10px; font-size: 13px; }
.error-toast { background: rgba(255,107,107,0.1); color: var(--danger); padding: 10px 14px; border-radius: 4px; margin-bottom: 10px; font-size: 13px; }
.author-hint { font-size: 12px; color: var(--text-muted); margin-bottom: 10px; }
.release-section { margin-bottom: 16px; }
.release-section h2 { font-size: 14px; color: var(--text-muted); margin-bottom: 8px; }
.release-row { display: flex; justify-content: space-between; align-items: center; padding: 10px 14px; background: var(--bg-card); border-radius: 6px; margin-bottom: 4px; cursor: pointer; box-shadow: var(--card-shadow); }
.release-row:hover { background: var(--sidebar-active); }
.rel-info { display: flex; gap: 10px; align-items: baseline; }
.rel-tag { font-family: monospace; font-size: 13px; font-weight: 600; background: var(--accent); color: #fff; padding: 2px 8px; border-radius: 3px; }
.latest-badge { font-size: 11px; background: var(--success); color: #fff; padding: 2px 6px; border-radius: 3px; }
.rel-name { font-size: 13px; color: var(--text-secondary); }
.rel-date { font-size: 12px; color: var(--text-muted); }
.queue { margin-top: 20px; }
.section { margin-bottom: 16px; }
.section h2 { font-size: 14px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px; }
.section-hdr { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.task-row { background: var(--bg-card); border-radius: 6px; padding: 12px; margin-bottom: 6px; box-shadow: var(--card-shadow); }
.task-row.active { border-left: 3px solid var(--accent); }
.task-row.queued { border-left: 3px solid var(--warning); opacity: 0.7; }
.task-row.done { display: flex; justify-content: space-between; align-items: center; }
.task-row.failed { border-left: 3px solid var(--danger); }
.task-info { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.fname { font-size: 14px; font-weight: 500; }
.meta { font-size: 12px; color: var(--text-secondary); }
.progress-bar { height: 4px; background: var(--bg-secondary); border-radius: 2px; margin-bottom: 6px; overflow: hidden; }
.fill { height: 100%; background: var(--accent); border-radius: 2px; transition: width 0.3s; min-width: 2%; }
.fill.paused { background: var(--warning); }
.actions { display: flex; gap: 6px; }
.btn-sm { padding: 3px 10px; background: var(--border-color); color: var(--text-secondary); border: none; border-radius: 3px; cursor: pointer; font-size: 12px; }
.btn-sm:hover { background: var(--text-muted); color: var(--text-primary); }
.btn-sm.danger { color: var(--danger); }
.done-badge { color: var(--success); font-size: 13px; }
.err-msg { color: var(--danger); font-size: 12px; }
</style>
