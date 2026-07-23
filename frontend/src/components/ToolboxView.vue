<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { AnalyzeConflicts, ReadCrashReport, DiagnoseMods, StartDownloadWithMirror, GetDownloadList, PauseDownload, ResumeDownload, CancelDownload, RetryDownload, RemoveDownload, SelectDownloadDirectory, SaveDownloadedFile } from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { app } from '../../wailsjs/go/models'

type ConflictReport = app.ConflictReport
type DiagIssue = { mod: string; severity: string; message: string; detail: string }

// ── Conflict / Crash / Diagnose state ──
const reports = ref<ConflictReport[]>([])
const loading = ref(false)
const error = ref('')
const activeTab = ref<'conflict' | 'crash' | 'diagnose' | 'download'>('conflict')

const activeLevel = ref('override')
const activeCat = ref('building')
const catWidth = ref(160)
const diagCatWidth = ref(160)

const levelConfig: Record<string, { label: string; icon: string; color: string }> = {
  safe: { label: '安全覆盖', icon: '🟢', color: '#52c41a' },
  risk: { label: '兼容风险', icon: '🟡', color: '#faad14' },
  override: { label: '扩展互盖', icon: '🔴', color: '#ff4d4f' },
  incompatible: { label: '互斥声明', icon: '⚪', color: '#8c8c8c' },
}

const catConfig: Record<string, string> = {} // backend sends Chinese labels directly

const catOrder = ['compat','建筑','单位','单位类型','单位晋升','科技','地块资源','地形','地块改良','国家','信仰','宗教','政策','事件','任务','遗迹','难度','时代','速度','专家','城邦','全局规则','教程','胜利条件','other']

const reportsByLevel = computed(() => {
  const m: Record<string, ConflictReport[]> = {}
  for (const r of reports.value) {
    (m[r.level] ??= []).push(r)
  }
  return m
})

const availableCats = computed(() => {
  const items = reportsByLevel.value[activeLevel.value] || []
  return catOrder.filter(c => items.some(r => r.category === c))
})

const activeItems = computed(() => {
  const items = reportsByLevel.value[activeLevel.value] || []
  return items.filter(r => r.category === activeCat.value)
})

const levelTabs = ['incompatible','override','risk','safe']
const levelTotals = computed(() => {
  const m: Record<string, number> = {}
  for (const l of levelTabs) m[l] = (reportsByLevel.value[l] || []).length
  return m
})

function catTotal(level: string, cat: string) {
  return (reportsByLevel.value[level] || []).filter(r => r.category === cat).length
}

async function runAnalysis() {
  loading.value = true
  error.value = ''
  try {
    reports.value = await AnalyzeConflicts()
  } catch (e: any) {
    error.value = '分析失败: ' + e
  } finally {
    loading.value = false
  }
}

// ── Crash report ──
const crash = ref<any>(null)
const crashLoading = ref(false)

async function loadCrash() {
  crashLoading.value = true
  try { crash.value = await ReadCrashReport() }
  catch { crash.value = { found: false } }
  finally { crashLoading.value = false }
}

// ── Diagnose ──
const diagIssues = ref<DiagIssue[]>([])
const diagLoading = ref(false)
const diagDone = ref(false)
const activeDiagSeverity = ref<'error' | 'warning'>('error')
const activeDiagMod = ref('')

const diagErrors = computed(() => diagIssues.value.filter(d => d.severity === 'error'))
const diagWarnings = computed(() => diagIssues.value.filter(d => d.severity === 'warning'))

const diagModsForSeverity = computed(() => {
  const items = diagIssues.value.filter(d => d.severity === activeDiagSeverity.value)
  return [...new Set(items.map(d => d.mod))].sort()
})

const diagEntries = computed(() => {
  const items = diagIssues.value.filter(d => d.severity === activeDiagSeverity.value && d.mod === activeDiagMod.value)
  return items
})

function diagCountByMod(mod: string, sev: string) {
  return diagIssues.value.filter(d => d.mod === mod && d.severity === sev).length
}

async function runDiagnose() {
  diagDone.value = true
  diagLoading.value = true
  try { diagIssues.value = await DiagnoseMods() }
  catch (e: any) { error.value = '诊断失败: ' + e }
  finally { diagLoading.value = false }
}

