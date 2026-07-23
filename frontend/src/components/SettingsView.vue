<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { GetAppConfig, SaveAppConfig, SelectUncivDir, AutoDetectUncivPaths, SetUncivPath, MigrateUncivData, GetMirrorHealth, TestSingleMirror } from '../../wailsjs/go/app/App'
import { app } from '../../wailsjs/go/models'
import IconFolder from './icons/IconFolder.vue'
import IconSearch from './icons/IconSearch.vue'
import IconEye from './icons/IconEye.vue'
import IconArchive from './icons/IconArchive.vue'
import IconGlobe from './icons/IconGlobe.vue'
import IconRefresh from './icons/IconRefresh.vue'
import IconSun from './icons/IconSun.vue'
import IconCog from './icons/IconCog.vue'

type MirrorInfo = {
  url: string; label: string; latency: number
  alive: boolean; isCustom: boolean; lastChecked: string
}

const config = ref<app.AppConfig>({
  uncivPath: '', savedPaths: [], lastActivePath: '',
  zoomLevel: 100, sidebarPos: 'left', sidebarWidth: 220, hiddenNav: [], theme: 'light', themeVariant: 'pure',
  translateProvider: 'microsoft', translateCustomUrl: '', translateCustomKey: '', translateCustomModel: '',
  githubToken: '', mpServer: '', mpUid: '', mpPassword: '',
  customMirrors: [], mirrorMode: 'auto', selectedMirror: '', maxSaves: 100,
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

const detectedPaths = ref<{ path: string; version: string; hasExe: boolean; hasJar: boolean }[]>([])
const showDetectPopup = ref(false)
const migrateFrom = ref('')
const migrateTo = ref('')
const migrateMsg = ref('')

async function autoDetect() {
  try {
    const paths = await AutoDetectUncivPaths()
    if (paths.length === 1) {
      await SetUncivPath(paths[0].path)
      config.value = await GetAppConfig()
    } else if (paths.length > 1) {
      detectedPaths.value = paths
      if (!localStorage.getItem('umm_hide_detect_popup')) {
        showDetectPopup.value = true
      }
    }
  } catch (e) { console.error(e) }
}

function dismissPopupForever() {
  localStorage.setItem('umm_hide_detect_popup', '1')
  showDetectPopup.value = false
}

async function pickDetectedPath(path: string) {
  showDetectPopup.value = false
  await SetUncivPath(path)
  config.value = await GetAppConfig()
}

async function doMigrate() {
  if (!migrateFrom.value || !migrateTo.value) return
  if (migrateFrom.value === migrateTo.value) {
    migrateMsg.value = '❌ 源版本和目标版本相同'
    return
  }
  try {
    const r = await MigrateUncivData(migrateFrom.value, migrateTo.value)
    migrateMsg.value = `✅ 迁移完成：模组 ${r.mods} 个、存档 ${r.saves} 个、地图 ${r.maps} 个`
  } catch (e: any) {
    migrateMsg.value = '迁移失败: ' + e
  }
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

function setThemeVariant() {
  document.documentElement.setAttribute('data-theme-variant', config.value.themeVariant || 'pure')
  save()
}

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
  <div class="settings-view view-card">
    <div class="view-header"><h1>⚙ 设置</h1></div>

    <div class="settings-grid">

      <div class="card">
        <div class="card-icon"><IconFolder :size="24" /></div>
        <div class="card-body">
          <div class="card-title">Unciv 路径</div>
          <div class="card-desc">设置 Unciv 安装目录</div>
          <div class="row">
            <input v-model="config.uncivPath" readonly placeholder="未设置" class="input-full" />
          </div>
          <div class="row" style="margin-top:8px">
            <button @click="selectPath" class="btn-primary">选择目录</button>
            <button @click="autoDetect" class="btn-outline">自动检测</button>
          </div>
          <!-- 多版本选择弹窗 -->
          <div v-if="showDetectPopup" class="detect-overlay" @click.self="showDetectPopup = false">
            <div class="detect-popup">
              <div class="detect-popup-title">检测到 {{ detectedPaths.length }} 个 Unciv 安装</div>
              <div
                v-for="p in detectedPaths"
                :key="p.path"
                class="detect-option"
                @click="pickDetectedPath(p.path)"
              >
                <div class="detect-option-path">{{ p.path }}</div>
                <div class="detect-option-ver">
                  <template v-if="p.version">🕹️ v{{ p.version }}</template>
                  <template v-else>版本未知</template>
                  <span v-if="p.hasExe" class="tag">exe</span>
                  <span v-if="p.hasJar" class="tag">jar</span>
                </div>
              </div>

              <!-- 迁移区 -->
              <div v-if="detectedPaths.length >= 2" class="migrate-section">
                <div class="migrate-title">📦 迁移数据到新版</div>
                <div class="migrate-row">
                  <label>源版本（旧）：</label>
                  <select v-model="migrateFrom">
                    <option value="" disabled>-- 选择旧版 --</option>
                    <option v-for="p in detectedPaths" :key="p.path" :value="p.path">
                      {{ p.version || p.path.slice(-30) }}
                    </option>
                  </select>
                </div>
                <div class="migrate-row">
                  <label>目标版本（新）：</label>
                  <select v-model="migrateTo">
                    <option value="" disabled>-- 选择新版 --</option>
                    <option v-for="p in detectedPaths" :key="p.path" :value="p.path">
                      {{ p.version || p.path.slice(-30) }}
                    </option>
                  </select>
                </div>
                <button class="btn-primary" style="margin-top:6px;width:100%" @click="doMigrate">
                  ⬇ 迁移 mods + 存档 + 地图
                </button>
                <div v-if="migrateMsg" class="mirror-msg" :class="{ ok: migrateMsg.includes('✅') }">
                  {{ migrateMsg }}
                </div>
              </div>

              <div style="margin-top:10px;display:flex;gap:8px;align-items:center">
                <button class="btn-outline" @click="showDetectPopup = false">关闭</button>
                <a class="dismiss-link" @click="dismissPopupForever">不再提示</a>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconSun :size="24" /></div>
        <div class="card-body">
          <div class="card-title">浅色主题风格</div>
          <div class="card-desc">仅亮色模式下生效</div>
          <select v-model="config.themeVariant" @change="setThemeVariant" class="input-select">
            <option value="pure">纯白</option>
            <option value="warm">暖白</option>
            <option value="blue">浅蓝</option>
            <option value="green">浅绿</option>
          </select>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconSearch :size="24" /></div>
        <div class="card-body">
          <div class="card-title">页面缩放</div>
          <div class="card-desc">调整界面显示大小</div>
          <div class="row">
            <input type="range" min="80" max="150" :value="config.zoomLevel || 100" @input="setZoom(Number(($event.target as HTMLInputElement).value))" class="slider" />
            <span class="val">{{ config.zoomLevel || 100 }}%</span>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconCog :size="24" /></div>
        <div class="card-body">
          <div class="card-title">侧边栏位置</div>
          <div class="card-desc">选择侧边栏在左还是在右</div>
          <div class="row">
            <label class="radio-label"><input type="radio" value="left" :checked="config.sidebarPos !== 'right'" @change="setSidebar('left')" /> 左侧</label>
            <label class="radio-label"><input type="radio" value="right" :checked="config.sidebarPos === 'right'" @change="setSidebar('right')" /> 右侧</label>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconEye :size="24" /></div>
        <div class="card-body">
          <div class="card-title">存档显示数</div>
          <div class="card-desc">最多显示多少条存档，默认 100</div>
          <div class="row">
            <input type="number" min="10" max="1000" step="10" :value="config.maxSaves || 100" @input="config.maxSaves = Number(($event.target as HTMLInputElement).value); save()" class="input-num" />
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconEye :size="24" /></div>
        <div class="card-body">
          <div class="card-title">隐藏功能</div>
          <div class="card-desc">取消勾选可隐藏导航项，设置和关于不可隐藏</div>
          <div class="check-group">
            <label v-for="item in navItems" :key="item.id" class="check-row" :class="{ locked: item.locked }">
              <input type="checkbox" :checked="!(config.hiddenNav || []).includes(item.id)" :disabled="item.locked" @change="toggleNav(item.id, item.locked)" />
              {{ item.label }}
            </label>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconArchive :size="24" /></div>
        <div class="card-body">
          <div class="card-title">一键迁移</div>
          <div class="card-desc">把旧版 Unciv 的 mods、存档、地图复制到新版</div>
          <div class="migrate-row">
            <label>源版本：</label>
            <select v-model="migrateFrom">
              <option value="" disabled>-- 旧版 --</option>
              <option v-for="p in detectedPaths" :key="'src'+p.path" :value="p.path">
                {{ p.version || p.path.slice(-30) }}
              </option>
            </select>
          </div>
          <div class="migrate-row">
            <label>目标版本：</label>
            <select v-model="migrateTo">
              <option value="" disabled>-- 新版 --</option>
              <option v-for="p in detectedPaths" :key="'dst'+p.path" :value="p.path">
                {{ p.version || p.path.slice(-30) }}
              </option>
            </select>
          </div>
          <div class="row" style="margin-top:8px;gap:8px">
            <button class="btn-primary" @click="doMigrate">⬇ 一键迁移</button>
            <button class="btn-outline" @click="autoDetect">🔍 重新扫描</button>
          </div>
          <div v-if="migrateMsg" class="mirror-msg" :class="{ ok: migrateMsg.includes('✅') }">
            {{ migrateMsg }}
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconGlobe :size="24" /></div>
        <div class="card-body">
          <div class="card-title">翻译服务</div>
          <div class="card-desc">用于模组 README 翻译，微软/Yandex 免费免配置</div>
          <div class="row" style="margin-bottom:8px;flex-wrap:wrap">
            <label class="radio-label"><input type="radio" value="microsoft" v-model="config.translateProvider" @change="save" /> 微软翻译（免费）</label>
            <label class="radio-label"><input type="radio" value="yandex" v-model="config.translateProvider" @change="save" /> Yandex（免费）</label>
            <label class="radio-label"><input type="radio" value="custom" v-model="config.translateProvider" @change="save" /> 自定义 AI</label>
          </div>
          <div v-if="config.translateProvider === 'custom'" class="custom-translate">
            <input v-model="config.translateCustomUrl" placeholder="API 地址 (如 https://api.deepseek.com)" @change="save" />
            <input v-model="config.translateCustomKey" placeholder="API Key" @change="save" type="password" />
            <input v-model="config.translateCustomModel" placeholder="模型 (如 deepseek-chat)" @change="save" />
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-icon"><IconRefresh :size="24" /></div>
        <div class="card-body">
          <div class="card-title">镜像线路</div>
          <div class="card-desc">用于加速 GitHub 访问，适合国内用户</div>

          <div class="row" style="margin-bottom:8px;flex-wrap:wrap">
            <label class="radio-label"><input type="radio" value="auto" v-model="config.mirrorMode" @change="save" /> 自动（故障切换）</label>
            <label class="radio-label"><input type="radio" value="manual" v-model="config.mirrorMode" @change="save" /> 手动选择</label>
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
              <span v-if="m.isCustom" class="badge-custom">自定义</span>
              <span class="mirror-latency" :class="{ dead: !m.alive }">{{ mirrorStatus(m) }}</span>
              <span class="mirror-checked">{{ formatTime(m.lastChecked) }}</span>
              <button v-if="m.isCustom" class="btn-sm danger" @click="removeMirror(m.url)">删除</button>
            </div>
          </div>

          <div class="add-mirror">
            <input v-model="newMirrorUrl" placeholder="https://your-mirror.com/" @keyup.enter="testAndAddMirror" />
            <button :disabled="!newMirrorUrl" @click="testAndAddMirror" class="btn-primary">测试并添加</button>
          </div>

          <div class="mirror-actions">
            <button class="btn-retest" @click="refreshHealth" :disabled="testingRef">
              {{ testingRef ? '⏳ 测速中...' : '🔄 重新测试所有镜像' }}
            </button>
          </div>

          <div v-if="msg" class="mirror-msg" :class="{ ok: msg.includes('已添加') }">{{ msg }}</div>
        </div>
      </div>

    </div>
  </div>
</template>

<style scoped>
.settings-view { height: 100%; }
.view-header h1 { font-size: 24px; font-weight: 600; margin-bottom: 20px; color: var(--text-primary); }
.settings-grid { display: flex; flex-direction: column; gap: 12px; max-width: 640px; }

.card { display: flex; gap: 14px; background: var(--bg-card); border-radius: 10px; padding: 18px; box-shadow: var(--card-shadow); border: 1px solid var(--border-color); transition: box-shadow 0.2s; }
.card:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
.card-icon { font-size: 24px; line-height: 1; padding-top: 2px; flex-shrink: 0; }
.card-body { flex: 1; min-width: 0; }
.card-title { font-size: 15px; font-weight: 600; color: var(--text-primary); margin-bottom: 2px; }
.card-desc { font-size: 12px; color: var(--text-muted); margin-bottom: 10px; }

.row { display: flex; align-items: center; gap: 8px; }
.input-full { flex: 1; padding: 7px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.input-full:focus { border-color: var(--accent); }

.btn-primary { padding: 6px 14px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; font-weight: 500; }
.btn-primary:active { opacity: 0.85; }
.btn-outline { padding: 6px 14px; background: transparent; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; font-size: 13px; color: var(--text-secondary); }
.btn-outline:hover { border-color: var(--accent); color: var(--accent); }
.btn-sm { padding: 3px 10px; background: var(--border-color); color: var(--text-secondary); border: none; border-radius: 3px; cursor: pointer; font-size: 12px; }
.btn-sm:hover { background: var(--text-muted); color: var(--text-primary); }
.btn-sm.danger { color: var(--danger); }

.slider { flex: 1; accent-color: var(--accent); }
.val { font-size: 14px; color: var(--text-secondary); min-width: 40px; font-weight: 600; }
.radio-label { display: flex; align-items: center; gap: 4px; cursor: pointer; font-size: 13px; color: var(--text-primary); }
.radio-label input { margin: 0; }

.check-group { display: flex; flex-wrap: wrap; gap: 6px; }
.check-row { display: flex; align-items: center; gap: 6px; font-size: 13px; cursor: pointer; padding: 4px 8px; background: var(--bg-secondary); border-radius: 4px; }
.check-row.locked { color: var(--text-muted); cursor: not-allowed; opacity: 0.6; }
.check-row input { margin: 0; }
.check-row.locked input { opacity: 0.5; }

.custom-translate { display: flex; flex-direction: column; gap: 6px; margin-top: 6px; }
.custom-translate input { padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.custom-translate input:focus { border-color: var(--accent); }

/* Mirror */
.mirror-manual { margin-bottom: 8px; }
.mirror-select { flex: 1; padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.mirror-status { font-size: 12px; font-weight: 600; }
.mirror-status.alive { color: var(--success); }
.mirror-status.dead { color: var(--danger); }
.mirror-list { display: flex; flex-direction: column; gap: 3px; margin-bottom: 8px; max-height: 200px; overflow-y: auto; }
.mirror-row { display: flex; align-items: center; gap: 8px; padding: 5px 8px; background: var(--bg-secondary); border-radius: 4px; font-size: 13px; }
.mirror-url { font-weight: 600; color: var(--text-primary); min-width: 100px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.badge-custom { font-size: 10px; background: var(--warning); color: #fff; padding: 1px 6px; border-radius: 3px; font-weight: 600; }
.mirror-latency { color: var(--text-secondary); min-width: 50px; text-align: right; font-size: 12px; }
.mirror-latency.dead { color: var(--danger); }
.mirror-checked { font-size: 11px; color: var(--text-muted); flex: 1; }
.add-mirror { display: flex; gap: 6px; }
.add-mirror input { flex: 1; padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; }
.add-mirror input:focus { border-color: var(--accent); }
.mirror-actions { margin-top: 8px; }
.btn-retest { padding: 6px 14px; background: transparent; border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-secondary); cursor: pointer; font-size: 13px; }
.btn-retest:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-retest:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.mirror-msg { font-size: 12px; padding: 4px 0; color: var(--danger); }
.mirror-msg.ok { color: var(--success); }

.dismiss-link { font-size: 12px; color: var(--text-muted); cursor: pointer; text-decoration: underline; }
.dismiss-link:hover { color: var(--text-secondary); }

/* ── 多版本检测弹窗 ── */
.detect-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000;
}
.detect-popup {
  background: var(--bg-primary); border: 1px solid var(--border-color);
  border-radius: 12px; padding: 20px; min-width: 420px; max-width: 560px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
}
.detect-popup-title {
  font-size: 15px; font-weight: 700; margin-bottom: 12px;
  color: var(--text-primary);
}
.detect-option {
  padding: 10px 12px; border: 1px solid var(--border-color);
  border-radius: 8px; margin-bottom: 6px; cursor: pointer;
  transition: all 0.15s;
}
.detect-option:hover { border-color: var(--accent); background: rgba(79,70,229,0.05); }
.detect-option-path { font-size: 13px; font-weight: 600; color: var(--text-primary); word-break: break-all; }
.detect-option-ver { font-size: 12px; color: var(--text-secondary); margin-top: 4px; display: flex; align-items: center; gap: 6px; }
.tag { padding: 1px 6px; background: var(--bg-card); border-radius: 3px; font-size: 11px; font-weight: 600; }

/* ── 迁移 ── */
.migrate-section { margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border-color); }
.migrate-title { font-size: 13px; font-weight: 700; margin-bottom: 8px; color: var(--text-primary); }
.migrate-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.migrate-row label { font-size: 12px; color: var(--text-secondary); white-space: nowrap; min-width: 90px; }
.migrate-row select { flex: 1; padding: 6px 8px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 12px; }
.input-select { padding: 6px 10px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 13px; cursor: pointer; }
</style>
