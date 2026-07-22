<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { CheckSelfUpdate, DownloadSelfUpdate, InstallSelfUpdate, GetUMMVersion } from '../../wailsjs/go/app/App'
import { BrowserOpenURL, Quit, EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const ummVersion = ref('')

onMounted(async () => {
  try {
    ummVersion.value = await GetUMMVersion()
  } catch { /* ignore */ }

  // Listen for download completion to auto-unlock the install button
  EventsOn('download:complete', (payload: any) => {
    const fn = (payload.filename || '').toLowerCase()
    if (fn === 'umm_update.zip' || fn === 'umm_update.exe') {
      downloadCompleted.value = true
      downloadMsg.value = '✅ 下载完成，可以安装了'
    }
  })
})

onUnmounted(() => {
  EventsOff('download:complete')
})

const checking = ref(false)
const updateMsg = ref('')
const updateInfo = ref<{ currentVersion: string; latestVersion: string; downloadUrl: string; hasUpdate: boolean; releaseName: string; cachedAt?: string } | null>(null)
const downloading = ref(false)
const downloadMsg = ref('')
const downloadQueued = ref(false)
const downloadCompleted = ref(false)
const installing = ref(false)
const installMsg = ref('')

async function checkUpdate() {
  checking.value = true
  updateMsg.value = ''
  updateInfo.value = null
  downloadMsg.value = ''
  downloadQueued.value = false
  downloadCompleted.value = false
  installMsg.value = ''
  try {
    const info = await CheckSelfUpdate()
    updateInfo.value = info
    if (info.cachedAt) {
      const t = new Date(info.cachedAt).toLocaleString()
      if (info.hasUpdate) {
        updateMsg.value = `发现新版本 v${info.latestVersion}（离线缓存，上次检查 ${t}）`
      } else {
        updateMsg.value = `当前已是最新版本（离线缓存，上次检查 ${t}）`
      }
    } else if (info.hasUpdate) {
      updateMsg.value = `发现新版本 v${info.latestVersion}！`
    } else {
      updateMsg.value = '当前已是最新版本'
    }
  } catch (e: any) {
    updateMsg.value = '检查失败: ' + e
  } finally {
    checking.value = false
  }
}

async function downloadUpdate() {
  if (!updateInfo.value?.downloadUrl) return
  downloading.value = true
  downloadMsg.value = ''
  downloadQueued.value = false
  downloadCompleted.value = false
  installMsg.value = ''
  try {
    await DownloadSelfUpdate(updateInfo.value.downloadUrl)
    downloadQueued.value = true
    downloadMsg.value = '⬇ 已加入下载队列，完成后自动出现安装按钮'
  } catch (e: any) {
    downloadMsg.value = '下载失败: ' + e
  } finally {
    downloading.value = false
  }
}

async function installUpdate() {
  installing.value = true
  installMsg.value = ''
  try {
    const result = await InstallSelfUpdate()
    if (result.restartRequired) {
      installMsg.value = '更新已安装！请关闭 UMM 后重新启动以运行新版本'
    }
  } catch (e: any) {
    installMsg.value = '安装失败: ' + e
  } finally {
    installing.value = false
  }
}

function openURL(url: string) {
  BrowserOpenURL(url)
}
</script>

<template>
  <div class="about-page view-card">
    <section class="about-hero">
      <div class="about-logo-mark">
        <img src="../assets/icon.png" class="about-logo" />
      </div>
      <div class="about-hero-copy">
        <h2>Unciv Mod Manager</h2>
        <p>轻量、清爽的 Unciv 模组管理工具</p>
      </div>
    </section>

    <section class="about-grid" aria-label="应用信息">
      <div class="about-info-panel">
        <div class="about-panel-title">项目信息</div>
        <dl class="about-meta-list">
          <div><dt>当前版本</dt><dd>v{{ ummVersion }}</dd></div>
          <div><dt>开源协议</dt><dd>GPL-3.0</dd></div>
          <div><dt>项目仓库</dt><dd>Permanent995/Unciv-mod-manager</dd></div>
          <div><dt>问题反馈</dt><dd>GitHub Issues</dd></div>
        </dl>
      </div>

      <div class="about-info-panel">
        <div class="about-panel-title">操作</div>
        <div class="about-actions">
          <button class="about-action-btn primary" type="button" @click="openURL('https://github.com/Permanent995/unciv-mod-manager')">
            <span>打开 GitHub 仓库</span>
          </button>
          <button class="about-action-btn" type="button" @click="openURL('https://github.com/Permanent995/unciv-mod-manager/blob/master/LICENSE')">
            <span>查看开源协议</span>
          </button>
          <button class="about-action-btn" type="button" @click="openURL('https://github.com/Permanent995/unciv-mod-manager/issues/new')">
            <span>问题反馈</span>
          </button>
          <button class="about-action-btn" type="button" :disabled="checking" @click="checkUpdate">
            <span>{{ checking ? '检查中...' : '检查更新' }}</span>
          </button>
        </div>
        <div v-if="updateMsg" class="about-update-status" :class="{ success: updateInfo && !updateInfo.hasUpdate, error: updateMsg.includes('失败') }">
          {{ updateMsg }}
        </div>

        <div v-if="updateInfo?.hasUpdate" class="about-update-detail">
          <button class="about-action-btn primary" type="button" :disabled="downloading" @click="downloadUpdate">
            <span>{{ downloading ? '添加中...' : '⬇ 下载更新' }}</span>
          </button>
          <div v-if="downloadMsg" class="about-update-status" :class="{ success: downloadCompleted }">
            {{ downloadMsg }}
          </div>
          <button v-if="downloadCompleted" class="about-action-btn primary" type="button" :disabled="installing" @click="installUpdate" style="margin-top:8px">
            <span>{{ installing ? '安装中...' : '⬆ 安装更新（自动解压替换，完成后重启即可）' }}</span>
          </button>
          <div v-if="installMsg" class="about-update-status" :class="{ success: installMsg.includes('更新已安装'), error: installMsg.includes('失败') }">
            {{ installMsg }}
          </div>
          <div v-if="installMsg.includes('更新已安装')" class="about-rollback-hint">
            🔄 旧版已备份为 .bak，如有异常可删除新版 exe 并去掉 .bak 后缀恢复
            <button class="about-action-btn primary" type="button" @click="Quit" style="margin-top:8px">
              <span>🔁 退出并重启 UMM</span>
            </button>
          </div>
        </div>
      </div>
    </section>

    <section class="about-features">
      <h3>功能介绍</h3>
      <div class="about-feature-list">
        <span>模组扫描与分类</span><span>冲突检测与分析</span><span>镜像加速下载</span>
        <span>在线浏览与更新检查</span><span>崩溃报告查看</span><span>联机快照比对</span>
      </div>
    </section>
  </div>
</template>

<style scoped>
.about-page {
  max-width: 900px;
}

.about-hero {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 20px;
  align-items: center;
  padding: 28px;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(79,70,229,0.08), rgba(59,130,246,0.04)), var(--bg-primary);
  margin-bottom: 16px;
}

