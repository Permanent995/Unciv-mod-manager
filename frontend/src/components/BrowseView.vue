<script setup lang="ts">
import { ref, computed } from 'vue'
import { SearchOnlineMods, FetchReadme } from '../../wailsjs/go/app/App'
import { TranslateText } from '../../wailsjs/go/app/App'

type OnlineMod = {
  name: string; owner: string; repo: string; description: string
  stars: number; updatedAt: string; topics: string[]; htmlUrl: string
}

const browseQuery = ref('')
const onlineMods = ref<OnlineMod[]>([])
const browsing = ref(false)
const browseErr = ref('')
const browseCat = ref('all')
const sortMode = ref<'stars' | 'name' | 'updated'>('stars')
const loadedMods = ref(false) // track if we've loaded at least once

const detail = ref<OnlineMod | null>(null)
const detailReadme = ref('')
const detailLoading = ref(false)
const detailTranslating = ref(false)
const detailTranslated = ref('')

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
  // Filter by category
  if (browseCat.value !== 'all') {
    list = list.filter(m => {
      for (const t of m.topics || []) {
        if (t.replace('unciv-mod-', '') === browseCat.value) return true
      }
      return browseCat.value === 'other' && !(m.topics || []).some(t => catLabels[t.replace('unciv-mod-', '')])
    })
  }
  // Sort
  const sorted = [...list]
  if (sortMode.value === 'stars') sorted.sort((a, b) => b.stars - a.stars)
  else if (sortMode.value === 'name') sorted.sort((a, b) => (a.repo || '').localeCompare(b.repo || ''))
  else if (sortMode.value === 'updated') sorted.sort((a, b) => (b.updatedAt || '').localeCompare(a.updatedAt || ''))
  return sorted
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
}

async function loadReadme() {
  if (!detail.value) return
  detailLoading.value = true
  detailReadme.value = ''
  try { detailReadme.value = await FetchReadme(detail.value.owner, detail.value.repo) }
  catch { detailReadme.value = '无法加载 README' }
  finally { detailLoading.value = false }
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
function formatStars(n: number): string { return n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n) }
</script>

<template>
  <div class="browse-view">
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
          <span>by {{ detail.owner }}</span><span>⭐ {{ formatStars(detail.stars) }}</span>
          <span>{{ (detail.updatedAt || '').slice(0, 10) }}</span>
        </div>
        <div v-if="detail.description" class="detail-desc">{{ detail.description }}</div>
        <div v-if="!detailReadme && !detailLoading" style="margin-bottom:12px">
          <button class="btn-go outline" @click="loadReadme">📖 查看 README</button>
        </div>
        <div class="detail-readme" v-if="detailReadme || detailLoading">
          <h3>📖 README</h3>
          <div v-if="detailLoading" class="loading-sm">加载中...</div>
          <pre v-else class="readme-text">{{ detailReadme }}</pre>
        </div>
        <div v-if="detailReadme">
          <button class="btn-go outline" :disabled="detailTranslating" @click="translateDetail">
            {{ detailTranslating ? '翻译中...' : '🌐 翻译 README' }}
          </button>
        </div>
        <div v-if="detailTranslated" class="detail-readme">
          <h3>📝 中文翻译</h3>
          <pre class="readme-text">{{ detailTranslated }}</pre>
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
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 12px; }
.add-bar { display: flex; gap: 8px; margin-bottom: 10px; }
.add-bar input { flex: 1; padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 14px; }
.btn-go { padding: 8px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; white-space: nowrap; }
.btn-go:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-go.outline { background: transparent; border: 1px solid var(--accent); color: var(--accent); margin-top: 10px; }
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
.bcat-item { display: flex; justify-content: space-between; padding: 8px 10px; cursor: pointer; font-size: 12px; color: var(--text-secondary); border-bottom: 1px solid var(--border-color); }
.bcat-item:last-child { border-bottom: none; }
.bcat-item:hover { background: var(--sidebar-active); color: var(--text-primary); }
.bcat-item.active { background: var(--accent); color: #fff; }
.bcat-count { font-size: 10px; opacity: 0.7; }

.browse-detail { flex: 1; overflow-y: auto; padding: 0 0 0 12px; }
.back-btn { padding: 4px 12px; background: transparent; border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-secondary); cursor: pointer; font-size: 12px; margin-bottom: 10px; }
.browse-detail h2 { font-size: 20px; margin-bottom: 6px; color: var(--text-primary); }
.detail-meta { display: flex; gap: 16px; font-size: 13px; color: var(--text-muted); margin-bottom: 8px; }
.detail-desc { font-size: 13px; color: var(--text-secondary); margin-bottom: 12px; }
.detail-readme { margin-bottom: 14px; }
.detail-readme h3 { font-size: 14px; margin-bottom: 6px; color: var(--text-secondary); }
.loading-sm { font-size: 12px; color: var(--text-muted); padding: 20px; }
.readme-text { background: #1a1a24; border: 1px solid #333; border-radius: 6px; padding: 14px; font-size: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-word; color: #d4d4d4; }

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
