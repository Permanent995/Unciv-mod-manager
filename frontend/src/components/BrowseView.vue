<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { SearchOnlineMods, FetchReadme, FetchReleases, StartDownloadWithMirror, GetAppConfig, ExtractMod, CleanupTempFile } from '../../wailsjs/go/app/App'
import { TranslateText } from '../../wailsjs/go/app/App'
import { marked } from 'marked'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

type OnlineMod = {
  name: string; owner: string; repo: string; description: string
  stars: number; updatedAt: string; topics: string[]; htmlUrl: string
}

type Release = {
  tag_name: string; name: string; published_at: string
  zipball_url: string; assets: { name: string; size: number; browser_download_url: string }[]
}

const browseQuery = ref('')
const onlineMods = ref<OnlineMod[]>([])
const browsing = ref(false)
const browseErr = ref('')
const browseCat = ref('all')
const sortMode = ref<'stars' | 'name' | 'updated'>('stars')
const loadedMods = ref(false)

const detail = ref<OnlineMod | null>(null)
const detailReadme = ref('')
const renderedReadme = computed(() => detailReadme.value ? marked(detailReadme.value) : '')
const detailLoading = ref(false)
const detailTranslating = ref(false)
const detailTranslated = ref('')

// Release state
const detailReleases = ref<Release[]>([])
const loadingReleases = ref(false)
const releasesError = ref('')
const downloadingRelease = ref('')
const downloadingLatest = ref(false)

const catLabels: Record<string, string> = {
  all: '全部',
  rulesets: '规则集', expansions: '扩展', graphics: '图形',
  audio: '音频', maps: '地图', fun: '趣味', modsofmods: '子模组', other: '其他',
}

const browseCats = computed(() => {
  const counts: Record<string, number> = { all: onlineMods.value.length }
  for (const m of onlineMods.value) {
    let matched = false
    for (const t of m.topics || []) {
      const cat = t.replace('unciv-mod-', '')
      if (catLabels[cat]) { counts[cat] = (counts[cat] || 0) + 1; matched = true }
    }
    if (!matched) counts['other'] = (counts['other'] || 0) + 1
  }
  return Object.entries(counts).map(([k, v]) => ({ key: k, label: catLabels[k] || k, count: v }))
})

const filteredMods = computed(() => {
  let list = onlineMods.value
  if (browseCat.value !== 'all') {
    list = list.filter(m => {
      for (const t of m.topics || []) {
        if (t.replace('unciv-mod-', '') === browseCat.value) return true
      }
      return browseCat.value === 'other' && !(m.topics || []).some(t => catLabels[t.replace('unciv-mod-', '')])
    })
  }
  const sorted = [...list]
  if (sortMode.value === 'stars') sorted.sort((a, b) => b.stars - a.stars)
  else if (sortMode.value === 'name') sorted.sort((a, b) => (a.repo || '').localeCompare(b.repo || ''))
  else if (sortMode.value === 'updated') sorted.sort((a, b) => (b.updatedAt || '').localeCompare(a.updatedAt || ''))
  return sorted
})

// Track BrowseView-initiated downloads so we only auto-extract those
const browseDLIds = new Set<string>()

onMounted(() => {
  EventsOn('download:complete', async (payload: any) => {
    if (!browseDLIds.has(payload.id)) return
    browseDLIds.delete(payload.id)
    const fn = (payload.filename || '').toLowerCase()
    if (!fn.endsWith('.zip')) return
    try {
      const cfg = await GetAppConfig()
      if (!cfg.uncivPath) return
      const modsDir = cfg.uncivPath + '/mods'
      await ExtractMod(payload.filePath, modsDir)
      CleanupTempFile(payload.filePath)
      browseErr.value = `已安装模组: ${fn}`
    } catch (e: any) {
      browseErr.value = '安装失败: ' + e
    }
  })
})

onUnmounted(() => {
  EventsOff('download:complete')
})

async function doBrowse() {
  browsing.value = true; browseErr.value = ''; onlineMods.value = []; detail.value = null; loadedMods.value = true
  try { onlineMods.value = await SearchOnlineMods(browseQuery.value || '') }
  catch (e: any) { browseErr.value = '搜索失败: ' + e }
  finally { browsing.value = false }
}

async function openOnlineMod(m: OnlineMod) {
  detail.value = m
  detailReadme.value = ''
  detailTranslated.value = ''
  detailReleases.value = []
  releasesError.value = ''
}

async function loadReadme() {
  if (!detail.value) return
  detailLoading.value = true
  detailReadme.value = ''
  try { detailReadme.value = await FetchReadme(detail.value.owner, detail.value.repo) }
  catch { detailReadme.value = '无法加载 README' }
  finally { detailLoading.value = false }
}