.about-logo-mark {
  width: 72px;
  height: 72px;
  display: grid;
  place-items: center;
  border-radius: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  box-shadow: var(--card-shadow);
}

.about-logo {
  width: 48px;
  height: 48px;
  object-fit: contain;
}

.about-hero-copy { min-width: 0; }

.about-kicker {
  color: var(--accent);
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.about-hero h2 {
  margin: 4px 0 0;
  font-size: 28px;
  font-weight: 800;
  line-height: 1.15;
}

.about-hero p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  font-size: 15px;
}

.about-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.about-info-panel {
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: var(--bg-primary);
}

.about-panel-title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}

.about-meta-list {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin: 0;
}

.about-meta-list div {
  display: grid;
  gap: 2px;
}

.about-meta-list dt {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-muted);
}

.about-meta-list dd {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.about-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.about-action-btn {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 14px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-card);
  color: var(--text-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s;
}

.about-action-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
  transform: translateY(-1px);
}

.about-action-btn.primary {
  grid-column: 1 / -1;
  border-color: transparent;
  background: linear-gradient(135deg, var(--accent), #6366f1);
  color: #fff;
}

.about-action-btn.primary:hover {
  opacity: 0.9;
  transform: translateY(-1px);
  border-color: transparent;
  color: #fff;
}

.about-action-btn:disabled {
  opacity: 0.6;
  cursor: default;
  transform: none;
}

.about-update-status {
  margin-top: 12px;
  font-size: 14px;
  color: var(--text-muted);
}

.about-update-status.success { color: var(--success); }
.about-update-status.error { color: var(--danger); }

.about-update-detail { margin-top: 10px; }

.about-rollback-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px;
  background: var(--bg-card);
  border-radius: 6px;
}

.about-features {
  padding: 20px;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: var(--bg-primary);
}

.about-features h3 {
  font-size: 15px;
  font-weight: 700;
  margin-bottom: 12px;
}

.about-feature-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.about-feature-list span {
  padding: 4px 12px;
  background: var(--bg-card);
  border-radius: 6px;
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 600;
}
</style>
