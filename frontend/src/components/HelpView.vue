<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ListDocs, ReadDoc } from '../../wailsjs/go/main/DocReader'
import { ExportLogFile } from '../../wailsjs/go/app/App'

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

// Log export state
const showExportDialog = ref(false)
const exporting = ref(false)
const exportMsg = ref('')
const exportOk = ref(false)

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

function showExport() {
  exportMsg.value = ''
  exportOk.value = false
  showExportDialog.value = true
}

function cancelExport() {
  showExportDialog.value = false
}

async function doExport() {
  showExportDialog.value = false
  exporting.value = true
  exportMsg.value = ''
  exportOk.value = false
  try {
    const path = await ExportLogFile()
    if (path) {
      exportMsg.value = '日志已导出到: ' + path
      exportOk.value = true
    }
  } catch (e: any) {
    exportMsg.value = '导出失败: ' + e
    exportOk.value = false
  } finally {
    exporting.value = false
  }
}
</script>

<template>
  <div class="help-view view-card">
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

        <div class="sidebar-divider"></div>

        <div class="help-sidebar-title">导出日志</div>
        <button class="export-btn" :disabled="exporting" @click="showExport">
          {{ exporting ? '导出中...' : '💾 导出为 TXT' }}
        </button>
        <div v-if="exportMsg" class="export-status" :class="{ ok: exportOk }">{{ exportMsg }}</div>
      </div>

      <div class="help-content">
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else-if="errMsg" class="error">{{ errMsg }}</div>
        <pre v-else-if="docContent" class="doc-content">{{ docContent }}</pre>
        <div v-else class="empty-hint">请从左侧选择一篇文档</div>
      </div>
    </div>

    <!-- Export confirmation dialog -->
    <Teleport to="body">
      <div v-if="showExportDialog" class="modal-overlay" @click.self="cancelExport">
        <div class="modal-dialog">
          <div class="modal-header">导出日志</div>
          <p class="modal-body">很抱歉，您的游戏出现了一些问题。如果要寻求帮助，请将错误报告发给他人，而不是发送报错窗口的截图。</p>
          <div class="modal-buttons">
            <button class="modal-btn" @click="cancelExport">取消</button>
            <button class="modal-btn primary" @click="doExport">导出日志</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.help-view { height: 100%; display: flex; flex-direction: column; }
.view-header { margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 700; color: var(--text-primary); }

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

.sidebar-divider { height: 1px; background: var(--border-color); margin: 10px 14px; }

.export-btn {
  margin: 0 14px; padding: 8px 12px;
  background: var(--accent); color: #fff;
  border: none; border-radius: 6px;
  font-size: 13px; font-weight: 600; cursor: pointer;
  text-align: center; transition: opacity 0.15s;
}
.export-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.export-btn:hover:not(:disabled) { opacity: 0.9; }
.export-status { font-size: 11px; padding: 4px 14px; color: var(--danger); word-break: break-all; }
.export-status.ok { color: var(--success); }

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

<style>
/* Unscoped: modal over body */
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex; align-items: center; justify-content: center;
  z-index: var(--z-modal);
}
.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
  min-width: 360px;
  max-width: 460px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.2);
}
.modal-header {
  font-size: 16px; font-weight: 700;
  margin-bottom: 12px;
  color: var(--text-primary);
}
.modal-body {
  font-size: 14px; line-height: 1.6;
  color: var(--text-secondary);
  margin-bottom: 20px;
}
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
