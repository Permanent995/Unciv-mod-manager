<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetDevDoc } from '../../wailsjs/go/main/DocReader'

const docContent = ref('')
const loading = ref(false)
const activeSection = ref('devdoc')

interface DocSection {
  id: string
  label: string
}

const sections: DocSection[] = [
  { id: 'devdoc', label: '开发文档查看' },
]

onMounted(loadDoc)

async function loadDoc() {
  loading.value = true
  try {
    docContent.value = await GetDevDoc()
  } catch (e: any) {
    docContent.value = '# 读取失败\n\n' + e
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
        <div
          v-for="s in sections"
          :key="s.id"
          class="help-nav-item"
          :class="{ active: activeSection === s.id }"
          @click="activeSection = s.id"
        >
          {{ s.label }}
        </div>
      </div>

      <div class="help-content">
        <div v-if="activeSection === 'devdoc'" class="doc-section">
          <h2>开发文档</h2>
          <div v-if="loading" class="loading">加载中...</div>
          <pre v-else class="doc-content">{{ docContent }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.help-view { height: 100%; display: flex; flex-direction: column; }
.view-header { margin-bottom: 20px; }
.view-header h1 { font-size: 24px; font-weight: 700; }

.help-layout { display: flex; gap: 20px; flex: 1; overflow: hidden; }

.help-sidebar { width: 180px; flex-shrink: 0; display: flex; flex-direction: column; gap: 2px; }
.help-nav-item {
  padding: 10px 14px; border-radius: 6px; cursor: pointer; font-size: 14px; font-weight: 500;
  color: var(--text-primary); transition: background 0.15s;
}
.help-nav-item:hover { background: var(--bg-hover); }
.help-nav-item.active { background: var(--accent); color: #fff; }

.help-content { flex: 1; overflow-y: auto; }

.doc-section h2 { font-size: 18px; font-weight: 700; margin-bottom: 12px; }
.loading { padding: 20px; color: var(--text-muted); }

.doc-content {
  font-family: 'Microsoft YaHei', 'Segoe UI', monospace;
  font-size: 13px; line-height: 1.7; white-space: pre-wrap; word-wrap: break-word;
  background: var(--bg-card); border-radius: 8px; padding: 16px;
  color: var(--text-primary); border: 1px solid var(--border-color);
  max-height: calc(100vh - 200px); overflow-y: auto;
}
</style>