async function loadReleases() {
  if (!detail.value) return
  loadingReleases.value = true
  releasesError.value = ''
  detailReleases.value = []
  try {
    detailReleases.value = await FetchReleases(detail.value.htmlUrl)
    if (detailReleases.value.length === 0) {
      releasesError.value = '该仓库没有发布任何 Release（版本）'
    }
  } catch (e: any) { releasesError.value = '获取版本失败: ' + e }
  finally { loadingReleases.value = false }
}

async function downloadRelease(rel: Release) {
  if (!detail.value) return
  downloadingRelease.value = rel.tag_name
  try {
    const cfg = await GetAppConfig()
    let dlUrl = ''
    let fname = rel.tag_name + '.zip'
    for (const a of rel.assets || []) {
      if (a.name.toLowerCase().endsWith('.zip')) {
        dlUrl = a.browser_download_url
        fname = a.name
        break
      }
    }
    if (!dlUrl) {
      dlUrl = `https://github.com/${detail.value.owner}/${detail.value.repo}/archive/refs/tags/${rel.tag_name}.zip`
    }
    const mirror = cfg.mirrorMode === 'auto' ? 'auto' : (cfg.selectedMirror || 'auto')
    const taskId = await StartDownloadWithMirror(dlUrl, fname, mirror)
    browseDLIds.add(taskId)
    browseErr.value = `已添加下载: ${fname}`
  } catch (e: any) { browseErr.value = '下载失败: ' + e }
  finally { downloadingRelease.value = '' }
}

// Download default branch (main.zip) for repos without releases
async function downloadLatest() {
  if (!detail.value) return
  downloadingLatest.value = true
  try {
    const cfg = await GetAppConfig()
    const dlUrl = `https://github.com/${detail.value.owner}/${detail.value.repo}/archive/refs/heads/main.zip`
    const fname = `${detail.value.repo}-main.zip`
    const mirror = cfg.mirrorMode === 'auto' ? 'auto' : (cfg.selectedMirror || 'auto')
    const taskId = await StartDownloadWithMirror(dlUrl, fname, mirror)
    browseDLIds.add(taskId)
    browseErr.value = `已添加下载: ${fname}`
  } catch (e: any) { browseErr.value = '下载失败: ' + e }
  finally { downloadingLatest.value = false }
}

async function translateDetail() {
  if (!detailReadme.value) return
  detailTranslating.value = true
  detailTranslated.value = ''
  try { detailTranslated.value = await TranslateText(detailReadme.value) }
  catch (e: any) { detailTranslated.value = '翻译失败: ' + e }
  finally { detailTranslating.value = false }
}

function closeDetail() { detail.value = null }

function formatStars(n: number): string {
  return n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n)
}

