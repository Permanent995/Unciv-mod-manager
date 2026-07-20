<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { GetAppConfig, SaveAppConfig, SelectUncivDir, AutoDetectUncivPath, GetMirrorHealth, TestSingleMirror } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'

type MirrorInfo = {
  url: string; label: string; latency: number
  alive: boolean; isCustom: boolean; lastChecked: string
}

const config = ref<app.AppConfig>({
  uncivPath: '', savedPaths: [], lastActivePath: '',
  zoomLevel: 100, sidebarPos: 'left', sidebarWidth: 220, hiddenNav: [], theme: 'light',
  translateProvider: 'microsoft', translateCustomUrl: '', translateCustomKey: '', translateCustomModel: '',
  githubToken: '', mpServer: '', mpUid: '', mpPassword: '',
  customMirrors: [], mirrorMode: 'auto', selectedMirror: '',
})

const navItems = [
  { id: 'mods', label: '模组库' },
  { id: 'maps', label: '地图' },
  { id: 'downloads', label: '下载' },
  { id: 'toolbox', label: '工具箱' },
  { id: 'settings', label: '设置', locked: true },
  { id: 'about', label: '关于', locked: true },
]

const mirrorHealth = ref<MirrorInfo[]>([])
const newMirrorUrl = ref('')
const testingRef = ref(false)
const msg = ref('')

onMounted(async () => {
  config.value = await GetAppConfig()
  mirrorHealth.value = await GetMirrorHealth()
})

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
  if (locked) return
  const h = config.value.hiddenNav || []
  const idx = h.indexOf(id)
  if (idx >= 0) h.splice(idx, 1)
  else h.push(id)
  config.value.hiddenNav = h
  save()
}

function setZoom(v: number) { config.value.zoomLevel = v; save() }
function setSidebar(pos: string) { config.value.sidebarPos = pos; save() }

// Mirror functions
async function refreshHealth() {
  testingRef.value = true
  msg.value = ''
  try {
    mirrorHealth.value = await GetMirrorHealth()
  } catch (e: any) {
    msg.value = '测速失败: ' + e
  } finally {
    testingRef.value = false
  }
}

async function testAndAddMirror() {
  if (!newMirrorUrl.value) return
  const latency = await TestSingleMirror(newMirrorUrl.value)
  if (latency < 0) {
    msg.value = '该镜像不可用'
    return
  }
  config.value.customMirrors = [...(config.value.customMirrors || []), newMirrorUrl.value]
  await save()
  mirrorHealth.value = await GetMirrorHealth()
  newMirrorUrl.value = ''
  msg.value = '已添加镜像'
}

async function removeMirror(url: string) {
  config.value.customMirrors = (config.value.customMirrors || []).filter((u: string) => u !== url)
  await save()
  mirrorHealth.value = await GetMirrorHealth()
}

const selectedHealth = computed(() =>
  mirrorHealth.value.find((m: MirrorInfo) => m.url === config.value.selectedMirror)
)

function mirrorStatus(m: MirrorInfo): string {
  if (!m.alive) return '不可用'
  return m.latency > 0 ? m.latency + 'ms' : '未知'
}

function formatTime(iso: string): string {
  if (!iso) return '未测'
  return iso.slice(0, 19).replace('T', ' ')
}
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

    <!-- Mirror Settings -->
    <div class="section">
      <h2>镜像线路</h2>
      <p class="hint">用于加速 GitHub 访问，适合国内用户</p>

      <div class="row" style="margin-bottom:8px">
        <label><input type="radio" value="auto" v-model="config.mirrorMode" @change="save" /> 自动（故障切换）</label>
        <label><input type="radio" value="manual" v-model="config.mirrorMode" @change="save" /> 手动选择</label>
      </div>

      <div v-if="config.mirrorMode === 'manual'" class="mirror-manual">
        <div class="row">
          <select v-model="config.selectedMirror" @change="save" class="mirror-select">
            <option value="direct">直连</option>
            <option v-for="m in mirrorHealth" :key="m.url" :value="m.url" :disabled="!m.alive">
              {{ m.label }} ({{ mirrorStatus(m) }})
            </option>
          </select>
          <span v-if="selectedHealth" class="mirror-status" :class="{ alive: selectedHealth.alive, dead: !selectedHealth.alive }">
            {{ selectedHealth.alive ? '正常' : '不可用' }}
          </span>
        </div>
      </div>

      <div class="mirror-list">
        <div v-for="m in mirrorHealth" :key="m.url" class="mirror-row">
          <span class="mirror-url">{{ m.label }}</span>
          <span v-if="m.isCustom" class="mirror-badge">自定义</span>
          <span class="mirror-latency" :class="{ dead: !m.alive }">
            {{ mirrorStatus(m) }}
          </span>
          <span class="mirror-checked">{{ formatTime(m.lastChecked) }}</span>
          <button v-if="m.isCustom" class="btn-sm danger" @click="removeMirror(m.url)">删除</button>
        </div>
      </div>

      <div class="add-mirror">
        <input v-model="newMirrorUrl" placeholder="https://your-mirror.com/" @keyup.enter="testAndAddMirror" />
        <button :disabled="!newMirrorUrl" @click="testAndAddMirror">测试并添加</button>
      </div>

      <div class="mirror-actions">
        <button class="btn-retest" @click="refreshHealth" :disabled="testingRef">
          {{ testingRef ? '⏳ 测速中...' : '🔄 重新测试所有镜像' }}
        </button>
      </div>

      <div v-if="msg" class="mirror-msg" :class="{ ok: msg.includes('已添加') }">{{ msg }}</div>
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

/* Mirror section */
.mirror-manual { margin-bottom: 10px; }
.mirror-select { flex: 1; padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.mirror-status { font-size: 12px; font-weight: 600; }
.mirror-status.alive { color: var(--success); }
.mirror-status.dead { color: var(--danger); }
.mirror-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 10px; }
.mirror-row { display: flex; align-items: center; gap: 8px; padding: 6px 8px; background: var(--bg-secondary); border-radius: 4px; font-size: 13px; }
.mirror-url { font-weight: 600; color: var(--text-primary); min-width: 120px; }
.mirror-badge { font-size: 10px; background: var(--warning); color: #fff; padding: 1px 6px; border-radius: 3px; }
.mirror-latency { color: var(--text-secondary); min-width: 50px; text-align: right; }
.mirror-latency.dead { color: var(--danger); }
.mirror-checked { font-size: 11px; color: var(--text-muted); flex: 1; }
.btn-sm { padding: 3px 10px; background: var(--border-color); color: var(--text-secondary); border: none; border-radius: 3px; cursor: pointer; font-size: 12px; }
.btn-sm.danger { color: var(--danger); }
.btn-sm:hover { background: var(--text-muted); color: var(--text-primary); }
.add-mirror { display: flex; gap: 6px; margin-bottom: 8px; }
.add-mirror input { flex: 1; padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.add-mirror button { padding: 6px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.add-mirror button:disabled { opacity: 0.5; cursor: not-allowed; }
.mirror-actions { margin-bottom: 6px; }
.btn-retest { padding: 6px 14px; background: transparent; border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-secondary); cursor: pointer; font-size: 13px; }
.btn-retest:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-retest:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.mirror-msg { font-size: 12px; padding: 4px 0; color: var(--danger); }
.mirror-msg.ok { color: var(--success); }
</style>
