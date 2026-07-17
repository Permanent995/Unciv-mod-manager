<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ListDocs, ReadDoc } from '../../wailsjs/go/main/DocReader'

interface DocInfo {
  name: string
  title: string
  size: number
  modTime: string
}

const docs = ref<DocInfo[]>([])
const docContent = ref('')
const loading = ref(false)
const loadingList = ref(false)
const activeDoc = ref('')
const errMsg = ref('')

onMounted(loadDocs)

async function loadDocs() {
  loadingList.value = true
  errMsg.value = ''
  try {
    docs.value = await ListDocs()
    if (docs.value.length > 0 && !activeDoc.value) {
      selectDoc(docs.value[0])
    }
  } catch (e: any) {
    errMsg.value = '读取文档列表失败: ' + e
  } finally {
    loadingList.value = false
  }
}

async function selectDoc(doc: DocInfo) {
  activeDoc.value = doc.name
  loading.value = true
  errMsg.value = ''
  docContent.value = ''
  try {
    docContent.value = await ReadDoc(doc.name)
  } catch (e: any) {
    errMsg.value = '读取文档失败: ' + e
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="help-view">
    <div class="view-header">
      <h1>📖 帮助</h1>
    </div>

    <div class="help-layout">
      <div class="help-sidebar">
        <div class="help-sidebar-title">开发文档</div>
        <a class="github-link" href="https://github.com/Permanent995/Unciv-mod-manager/tree/master/docs" target="_blank" rel="noopener">
          <span class="gh-icon">&#x1F30D;</span> 在 GitHub 上查看
        </a>
        <div v-if="loadingList" class="loading">加载中...</div>
        <div
          v-for="doc in docs"
          :key="doc.name"
          class="help-nav-item"
          :class="{ active: activeDoc === doc.name }"
          @click="selectDoc(doc)"
        >
          <span class="doc-title">{{ doc.title }}</span>
          <span class="doc-meta">{{ doc.modTime }}</span>
        </div>
        <div v-if="!loadingList && docs.length === 0" class="no-docs">
          未找到开发文档
        </div>
      </div>

      <div class="help-content">
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else-if="errMsg" class="error">{{ errMsg }}</div>
        <pre v-else-if="docContent" class="doc-content">{{ docContent }}</pre>
        <div v-else class="empty-hint">请从左侧选择一篇文档</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.help-view { height: 100%; display: flex; flex-direction: column; }
.view-header { margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 700; }

.help-layout { display: flex; gap: 20px; flex: 1; overflow: hidden; }

.help-sidebar { width: 220px; flex-shrink: 0; display: flex; flex-direction: column; gap: 2px; overflow-y: auto; }
.help-sidebar-title { font-size: 13px; font-weight: 600; color: var(--text-muted); padding: 4px 14px 8px; text-transform: uppercase; letter-spacing: 0.5px; }
.github-link {
  display: flex; align-items: center; gap: 4px;
  padding: 6px 14px; margin: 0 0 6px 0;
  font-size: 12px; font-weight: 600; color: var(--accent);
  text-decoration: none; border-radius: 4px;
  transition: background 0.15s;
}
.github-link:hover { background: var(--bg-hover); }
.gh-icon { font-size: 14px; }
.help-nav-item {
  padding: 10px 14px; border-radius: 6px; cursor: pointer; transition: background 0.15s;
  display: flex; flex-direction: column; gap: 2px;
}
.help-nav-item:hover { background: var(--bg-hover); }
.help-nav-item.active { background: var(--accent); color: #fff; }
.help-nav-item.active .doc-meta { color: rgba(255,255,255,0.7); }
.doc-title { font-size: 13px; font-weight: 600; line-height: 1.3; }
.doc-meta { font-size: 11px; color: var(--text-muted); }

.help-content { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.loading { padding: 20px; color: var(--text-muted); }
.error { padding: 20px; color: var(--danger); }
.empty-hint { padding: 40px; text-align: center; color: var(--text-muted); }
.no-docs { padding: 14px; color: var(--text-muted); font-size: 13px; }

.doc-content {
  flex: 1; overflow-y: auto;
  font-family: 'Microsoft YaHei', 'Segoe UI', monospace;
  font-size: 13px; line-height: 1.7; white-space: pre-wrap; word-wrap: break-word;
  background: var(--bg-card); border-radius: 8px; padding: 20px;
  color: var(--text-primary); border: 1px solid var(--border-color);
}
</style>