function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return '?'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="browse-view view-card">
    <div class="view-header"><h1>🔍 模组发现</h1></div>

    <div class="add-bar">
      <input v-model="browseQuery" placeholder="搜索模组（留空为热门排序）" @keyup.enter="doBrowse" />
      <button class="btn-go" :disabled="browsing" @click="doBrowse">{{ browsing ? '搜索中...' : '搜索' }}</button>
    </div>
    <div v-if="onlineMods.length > 0" class="sort-bar">
      <span class="sort-label">排序：</span>
      <button class="sort-btn" :class="{ active: sortMode === 'stars' }" @click="sortMode = 'stars'">⭐ 热门</button>
      <button class="sort-btn" :class="{ active: sortMode === 'name' }" @click="sortMode = 'name'">A-Z</button>
      <button class="sort-btn" :class="{ active: sortMode === 'updated' }" @click="sortMode = 'updated'">最近更新</button>
      <span class="sort-total">{{ filteredMods.length }} / {{ onlineMods.length }} 个模组</span>
    </div>
    <div v-if="browseErr" class="error-toast">{{ browseErr }}</div>
    <div v-if="browsing" class="loading">加载中...</div>

    <div v-else-if="onlineMods.length > 0" class="browse-body">
      <div class="browse-sidebar">
        <div v-for="c in browseCats" :key="c.key"
          class="bcat-item" :class="{ active: browseCat === c.key }"
          @click="browseCat = c.key; detail = null">
          <span>{{ c.label }}</span><span class="bcat-count">{{ c.count }}</span>
        </div>
      </div>

      <!-- Detail -->
      <div v-if="detail" class="browse-detail">
        <button class="back-btn" @click="closeDetail">← 返回列表</button>
        <h2>{{ detail.repo }}</h2>
        <div class="detail-meta">
          <span>by {{ detail.owner }}</span>
          <span>⭐ {{ formatStars(detail.stars) }}</span>
          <span>{{ (detail.updatedAt || '').slice(0, 10) }}</span>
        </div>

        <!-- Repo link -->
        <a class="detail-link" :href="detail.htmlUrl" target="_blank" rel="noopener">
          🌐 {{ detail.htmlUrl }}
        </a>

        <div v-if="detail.description" class="detail-desc">{{ detail.description }}</div>

        <!-- Releases section -->
        <div class="detail-section">
          <div class="detail-section-hdr">
            <h3>📦 版本</h3>
            <button v-if="!detailReleases.length || loadingReleases" class="btn-go outline sm" :disabled="loadingReleases" @click="loadReleases">
              {{ loadingReleases ? '加载中...' : '查看版本' }}
            </button>
          </div>
          <div v-if="loadingReleases" class="loading-sm">加载中...</div>
          <div v-if="releasesError" class="error-sm">
            {{ releasesError }}
            <button v-if="releasesError.includes('没有发布')" class="btn-go outline sm" style="margin-left:8px" :disabled="downloadingLatest" @click="downloadLatest">
              {{ downloadingLatest ? '下载中...' : '⬇ 下载最新代码' }}
            </button>
          </div>
          <div v-if="detailReleases.length > 0" class="release-list">
            <div v-for="rel in detailReleases" :key="rel.tag_name" class="release-item">
              <div class="rel-left">
                <span class="rel-tag">{{ rel.tag_name }}</span>
                <span v-if="rel.name && rel.name !== rel.tag_name" class="rel-name">{{ rel.name }}</span>
                <span class="rel-date">{{ (rel.published_at || '').slice(0, 10) }}</span>
              </div>
              <button
                class="btn-go xs"
                :disabled="downloadingRelease === rel.tag_name"
                @click="downloadRelease(rel)">
                {{ downloadingRelease === rel.tag_name ? '下载中...' : '下载' }}
              </button>
            </div>
          </div>
          <div v-else-if="!loadingReleases && !releasesError && detailReleases.length === 0" class="hint-text">
            点击「查看版本」获取发布列表
          </div>
        </div>

        <!-- README -->
        <div v-if="!detailReadme && !detailLoading" class="detail-section">
          <button class="btn-go outline sm" @click="loadReadme">📖 查看 README</button>
        </div>
        <div class="detail-readme" v-if="detailReadme || detailLoading">
          <h3>📖 README</h3>
          <div v-if="detailLoading" class="loading-sm">加载中...</div>
          <div v-else class="readme-text" v-html="renderedReadme"></div>
        </div>
        <div v-if="detailReadme" class="detail-section">
          <button class="btn-go outline sm" :disabled="detailTranslating" @click="translateDetail">
            {{ detailTranslating ? '翻译中...' : '🌐 翻译 README' }}
          </button>
        </div>
        <div v-if="detailTranslated" class="detail-readme">
          <h3>📝 中文翻译</h3>
          <div class="readme-text" v-html="detailTranslated"></div>
        </div>
      </div>

      <!-- Grid -->
      <div v-else class="browse-grid">
        <div v-for="m in filteredMods" :key="m.name" class="browse-card" @click="openOnlineMod(m)">
          <div class="browse-top"><span class="browse-name">{{ m.repo }}</span><span class="browse-stars">⭐ {{ formatStars(m.stars) }}</span></div>
          <div class="browse-owner">by {{ m.owner }}</div>
          <div v-if="m.description" class="browse-desc">{{ m.description }}</div>
          <div class="browse-meta">
            <span v-for="t in (m.topics || []).slice(0,4)" :key="t" class="browse-topic">{{ t.replace('unciv-mod-','') }}</span>
            <span class="browse-date">{{ (m.updatedAt || '').slice(0, 10) }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="!loadedMods" class="placeholder">点击「搜索」浏览 GitHub 上的 Unciv 模组</div>
    <div v-else class="placeholder">未找到模组，请尝试其他关键词</div>
  </div>
</template>

<style scoped>
.browse-view { height: 100%; }
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 12px; color: var(--text-primary); }
.add-bar { display: flex; gap: 8px; margin-bottom: 10px; }
.add-bar input { flex: 1; padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 14px; }
.btn-go { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; white-space: nowrap; }
.btn-go:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-go.outline { background: transparent; border: 1px solid var(--accent); color: var(--accent); }
.btn-go.outline.sm { padding: 5px 12px; font-size: 12px; }
.btn-go.xs { padding: 3px 10px; font-size: 11px; }
.error-toast { background: rgba(255,107,107,0.1); color: var(--danger); padding: 10px 14px; border-radius: 4px; margin-bottom: 10px; font-size: 13px; }
.loading { text-align: center; padding: 40px; color: var(--text-muted); }
.placeholder { text-align: center; padding: 60px; color: var(--text-muted); }

