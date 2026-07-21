<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ExportModSnapshot, CompareModSnapshot, GetAppConfig, SaveAppConfig, LaunchUnciv, OpenSnapshotFolder } from '../../wailsjs/go/app/App'

type Diff = { mod: string; issue: string; valueA: string; valueB: string }

const msg = ref('')
const diffs = ref<Diff[]>([])
const loading = ref(false)
const launching = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const cfg = ref({ mpServer: 'http://bj.unciv.cn', mpUid: '', mpPassword: '' })

onMounted(async () => {
  const c = await GetAppConfig()
  cfg.value.mpServer = c.mpServer || 'http://bj.unciv.cn'
  cfg.value.mpUid = c.mpUid || ''
  cfg.value.mpPassword = c.mpPassword || ''
})

async function saveConfig() {
  const c = await GetAppConfig()
  c.mpServer = cfg.value.mpServer
  c.mpUid = cfg.value.mpUid
  c.mpPassword = cfg.value.mpPassword
  await SaveAppConfig(c)
  msg.value = '联机配置已保存'
}

async function launchMP() {
  launching.value = true
  try { await LaunchUnciv(); msg.value = 'Unciv 已启动' }
  catch (e: any) { msg.value = '启动失败: ' + e }
  finally { launching.value = false }
}

async function exportSnapshot() {
  try {
    const path = await ExportModSnapshot()
    msg.value = '快照已导出到 UMM 配置目录'
  } catch (e: any) { msg.value = '导出失败: ' + e }
}

async function openFolder() {
  try {
    await OpenSnapshotFolder()
  } catch (e: any) {
    msg.value = '打开目录失败: ' + e
  }
}

function pickSnapshot() { fileInput.value?.click() }

async function onSnapshotSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  loading.value = true; diffs.value = []
  try {
    const path = (file as any).path || file.name
    diffs.value = await CompareModSnapshot(path)
    msg.value = diffs.value.length === 0 ? '✅ 双方模组完全一致，可以联机' : `发现 ${diffs.value.length} 处不一致`
  } catch (e: any) { msg.value = '比对失败: ' + e }
  finally { loading.value = false; input.value = '' }
}

function issueLabel(issue: string): string {
  const m: Record<string, string> = {
    missing_in_remote: '对方缺少',
    missing_in_local: '本地缺少',
    version_mismatch: '版本不同',
  }
  return m[issue] || issue
}
</script>

<template>
  <div class="mp-view view-card">
    <div class="view-header"><h1>🔗 联机检查</h1></div>
    <p class="subtitle">确保联机双方 Unciv 版本和模组一致</p>

    <div class="config-section">
      <h3>⚙️ 联机配置</h3>
      <div class="config-row">
        <input v-model="cfg.mpServer" placeholder="服务器地址" />
        <input v-model="cfg.mpUid" placeholder="UID" />
        <input v-model="cfg.mpPassword" placeholder="联机密码" type="password" />
        <button class="btn-go" @click="saveConfig">保存</button>
      </div>
      <button class="btn-launch" :disabled="launching" @click="launchMP">{{ launching ? '启动中...' : '▶ 启动 Unciv 联机' }}</button>
    </div>

    <div class="step-row">
      <div class="step-card">
        <h3>📤 导出本机快照</h3>
        <p>生成模组清单文件，发给联机对象</p>
        <button class="btn-go" @click="exportSnapshot">导出快照</button>
        <button class="btn-go outline" style="margin-left:8px" @click="openFolder">📂 打开目录</button>
      </div>
      <div class="step-card">
        <h3>📥 导入对方快照</h3>
        <p>选择对方发来的快照文件进行比对</p>
        <button class="btn-go outline" @click="pickSnapshot" :disabled="loading">{{ loading ? '比对中...' : '选择文件比对' }}</button>
        <input ref="fileInput" type="file" accept=".json" style="display:none" @change="onSnapshotSelected" />
      </div>
    </div>

    <div v-if="msg" class="toast" :class="{ ok: diffs.length === 0 && msg.includes('✅') }">{{ msg }}</div>

    <div v-if="diffs.length > 0" class="diff-list">
      <div v-for="(d, i) in diffs" :key="i" class="diff-row" :class="d.issue">
        <span class="diff-mod">{{ d.mod }}</span>
        <span class="diff-issue">{{ issueLabel(d.issue) }}</span>
        <span class="diff-ver">本机: {{ d.valueA || '-' }} | 对方: {{ d.valueB || '-' }}</span>
      </div>
    </div>

    <div class="tips">
      <h3>联机步骤</h3>
      <ol>
        <li>双方导出快照 → 比对确保一致</li>
        <li>主机在 Unciv 中创建多人游戏 → 获取房间 ID</li>
        <li>主机把房间 ID 发给参与者</li>
        <li>参与者在 Unciv 中输入房间 ID + 密码 → 加入游戏</li>
        <li>服务器地址: bj.unciv.cn</li>
      </ol>
    </div>
  </div>