function startCatDrag(e: MouseEvent, key: 'cat' | 'diag') {
  e.preventDefault()
  const w = key === 'cat' ? catWidth : diagCatWidth
  const startX = e.clientX
  const startW = w.value
  function onMove(ev: MouseEvent) {
    w.value = Math.max(80, Math.min(400, startW + (ev.clientX - startX)))
  }
  function onUp() {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// ── Custom Download ──
const customUrl = ref('')
const customFilename = ref('')
const customUrlError = ref('')
const customMsg = ref('')
const downloadBusy = ref(false)
const savePath = ref('')

const dlTasks = ref<app.DownloadTask[]>([])
type DlMeta = { id: string; filePath: string; savedTo: string }
const dlMeta = ref<DlMeta[]>([])
let unsubDlProg: (() => void) | null = null
let unsubDlDone: (() => void) | null = null

const dlDownloading = computed(() => dlTasks.value.filter(t => t.status === 'downloading' || t.status === 'queued'))
const dlPaused = computed(() => dlTasks.value.filter(t => t.status === 'paused'))
const dlCompleted = computed(() => dlTasks.value.filter(t => t.status === 'completed'))
const dlFailed = computed(() => dlTasks.value.filter(t => t.status === 'failed'))

function extractFilename(url: string): string {
  try {
    // Try to extract from query params first (Content-Disposition style)
    const qm = url.indexOf('?')
    if (qm > 0) {
      const qs = url.slice(qm + 1)
      for (const p of qs.split('&')) {
        const [k, v] = p.split('=')
        if (k === 'filename' || k === 'rscd') {
          const m = decodeURIComponent(v || '').match(/filename[=:]\s*([^;&]+)/i)
          if (m) return m[1].trim()
        }
      }
    }
    // Fallback: last path segment, strip query
    let name = url.split('/').filter(Boolean).pop() || 'download'
    name = name.split('?')[0]
    return decodeURIComponent(name)
  } catch { return 'download' }
}
function formatSize(n: number): string {
  if (!n || n <= 0) return '?'
  if (n < 1024 * 1024) return (n / 1024).toFixed(0) + ' KB'
  return (n / (1024 * 1024)).toFixed(1) + ' MB'
}

onMounted(async () => {
  try { dlTasks.value = await GetDownloadList() } catch { /* ignore */ }
  unsubDlProg = EventsOn('download:progress', (d: any) => {
    const idx = dlTasks.value.findIndex(t => t.id === d.id)
    if (idx >= 0) dlTasks.value[idx] = { ...dlTasks.value[idx], ...d }
    else dlTasks.value.push(d as app.DownloadTask)
  })
  unsubDlDone = EventsOn('download:complete', async (d: any) => {
    customMsg.value = `✅ 下载完成: ${d.filename}`
    dlMeta.value.push({ id: d.id, filePath: d.filePath, savedTo: '' })
    dlTasks.value = await GetDownloadList()
    // Auto-save to user's directory if set
    if (savePath.value && d.filePath) {
      try {
        const dest = await SaveDownloadedFile(d.filePath, savePath.value)
        if (dest) {
          const m = dlMeta.value.find(x => x.id === d.id)
          if (m) m.savedTo = dest
          customMsg.value = `✅ 已保存到: ${dest}`
        }
      } catch { /* user can save manually */ }
    }
    setTimeout(() => { if (customMsg.value?.startsWith('✅')) customMsg.value = '' }, 8000)
  })
})
onUnmounted(() => { unsubDlProg?.(); unsubDlDone?.() })

function onUrlInput() { if (!customFilename.value) customFilename.value = extractFilename(customUrl.value) }

async function pickSavePath() {
  try {
    const dir = await SelectDownloadDirectory()
    if (dir) savePath.value = dir
  } catch { /* ignore */ }
}

async function startDownload() {
  const url = customUrl.value.trim()
  if (!url) { customUrlError.value = '请输入下载链接'; return }
  if (!url.startsWith('http')) { customUrlError.value = '请输入有效的 HTTP/HTTPS 链接'; return }
  downloadBusy.value = true; customUrlError.value = ''
  const name = customFilename.value.trim() || extractFilename(url)
  try {
    await StartDownloadWithMirror(url, name, 'auto')
    dlTasks.value = await GetDownloadList()
    customMsg.value = `已添加: ${name}`
  } catch (e: any) { customUrlError.value = '启动失败: ' + e }
  finally { downloadBusy.value = false }
}

async function doPause(id: string) { await PauseDownload(id); dlTasks.value = await GetDownloadList() }
async function doResume(id: string) { await ResumeDownload(id); dlTasks.value = await GetDownloadList() }
async function doCancel(id: string) { await CancelDownload(id); dlTasks.value = await GetDownloadList() }
async function doRetry(id: string) { await RetryDownload(id); dlTasks.value = await GetDownloadList() }
async function doRemove(id: string) { await RemoveDownload(id); dlTasks.value = await GetDownloadList() }
async function doClearDone() {
  for (const t of dlCompleted.value) await RemoveDownload(t.id)
  dlTasks.value = await GetDownloadList()
}
function getSavedTo(id: string) { return dlMeta.value.find(m => m.id === id)?.savedTo || '' }
</script>

<template>
  <div class="toolbox-view view-card">
    <div class="view-header">
      <h1>🧰 工具箱</h1>
      <p class="subtitle">冲突检测 · 兼容性分析 · 崩溃报告</p>
    </div>

    <!-- Tab bar -->
    <div class="tab-bar">
      <button class="tab-btn" :class="{ active: activeTab === 'conflict' }" @click="activeTab = 'conflict'">📋 冲突检测</button>
      <button class="tab-btn" :class="{ active: activeTab === 'diagnose' }" @click="activeTab = 'diagnose'">🔬 模组诊断</button>
      <button class="tab-btn" :class="{ active: activeTab === 'crash' }" @click="activeTab = 'crash'; loadCrash()">💥 崩溃报告</button>
      <button class="tab-btn" :class="{ active: activeTab === 'download' }" @click="activeTab = 'download'">📥 自定义下载</button>
    </div>

    <!-- ══ Conflict Analysis ══ -->
    <div v-show="activeTab === 'conflict'" class="tool-section">
      <div class="section-header">
        <h2>📋 覆盖分析报告</h2>
        <button class="btn-analysis" :disabled="loading" @click="runAnalysis">
          {{ loading ? '分析中...' : '重新扫描' }}
        </button>
      </div>

      <div v-if="error" class="error-banner">{{ error }}</div>

      <div v-if="reports.length === 0 && !loading" class="empty-state">
        <p>点击「重新扫描」开始分析模组冲突</p>
      </div>

      <div v-if="loading" class="loading-state">
        <div class="spinner"></div>
      </div>

      <div v-if="reports.length > 0" class="report-layout">
        <!-- Left: level tabs -->
        <div class="level-sidebar">
          <div
            v-for="l in levelTabs"
            :key="l"
            v-show="levelTotals[l] > 0"
            class="level-tab"
            :class="{ active: activeLevel === l }"
            :style="{ borderColor: activeLevel === l ? levelConfig[l]?.color : 'transparent' }"
            @click="activeLevel = l; activeCat = availableCats[0] || 'other'"
          >
            <span class="l-icon">{{ levelConfig[l]?.icon }}</span>
            <span class="l-label">{{ levelConfig[l]?.label }}</span>
            <span class="l-count">{{ levelTotals[l] }}</span>
          </div>
        </div>

        <!-- Middle: category tabs -->
        <div class="cat-sidebar" :style="{ width: catWidth + 'px' }">
          <div
            v-for="c in availableCats"
            :key="c"
            class="cat-tab"
            :class="{ active: activeCat === c }"
            @click="activeCat = c"
          >
            <span class="c-label">{{ c }}</span>
            <span class="c-count">{{ catTotal(activeLevel, c) }}</span>
          </div>
        </div>
        <div class="drag-handle" @mousedown="e => startCatDrag(e, 'cat')"></div>

        <!-- Right: entries -->
        <div class="entries-panel">
          <div v-for="(r, idx) in activeItems" :key="idx" class="entry" :class="'level-' + r.level">
            <div class="entry-entity">{{ r.entityID }}</div>
            <div class="entry-msg">{{ r.message }}</div>
            <div class="entry-mods">
              <span class="tag a">{{ r.modA }}</span>
              <span v-if="r.modB" class="vs">vs</span>
              <span v-if="r.modB" class="tag b">{{ r.modB }}</span>
            </div>
            <div v-if="r.detail" class="entry-detail">{{ r.detail }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ══ Crash Report ══ -->
    <div v-show="activeTab === 'crash'" class="tool-section">
      <div class="section-header">
        <h2>💥 崩溃报告</h2>
        <button class="btn-analysis" :disabled="crashLoading" @click="loadCrash">
          {{ crashLoading ? '读取中...' : '刷新' }}
        </button>
      </div>

      <div v-if="crashLoading" class="loading-state"><div class="spinner"></div></div>

      <div v-else-if="!crash" class="empty-state">
        <p>点击「刷新」读取 Unciv 崩溃日志</p>
      </div>

      <div v-else-if="!crash.found" class="empty-state">
        <p>✅ 未发现崩溃记录 (lasterror.txt)</p>
      </div>

      <div v-else class="crash-content">
        <div class="crash-meta">
          <span>崩溃时间：{{ crash.lastModTime || '未知' }}</span>
          <span class="crash-path">{{ crash.filePath }}</span>
        </div>

        <div v-if="crash.hasMatch" class="crash-diagnosis">
          <div class="diag-header">🔍 自动诊断</div>
          <div class="diag-text">{{ crash.diagnosis }}</div>
        </div>

        <div v-if="crash.suggestion" class="crash-suggestion">
          <div class="sug-header">💡 建议</div>
          <div class="sug-text">{{ crash.suggestion }}</div>
        </div>

        <details class="raw-details">
          <summary>📄 原始堆栈</summary>
          <pre class="raw-stack">{{ crash.raw }}</pre>
        </details>
      </div>
    </div>

    <!-- ══ Diagnose ══ -->
    <div v-show="activeTab === 'diagnose'" class="tool-section">
      <div class="section-header">
        <h2>🔬 模组诊断</h2>
        <button class="btn-analysis" :disabled="diagLoading" @click="runDiagnose">{{ diagLoading ? '诊断中...' : '重新扫描' }}</button>
      </div>
      <div v-if="error" class="error-banner">{{ error }}</div>
      <div v-if="diagIssues.length === 0 && !diagLoading" class="empty-state"><p>点击「重新扫描」开始诊断</p></div>
      <div v-if="diagLoading" class="loading-state"><div class="spinner"></div></div>

      <div v-if="diagIssues.length === 1 && diagIssues[0].severity === 'info' && !diagLoading" class="empty-state" style="margin-top:12px">
        <p>{{ diagIssues[0].message }}</p>
      </div>

      <div v-else-if="diagIssues.length > 0" class="report-layout">
        <!-- Left: severity tabs -->
        <div class="level-sidebar">
          <div class="level-tab" :class="{ active: activeDiagSeverity === 'error' }" @click="activeDiagSeverity = 'error'">
            <span class="l-icon">🔴</span><span class="l-label">错误</span><span class="l-count">{{ diagErrors.length }}</span>
          </div>
          <div class="level-tab" :class="{ active: activeDiagSeverity === 'warning' }" @click="activeDiagSeverity = 'warning'">
            <span class="l-icon">🟡</span><span class="l-label">警告</span><span class="l-count">{{ diagWarnings.length }}</span>
          </div>
        </div>
        <!-- Middle: mod list -->
        <div class="cat-sidebar" :style="{ width: diagCatWidth + 'px' }">
          <div v-for="m in diagModsForSeverity" :key="m" class="cat-tab" :class="{ active: activeDiagMod === m }" @click="activeDiagMod = m">
            <span class="c-label">{{ m }}</span><span class="c-count">{{ diagCountByMod(m, activeDiagSeverity) }}</span>
          </div>
        </div>
        <div class="drag-handle" @mousedown="e => startCatDrag(e, 'diag')"></div>
        <!-- Right: entries -->
        <div class="entries-panel">
          <div v-for="(d, i) in diagEntries" :key="i" class="entry" :class="'level-' + (d.severity === 'error' ? 'override' : 'risk')">
            <div class="entry-entity">{{ d.mod }}</div>
            <div class="entry-msg">{{ d.message }}</div>
            <div v-if="d.detail" class="entry-detail">{{ d.detail }}</div>
          </div>
        </div>
      </div>
      <div v-else-if="diagIssues.length === 0 && diagDone && !diagLoading && !loading" class="empty-state" style="margin-top:20px"><p>✅ 所有模组通过自检，无内部问题</p></div>
    </div>

    <!-- ══ Custom Download ══ -->
    <div v-show="activeTab === 'download'" class="tool-section">
      <div class="section-header"><h2>📥 自定义下载</h2></div>

      <!-- Save path bar (PCL-style) -->
      <div class="save-path-bar">
        <span class="save-path-icon">📁</span>
        <span class="save-path-text">{{ savePath || '未设置保存路径（将下载到临时目录）' }}</span>
        <button class="btn-browse" @click="pickSavePath" title="选择保存目录">…</button>
      </div>

      <!-- URL input row -->
      <div class="dl-bar">
        <input v-model="customUrl" placeholder="https://example.com/file.zip" class="dl-input" @input="onUrlInput" @keyup.enter="startDownload" />
        <input v-model="customFilename" placeholder="文件名" class="dl-name" />
        <button class="btn-analysis" :disabled="downloadBusy || !customUrl.trim()" @click="startDownload">{{ downloadBusy ? '添加中…' : '下载' }}</button>
      </div>

      <!-- Notifications -->
      <div v-if="customUrlError" class="error-banner">{{ customUrlError }}</div>
      <div v-if="customMsg" class="dl-toast">{{ customMsg }}</div>

      <!-- Task list -->
      <div v-if="dlTasks.length === 0" class="empty-state"><p>输入文件链接并点击「下载」开始</p></div>
      <div v-else class="dl-queue">

        <!-- Downloading -->
        <div v-if="dlDownloading.length" class="dl-section">
          <h3>下载中 ({{ dlDownloading.length }})</h3>
          <div v-for="t in dlDownloading" :key="t.id" class="dl-task" :class="t.status === 'queued' ? 'queued' : 'active'">
            <div class="dl-info"><span class="dl-name">{{ t.filename }}</span><span class="dl-meta">{{ t.status === 'queued' ? '⏳ 排队中' : formatSize(t.downloaded)+' / '+formatSize(t.totalSize)+' · '+(t.speed||'…')+' · '+(t.percent||0).toFixed(0)+'%' }}</span></div>
            <div class="progress-bar"><div class="fill" :class="{ paused: t.status === 'queued' }" :style="{ width: Math.max(t.percent||0, 2)+'%' }"></div></div>
            <div class="dl-actions"><button class="btn-sm" v-if="t.status==='downloading'" @click="doPause(t.id)">暂停</button><button class="btn-sm danger" @click="doCancel(t.id)">取消</button></div>
          </div>
        </div>

        <!-- Paused -->
        <div v-if="dlPaused.length" class="dl-section">
          <h3>已暂停 ({{ dlPaused.length }})</h3>
          <div v-for="t in dlPaused" :key="t.id" class="dl-task">
            <div class="dl-info"><span class="dl-name">{{ t.filename }}</span><span class="dl-meta">{{ formatSize(t.downloaded) }} / {{ formatSize(t.totalSize) }}</span></div>
            <div class="progress-bar"><div class="fill paused" :style="{ width: Math.max(t.percent||0, 2)+'%' }"></div></div>
            <div class="dl-actions"><button class="btn-sm" @click="doResume(t.id)">继续</button><button class="btn-sm danger" @click="doCancel(t.id)">取消</button></div>
          </div>
        </div>

        <!-- Completed -->
        <div v-if="dlCompleted.length" class="dl-section">
          <div class="dl-section-hdr"><h3>已完成</h3><button class="btn-sm danger" @click="doClearDone">清空</button></div>
          <div v-for="t in dlCompleted" :key="t.id" class="dl-task done">
            <div class="dl-done-info">
              <span class="dl-name">{{ t.filename }}</span>
              <span v-if="getSavedTo(t.id)" class="dl-saved-path">📁 {{ getSavedTo(t.id) }}</span>
              <span v-else class="dl-saved-temp">📁 临时目录</span>
            </div>
          </div>
        </div>

        <!-- Failed -->
        <div v-if="dlFailed.length" class="dl-section">
          <h3>失败 ({{ dlFailed.length }})</h3>
          <div v-for="t in dlFailed" :key="t.id" class="dl-task failed">
            <div class="dl-info"><span class="dl-name">{{ t.filename }}</span><span class="dl-err">{{ t.error }}</span></div>
            <div class="dl-actions"><button class="btn-sm" @click="doRetry(t.id)">重试</button><button class="btn-sm danger" @click="doRemove(t.id)">删除</button></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbox-view { height: 100%; max-width: 1100px; }

/* Tab bar */
.tab-bar { display: flex; gap: 4px; margin-bottom: 16px; }
.tab-btn { padding: 8px 20px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 6px; color: var(--text-secondary); cursor: pointer; font-size: 14px; }
.tab-btn:hover { color: var(--text-primary); }
.tab-btn.active { background: var(--accent); color: #fff; border-color: var(--accent); }

/* Crash report */
.crash-content { padding: 4px 0; }
.crash-meta { display: flex; gap: 16px; font-size: 15px; color: var(--text-secondary); margin-bottom: 16px; flex-wrap: wrap; }
.crash-path { font-family: monospace; font-size: 12px; opacity: 0.7; }
.crash-diagnosis { background: rgba(255,107,107,0.08); border-left: 3px solid var(--danger); padding: 12px; border-radius: 4px; margin-bottom: 12px; }
.diag-header { font-size: 13px; font-weight: 600; color: var(--danger); margin-bottom: 4px; }
.diag-text { font-size: 13px; color: var(--text-primary); }
.crash-suggestion { background: rgba(74,158,255,0.08); border-left: 3px solid var(--accent); padding: 12px; border-radius: 4px; margin-bottom: 12px; }
.sug-header { font-size: 13px; font-weight: 600; color: var(--accent); margin-bottom: 4px; }
.sug-text { font-size: 13px; color: var(--text-primary); }
.raw-details { margin-top: 8px; }
.raw-details summary { font-size: 13px; color: var(--text-muted); cursor: pointer; }
.raw-stack { background: var(--code-bg); border: 1px solid var(--code-border); border-radius: 4px; padding: 12px; font-size: 11px; line-height: 1.5; max-height: 300px; overflow: auto; white-space: pre-wrap; word-break: break-all; margin-top: 8px; color: var(--code-text); font-family: 'Cascadia Code', 'Fira Code', monospace; }

.diag-list { display: flex; flex-direction: column; gap: 6px; }
.diag-item { padding: 10px 12px; border-radius: 4px; border-left: 3px solid #555; background: var(--bg-card); }
.diag-item.error { border-left-color: var(--danger); background: rgba(255,77,79,0.06); }
.diag-item.warning { border-left-color: var(--warning); background: rgba(250,173,20,0.06); }
.diag-mod { font-size: 12px; font-weight: 600; color: var(--accent); display: block; margin-bottom: 2px; }
.diag-msg { font-size: 13px; color: var(--text-primary); }
.diag-detail { font-size: 11px; color: var(--text-muted); display: block; margin-top: 2px; }
.view-header { margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 4px; color: var(--text-primary); }
.subtitle { color: var(--text-muted); font-size: 14px; }
.tool-section { background: var(--bg-card); border-radius: 8px; padding: 16px; box-shadow: var(--card-shadow); }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.section-header h2 { font-size: 18px; font-weight: 600; }
.btn-analysis { padding: 8px 20px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
.btn-analysis:disabled { opacity: 0.6; cursor: not-allowed; }
.error-banner { background: rgba(255,107,107,0.1); color: var(--danger); padding: 12px; border-radius: 4px; margin-bottom: 12px; }
.empty-state, .loading-state { text-align: center; padding: 40px; color: var(--text-secondary); }
.spinner { width: 32px; height: 32px; border: 3px solid var(--border-color); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 12px; }
@keyframes spin { to { transform: rotate(360deg); } }

.report-layout { display: flex; gap: 0; height: calc(100vh - 200px); min-height: 400px; }

.level-sidebar { width: 140px; flex-shrink: 0; background: var(--bg-secondary); border-radius: 6px 0 0 6px; overflow-y: auto; }
.level-tab { display: flex; align-items: center; gap: 6px; padding: 10px 12px; cursor: pointer; border-left: 3px solid transparent; font-size: 13px; }
.level-tab:hover { background: var(--border-color); }
.level-tab.active { background: var(--sidebar-active); }
.l-icon { font-size: 14px; }
.l-label { flex: 1; }
.l-count { color: var(--text-muted); font-size: 12px; }

.cat-sidebar { width: 100px; flex-shrink: 0; background: var(--bg-secondary); border-left: 1px solid var(--border-color); overflow-y: auto; }
.cat-tab { display: flex; align-items: center; justify-content: space-between; padding: 8px 10px; cursor: pointer; font-size: 12px; color: var(--text-secondary); }
.cat-tab:hover { background: var(--border-color); color: var(--text-primary); }
.cat-tab.active { background: var(--sidebar-active); color: var(--text-primary); font-weight: 600; }
.c-count { color: var(--text-muted); font-size: 11px; }

.drag-handle { width: 4px; cursor: col-resize; background: transparent; flex-shrink: 0; }
.drag-handle:hover { background: var(--accent); }

.entries-panel { flex: 1; overflow-y: auto; padding: 8px; background: var(--bg-primary); border-radius: 0 6px 6px 0; }
.entry { padding: 8px; border-radius: 4px; margin-bottom: 4px; border-left: 3px solid var(--border-color); }
.entry:last-child { margin-bottom: 0; }
.entry.level-safe { border-left-color: var(--success); background: rgba(82,196,26,0.08); }
.entry.level-risk { border-left-color: var(--warning); background: rgba(250,173,20,0.08); }
.entry.level-override { border-left-color: var(--danger); background: rgba(255,77,79,0.08); }
.entry.level-incompatible { border-left-color: #8c8c8c; background: rgba(140,140,140,0.08); }
.entry-entity { font-family: monospace; font-size: 13px; color: var(--text-primary); margin-bottom: 2px; font-weight: 600; }
.entry-msg { font-size: 14px; margin-bottom: 3px; color: var(--text-primary); }
.entry-mods { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.tag { padding: 2px 8px; border-radius: 3px; font-size: 12px; font-weight: 600; }
.tag.a { background: var(--accent); color: #fff; }
.tag.b { background: #b44eff; color: #fff; }
.vs { color: var(--text-secondary); font-size: 12px; }
.entry-detail { margin-top: 3px; font-size: 13px; color: var(--text-primary); padding: 4px 8px; background: var(--bg-card); border-radius: 3px; }

/* ── Custom Download ── */
.save-path-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 10px; padding: 8px 10px; background: var(--bg-card); border-radius: 4px; font-size: 13px; border: 1px solid var(--border-color); }
.save-path-icon { font-size: 16px; }
.save-path-text { flex: 1; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn-browse { padding: 2px 12px; background: var(--bg-hover); border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; font-size: 16px; font-weight: 700; color: var(--text-primary); line-height: 1.4; }
.btn-browse:hover { background: var(--accent); color: #fff; border-color: var(--accent); }
.dl-bar { display: flex; gap: 8px; margin-bottom: 6px; flex-wrap: wrap; }
.dl-input { flex: 1; padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 14px; min-width: 200px; }
.dl-input:focus { border-color: var(--accent); }
.dl-name { width: 150px; padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 14px; }
.dl-name:focus { border-color: var(--accent); }
.dl-toast { background: rgba(78,205,196,0.1); color: #4ecdc4; padding: 8px 12px; border-radius: 4px; margin-bottom: 8px; font-size: 13px; }
.dl-queue { margin-top: 10px; }
.dl-section { margin-bottom: 12px; }
.dl-section h3 { font-size: 13px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 6px; }
.dl-section-hdr { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.dl-task { background: var(--bg-card); border-radius: 6px; padding: 10px; margin-bottom: 4px; box-shadow: var(--card-shadow); }
.dl-task.active { border-left: 3px solid var(--accent); }
.dl-task.queued { border-left: 3px solid var(--warning); opacity: 0.7; }
.dl-task.done { display: flex; align-items: center; }
.dl-task.failed { border-left: 3px solid var(--danger); }
.dl-info { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.dl-name { font-size: 14px; font-weight: 500; }
.dl-meta { font-size: 12px; color: var(--text-secondary); }
.dl-err { color: var(--danger); font-size: 12px; }
.dl-actions { display: flex; gap: 6px; }
.dl-done-info { display: flex; flex-direction: column; gap: 2px; flex: 1; }
.dl-saved-path { font-size: 12px; color: var(--success); }
.dl-saved-temp { font-size: 12px; color: var(--text-muted); }
.progress-bar { height: 4px; background: var(--bg-secondary); border-radius: 2px; margin-bottom: 6px; overflow: hidden; }
.fill { height: 100%; background: var(--accent); border-radius: 2px; transition: width 0.3s; min-width: 2%; }
.fill.paused { background: var(--warning); }
</style>