.sort-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; font-size: 12px; padding: 0 2px; }
.sort-label { color: var(--text-muted); }
.sort-btn { padding: 3px 8px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-secondary); cursor: pointer; font-size: 11px; }
.sort-btn:hover { color: var(--text-primary); }
.sort-btn.active { background: var(--accent); color: #fff; border-color: var(--accent); }
.sort-total { margin-left: auto; color: var(--text-muted); font-size: 11px; }

.browse-body { display: flex; flex: 1; overflow: hidden; gap: 0; margin-top: 8px; height: calc(100% - 60px); }
.browse-sidebar { width: 110px; flex-shrink: 0; overflow-y: auto; background: var(--bg-card); border-radius: 6px; }
.bcat-item { display: flex; justify-content: space-between; padding: 8px 10px; cursor: pointer; font-size: 12px; color: var(--text-primary); border-bottom: 1px solid var(--border-color); }
.bcat-item:last-child { border-bottom: none; }
.bcat-item:hover { background: var(--sidebar-active); color: var(--text-primary); }
.bcat-item.active { background: var(--accent); color: #fff; }
.bcat-count { font-size: 10px; opacity: 0.7; }

.browse-detail { flex: 1; overflow-y: auto; padding: 0 0 0 12px; }
.back-btn { padding: 6px 16px; background: var(--bg-card); border: 1px solid var(--accent); border-radius: 6px; color: var(--accent); cursor: pointer; font-size: 13px; font-weight: 600; margin-bottom: 14px; display: inline-flex; align-items: center; gap: 6px; }
.back-btn:hover { background: var(--accent); color: #fff; }
.browse-detail h2 { font-size: 20px; margin-bottom: 6px; color: var(--text-primary); }
.detail-meta { display: flex; gap: 16px; font-size: 13px; color: var(--text-muted); margin-bottom: 6px; }
.detail-link { display: block; font-size: 12px; color: var(--accent); margin-bottom: 8px; word-break: break-all; text-decoration: none; }
.detail-link:hover { text-decoration: underline; }
.detail-desc { font-size: 13px; color: var(--text-secondary); margin-bottom: 12px; }
.detail-section { margin-bottom: 12px; }
.detail-section-hdr { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.detail-section-hdr h3 { font-size: 14px; color: var(--text-primary); margin: 0; }
.hint-text { font-size: 12px; color: var(--text-muted); }
.loading-sm { font-size: 12px; color: var(--text-muted); padding: 10px 0; }
.error-sm { font-size: 12px; color: var(--danger); padding: 4px 0; }
.release-list { display: flex; flex-direction: column; gap: 4px; }
.release-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 10px; background: var(--bg-secondary); border-radius: 4px; }
.rel-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.rel-tag { font-family: monospace; font-size: 12px; font-weight: 600; background: var(--accent); color: #fff; padding: 2px 6px; border-radius: 3px; }
.rel-name { font-size: 12px; color: var(--text-secondary); }
.rel-date { font-size: 11px; color: var(--text-muted); }
.detail-readme { margin-bottom: 14px; }
.detail-readme h3 { font-size: 14px; margin-bottom: 6px; color: var(--text-primary); }
.readme-text { background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 6px; padding: 14px; font-size: 13px; line-height: 1.7; white-space: pre-wrap; word-break: break-word; color: var(--text-primary); }

.browse-grid { flex: 1; overflow-y: auto; padding: 0 0 0 12px; display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 10px; align-content: start; }
.browse-card { background: var(--bg-card); border-radius: 8px; padding: 14px; cursor: pointer; transition: transform 0.15s; box-shadow: var(--card-shadow); border: 1px solid transparent; }
.browse-card:hover { transform: translateY(-2px); border-color: var(--accent); }
.browse-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.browse-name { font-size: 15px; font-weight: 600; }
.browse-stars { font-size: 12px; color: var(--warning); }
.browse-owner { font-size: 12px; color: var(--text-muted); margin-bottom: 6px; }
.browse-desc { font-size: 12px; color: var(--text-secondary); line-height: 1.4; margin-bottom: 8px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.browse-meta { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
.browse-topic { font-size: 10px; background: var(--bg-secondary); padding: 1px 6px; border-radius: 3px; color: var(--text-muted); }
.browse-date { font-size: 10px; color: var(--text-muted); margin-left: auto; }
</style>