</template>

<style scoped>
.mp-view { height: 100%; max-width: 700px; }
.view-header h1 { font-size: 26px; font-weight: 700; margin-bottom: 6px; color: var(--text-primary); }
.subtitle { color: var(--text-secondary); font-size: 15px; margin-bottom: 20px; }

.config-section { background: var(--bg-card); border-radius: 8px; padding: 16px 20px; margin-bottom: 20px; box-shadow: var(--card-shadow); }
.config-section h3 { font-size: 17px; font-weight: 700; margin-bottom: 10px; }
.config-row { display: flex; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.config-row input { padding: 8px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary); font-size: 15px; flex: 1; min-width: 120px; }
.btn-launch { width: 100%; padding: 12px; background: var(--success); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; font-weight: 700; }
.btn-launch:hover { opacity: 0.9; }
.btn-launch:disabled { opacity: 0.5; cursor: not-allowed; }

.step-row { display: flex; gap: 16px; margin-bottom: 20px; }
.step-card { flex: 1; background: var(--bg-card); border-radius: 8px; padding: 20px; box-shadow: var(--card-shadow); }
.step-card h3 { font-size: 17px; font-weight: 700; margin-bottom: 6px; }
.step-card p { font-size: 15px; color: var(--text-secondary); margin-bottom: 14px; }
.btn-go { padding: 9px 20px; background: var(--accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 15px; font-weight: 600; }
.btn-go:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-go.outline { background: transparent; border: 1px solid var(--accent); color: var(--accent); }

.toast { padding: 10px 14px; border-radius: 4px; margin-bottom: 16px; font-size: 15px; background: rgba(255,107,107,0.08); color: var(--danger); }
.toast.ok { background: rgba(82,196,26,0.08); color: var(--success); }

.diff-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 20px; }
.diff-row { padding: 12px 14px; border-radius: 4px; background: var(--bg-card); display: flex; align-items: center; gap: 12px; font-size: 15px; }
.diff-row.missing_in_remote { border-left: 3px solid var(--warning); }
.diff-row.missing_in_local { border-left: 3px solid var(--danger); }
.diff-row.version_mismatch { border-left: 3px solid var(--accent); }
.diff-mod { font-weight: 700; min-width: 140px; }
.diff-issue { font-size: 14px; padding: 2px 8px; border-radius: 3px; font-weight: 600; }
.diff-row.missing_in_remote .diff-issue { background: rgba(250,173,20,0.15); color: var(--warning); }
.diff-row.missing_in_local .diff-issue  { background: rgba(255,77,79,0.15); color: var(--danger); }
.diff-row.version_mismatch .diff-issue { background: rgba(74,158,255,0.15); color: var(--accent); }
.diff-ver { font-size: 14px; color: var(--text-primary); margin-left: auto; }

.tips { background: var(--bg-card); border-radius: 8px; padding: 16px 20px; box-shadow: var(--card-shadow); }
.tips h3 { font-size: 17px; font-weight: 700; margin-bottom: 8px; }
.tips ol { font-size: 15px; color: var(--text-primary); padding-left: 18px; line-height: 1.8; }
.tips li { padding: 3px 0; }
</style>
