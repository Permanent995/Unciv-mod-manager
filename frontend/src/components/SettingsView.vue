<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetAppConfig, SaveAppConfig, SelectUncivDir, AutoDetectUncivPath } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'

const config = ref<app.AppConfig>({
  uncivPath: '', savedPaths: [], lastActivePath: '',
  zoomLevel: 100, sidebarPos: 'left', sidebarWidth: 220, hiddenNav: [], theme: 'light',
  translateProvider: 'microsoft', translateCustomUrl: '', translateCustomKey: '', translateCustomModel: '',
  githubToken: '', mpServer: '', mpUid: '', mpPassword: '',
})

const navItems = [
  { id: 'mods', label: '模组库' },
  { id: 'maps', label: '地图' },
  { id: 'downloads', label: '下载' },
  { id: 'toolbox', label: '工具箱' },
  { id: 'settings', label: '设置', locked: true },
  { id: 'about', label: '关于', locked: true },
]

onMounted(async () => { config.value = await GetAppConfig() })

async function save() { await SaveAppConfig(config.value) }

async function selectPath() {
  try {
    const path = await SelectUncivDir()
    if (path) config.value = await GetAppConfig()
  } catch (e) { console.error(e) }
}

async function autoDetect() {
  try {
    const path = await AutoDetectUncivPath()
    if (path) config.value = await GetAppConfig()
  } catch (e) { console.error(e) }
}

function toggleNav(id: string, locked?: boolean) {
  if (locked) return // 禁止隐藏设置和关于
  const h = config.value.hiddenNav || []
  const idx = h.indexOf(id)
  if (idx >= 0) h.splice(idx, 1)
  else h.push(id)
  config.value.hiddenNav = h
  save()
}

function setZoom(v: number) { config.value.zoomLevel = v; save() }
function setSidebar(pos: string) { config.value.sidebarPos = pos; save() }
</script>

<template>
  <div class="settings-view">
    <div class="view-header"><h1>设置</h1></div>

    <div class="section">
      <h2>Unciv 路径</h2>
      <div class="row">
        <input v-model="config.uncivPath" readonly placeholder="未设置" />
        <button @click="selectPath">选择目录</button>
        <button @click="autoDetect">自动检测</button>
      </div>
    </div>

    <div class="section">
      <h2>页面缩放</h2>
      <div class="row">
        <input type="range" min="80" max="150" :value="config.zoomLevel || 100" @input="setZoom(Number(($event.target as HTMLInputElement).value))" />
        <span class="val">{{ config.zoomLevel || 100 }}%</span>
      </div>
    </div>

    <div class="section">
      <h2>侧边栏位置</h2>
      <div class="row">
        <label><input type="radio" value="left" :checked="config.sidebarPos !== 'right'" @change="setSidebar('left')" /> 左侧</label>
        <label><input type="radio" value="right" :checked="config.sidebarPos === 'right'" @change="setSidebar('right')" /> 右侧</label>
      </div>
    </div>

    <div class="section">
      <h2>隐藏功能</h2>
      <p class="hint">取消勾选可隐藏导航项，设置和关于不可隐藏</p>
      <div class="check-group">
        <label v-for="item in navItems" :key="item.id" class="check-row" :class="{ locked: item.locked }">
          <input type="checkbox" :checked="!(config.hiddenNav || []).includes(item.id)" :disabled="item.locked" @change="toggleNav(item.id, item.locked)" />
          {{ item.label }}
        </label>
      </div>
    </div>

    <div class="section">
      <h2>翻译服务</h2>
      <p class="hint">用于模组 README 翻译，微软/Yandex 免费免配置</p>
      <div class="row" style="margin-bottom:8px">
        <label><input type="radio" value="microsoft" v-model="config.translateProvider" @change="save" /> 微软翻译（免费）</label>
        <label><input type="radio" value="yandex" v-model="config.translateProvider" @change="save" /> Yandex（免费）</label>
        <label><input type="radio" value="custom" v-model="config.translateProvider" @change="save" /> 自定义 AI</label>
      </div>
      <div v-if="config.translateProvider === 'custom'" class="custom-translate">
        <input v-model="config.translateCustomUrl" placeholder="API 地址 (如 https://api.deepseek.com)" @change="save" />
        <input v-model="config.translateCustomKey" placeholder="API Key" @change="save" type="password" />
        <input v-model="config.translateCustomModel" placeholder="模型 (如 deepseek-chat)" @change="save" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-view { height: 100%; max-width: 600px; }
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 20px; color: var(--text-primary); }
.section { background: var(--bg-card); border-radius: 8px; padding: 16px; margin-bottom: 12px; box-shadow: var(--card-shadow); }
.section h2 { font-size: 16px; font-weight: 600; margin-bottom: 10px; color: var(--text-primary); }
.row { display: flex; align-items: center; gap: 8px; }
.hint { font-size: 12px; color: var(--text-muted); margin-bottom: 8px; }
.row input[type="text"] { flex: 1; padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); }
.row button { padding: 6px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; }
.val { font-size: 14px; color: var(--text-secondary); min-width: 40px; }
.check-group { display: flex; flex-direction: column; gap: 6px; }
.check-row { display: flex; align-items: center; gap: 6px; font-size: 14px; cursor: pointer; }
.check-row.locked { color: var(--text-muted); cursor: not-allowed; }
.check-row input { width: 16px; height: 16px; }
.check-row.locked input { opacity: 0.5; }
label { cursor: pointer; font-size: 14px; }
label input { margin-right: 4px; }
.custom-translate { display: flex; flex-direction: column; gap: 8px; margin-top: 8px; }
.custom-translate input { padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
</style>
