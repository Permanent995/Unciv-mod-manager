# UMM 全面审查报告

**审查日期**: 2026-07-16
**审查范围**: Go 后端（15文件）、Vue 前端（12组件）、项目配置、前后端联动
**原则**: 只报告问题，不修改代码

---

## 一、严重问题（必须修复）

### 1. 🔴 backup.go: `_backup_info.json` 残留到 mods/ 目录

**文件**: `internal/app/backup.go:45`

**问题**: `BackupMod()` 将模组目录整体 Rename 到 `umm_backups/`，然后在备份目录中写入 `_backup_info.json`。当 `RestoreBackup()` 执行 `os.Rename(backupPath, target)` 时，`_backup_info.json` 会被一起移回 `mods/` 目录。

**影响**: Unciv 的 `ModCompatibility` 类会扫描模组目录下的所有文件，遇到非标准的 `_backup_info.json` 可能触发解析异常。

**证据**: 已在 `mods/RekMOD-3.4/` 中发现 `_backup_info.json` 残留文件。`umm_backups/` 目录中仍有 3 个备份。

**修复建议**: `RestoreBackup()` 中 `os.Rename` 后应删除 `_backup_info.json`，或在 `BackupMod()` 中将元数据写入独立的索引文件而非备份目录内。

### 2. 🔴 backup.go: `RestoreBackup()` 先 Rename 再 RemoveAll 的竞态条件

**文件**: `internal/app/backup.go:104-110`

```go
// 当前代码：
a.BackupMod(modFolder, currentVersion)  // Rename 当前版本到备份
os.RemoveAll(target)                     // 删除已移走的路径（空操作）
return os.Rename(backupPath, target)     // 恢复旧版本
```

**问题**: 第 107 行 `BackupMod` 用 `os.Rename` 把当前版本移走了，第 108 行 `os.RemoveAll(target)` 删的是一个不存在的路径（空操作）。如果 `BackupMod` 失败（比如跨盘 Rename），当前版本就丢了。

**修复建议**: `os.RemoveAll` 应在 `BackupMod` 之前执行，或改用 copy+delete 替代 Rename。

### 3. 🔴 Unciv.json: vmArgs 末尾有孤立的 `"-"`

**文件**: `C:\Users\drj13\Desktop\官方unciv文件\Unciv.json`

```json
"vmArgs": ["-Xmx4G", "-"]
```

**问题**: `"-"` 不是有效的 JVM 参数。Java 会把它当作主类名处理，可能导致 JVM 启动行为异常。虽然这个文件不是 UMM 生成的，但 UMM 的 `GetUncivVersion()` 读取 `Unciv.jar` 时如果 jar 正在被异常的 JVM 使用，可能产生连锁问题。

---

## 二、一般问题（建议修复）

### 4. 🟡 downloader.go: `dlTasks` map 无并发上限保护

**文件**: `internal/app/downloader.go`

**问题**: `pruneOldTasks()` 在超过 50 条时清理，但如果短时间内创建大量任务（如 `DownloadAllUpdates`），map 可能临时膨胀。

**修复建议**: 在 `StartDownload` 入口检查当前 downloading 状态的任务数。

### 5. 🟡 downloader.go: `speedSamples` 切片无上限

**文件**: `internal/app/downloader.go`

**问题**: 3 秒滑动窗口通过 `calculateSpeed` 清理过期样本，但如果进度更新频率极高，切片可能短暂膨胀。

### 6. 🟡 scanner.go: `dirSize()` 对大模组性能差

**文件**: `internal/app/scanner.go:234-244`

**问题**: 每次 `ScanMods` 都会对所有没有 `modSize` 字段的模组执行 `filepath.Walk`，大模组（如 500MB 的贴图模组）可能耗时数秒。

**修复建议**: 缓存模组大小，只在 `ModOptions.json` 的修改时间变化时重新计算。

### 7. 🟡 scanner.go: `hasOnlyImageFiles()` 和 `hasMusicFiles()` 重复遍历

**文件**: `internal/app/scanner.go:168-204`

**问题**: `categorizeMod` 依次调用 `hasOnlyImageFiles` 和 `hasMusicFiles`，每个都独立 `filepath.Walk` 整个模组目录。可以合并为一次遍历。

### 8. 🟡 game_link.go: `GetUncivVersion` 在 `onMounted` 时调用

**文件**: `frontend/src/components/Sidebar.vue:32-36`

**问题**: 侧边栏组件挂载时调用 `GetUncivVersion()`，该函数用 `zip.OpenReader` 打开 `Unciv.jar`。虽然 Go 的 `archive/zip` 使用共享读取，但如果 Unciv 同时正在启动并锁定 jar 文件，可能导致短暂阻塞。

**修复建议**: 缓存版本号到 AppConfig，只在路径变更时重新读取。

### 9. 🟡 前端: 多个组件缺少错误边界

**文件**: `ModsView.vue`, `SavesView.vue`, `BrowseView.vue`

**问题**: Wails 绑定函数调用失败时（如路径未设置），部分组件没有 try/catch 或错误状态 UI，直接静默失败。

### 10. 🟡 前端: CSS 变量体系有冗余

**文件**: `App.vue`

**问题**: `:root` 定义了 LYT 风格的变量，然后 `[data-theme="light"]` 和 `[data-theme="dark"]` 又定义了 UMM 原有变量名，通过"兼容别名层"映射。两套变量名增加了维护复杂度。

### 11. 🟡 app_test.go: 测试覆盖不足

**当前 17 个测试**，缺少以下关键功能的测试：
- `ExtractMod()` 解包逻辑（目录展平、危险文件过滤）
- `ScanMods()` 模组扫描
- `BackupMod()` / `RestoreBackup()` 备份恢复（特别是 `_backup_info.json` 残留问题）
- `CheckModUpdates()` 更新检查
- `AnalyzeConflicts()` 冲突检测的核心逻辑

---

## 三、小问题（可选）

### 12. ⚪ backup.go: `readModVersion` 路径假设

**文件**: `internal/app/backup.go:124-131`

只检查 `jsons/ModOptions.json`，不检查根目录的 `ModOptions.json`。部分模组（如纯图形模组）可能没有 `jsons/` 目录。

### 13. ⚪ extractor.go: 无文件覆盖保护

**文件**: `internal/app/extractor.go:66-106`

解包时直接 `os.Create(outPath)` 覆盖已有文件，不检查目标文件是否已存在。对于模组更新场景是期望行为，但可能导致意外覆盖。

### 14. ⚪ 前端: ToolboxView 的拖拽手柄未清理 mousemove 监听

**文件**: `frontend/src/components/ToolboxView.vue`

`startCatDrag` 注册了 `mousemove`/`mouseup` 事件，但如果用户在拖拽过程中切换页面，`mousemove` 监听可能残留。

### 15. ⚪ app.go: UMMVersion 硬编码为 "1.0.0"

**文件**: `internal/app/app.go:13`

tag 已打 v1.1，但代码中 `UMMVersion = "1.0.0"` 未同步更新。

---

## 四、前后端联动审查

### 绑定完整性 ✅

| 检查项 | 结果 |
|--------|------|
| App.d.ts 声明数 | 58 个函数 |
| Go 暴露函数数 | 58 个（全部匹配） |
| 类型不匹配 | 0 个 |
| 前端调用但后端未暴露 | 0 个 |
| 后端暴露但前端未使用 | `BuildDownloadURL`, `ValidateDownloadURL`, `StartDownloadWithMirror` 等 3 个（可在前端简化后删除） |

### 事件通信 ✅

| 后端 EventsEmit | 前端 EventsOn | 匹配 |
|-----------------|---------------|------|
| `download:progress` | DownloadsView.vue | ✅ |
| `download:complete` | DownloadsView.vue | ✅ |

### 数据流完整性 ✅

- `ScanMods()` → `ModInfo[]` → 前端 `ModsView.vue` ✅
- `ScanSaves()` → `SaveInfo[]` → 前端 `SavesView.vue` ✅
- `AnalyzeConflicts()` → `ConflictReport[]` → 前端 `ToolboxView.vue` ✅
- `SearchOnlineMods()` → `OnlineMod[]` → 前端 `BrowseView.vue` ✅
- `CheckModUpdates()` → `ModUpdateInfo[]` → 前端 `ModsView.vue` ✅

---

## 五、UMM 写入 Unciv 目录的完整操作清单

| 操作 | 写入位置 | 触发函数 | 是否清理 |
|------|---------|---------|---------|
| 解压模组 | `mods/模组名/` | `ExtractMod()` | ❌ 用户删除模组时才清理 |
| 备份模组 | `umm_backups/` | `BackupMod()` | ❌ 永久保留 |
| 恢复备份 | `mods/模组名/` (含 `_backup_info.json`) | `RestoreBackup()` | ⚠️ **`_backup_info.json` 残留** |
| 删除模组 | `mods/模组名/` | `DeleteMod()` | ✅ 完全删除 |
| 导入地图 | `maps/` | `ImportFile()` | ❌ 用户删除时才清理 |
| Wesnoth 转换 | `maps/*.civ5map` + 原文件 `.bak` | `ConvertWesnothMap()` | ⚠️ `.bak` 备份文件残留 |
| 删除存档 | `SaveFiles/` | `DeleteSave()` | ✅ |
| 写入配置 | `%APPDATA%/UncivModManager/config.json` | `saveConfig()` | ✅ UMM 独占 |
| 下载临时文件 | `%TEMP%/unciv-mm-downloads/` | `StartDownload()` | ⚠️ 依赖前端调用 `CleanupTempFile` |

---

## 六、Unciv 崩溃关联分析

### 最可能的原因（按可能性排序）

#### 1. `_backup_info.json` 残留导致 ModCompatibility 解析异常 — **可能性: 中**

**证据**:
- `backup.go:45` 在备份目录中写入 `_backup_info.json`
- `RestoreBackup()` 将整个目录（含 `_backup_info.json`）Rename 回 `mods/`
- 已在 `mods/RekMOD-3.4/` 中发现残留的 `_backup_info.json`
- Unciv 的 `ModCompatibility.isAudioVisualMod()` 会遍历模组目录

**反驳**: `NoClassDefFoundError` 是 JVM 类加载错误，不是文件解析错误。`_backup_info.json` 可能导致模组验证失败，但不应该导致 Kotlin 基础类加载失败。

#### 2. Unciv 自身的 ModCompatibility 类 bug — **可能性: 高**

**证据**:
- 命令行直接加载 `Intrinsics` 和 `ModCompatibility` 均成功
- 仅在通过 `ModCheckTab` / `ModCheckboxTable` 调用链时失败
- 4.20.12 和 4.21.1 两个版本都复现
- 这是 Kotlin 编译器和 fat jar 打包的问题，不是外部文件干扰

**但用户观察到"之前没问题"**:
- 可能之前从未触发过 `ModCompatibility` 路径（不点模组详情、不点开始新游戏）
- 或者之前的 Unciv 版本没有这个 bug

#### 3. Unciv.json 的 vmArgs 异常 — **可能性: 低-中**

**证据**:
- `"vmArgs": ["-Xmx4G", "-"]` 中 `"-"` 是无效参数
- 可能导致 JVM 模块系统行为异常
- 但该文件是 Unciv 自带的，UMM 未修改过

### 结论

**UMM 的 `_backup_info.json` 残留是一个确认的 bug**，但它不太可能直接导致 `NoClassDefFoundError`。Unciv 崩溃更可能是 Unciv 自身的 `ModCompatibility` 类在特定运行时上下文中的类加载器问题。

**建议**:
1. 修复 `_backup_info.json` 残留 bug（不管是否与崩溃相关）
2. 修复 `RestoreBackup()` 的竞态条件
3. 删除 `Unciv.json` 中 vmArgs 的 `"-"`
4. 向 Unciv 提交 Issue 报告 ModCompatibility 崩溃

---

## 七、模块健康度评分

| 模块 | 可行性 | 一致性 | 简洁性 | 可维护性 | 总分 |
|------|--------|--------|--------|----------|------|
| app.go | 9 | 8 | 8 | 7 | **8.0** |
| scanner.go | 9 | 9 | 7 | 8 | **8.3** |
| downloader.go | 8 | 8 | 7 | 7 | **7.5** |
| extractor.go | 9 | 9 | 9 | 8 | **8.8** |
| conflict.go | 8 | 8 | 7 | 7 | **7.5** |
| backup.go | **5** | 8 | 8 | 7 | **7.0** |
| game_link.go | 8 | 8 | 8 | 8 | **8.0** |
| saves.go | 9 | 9 | 9 | 8 | **8.8** |
| github_api.go | 8 | 8 | 7 | 7 | **7.5** |
| updater.go | 8 | 8 | 8 | 8 | **8.0** |
| selfupdate.go | 8 | 8 | 8 | 8 | **8.0** |
| crash_reporter.go | 8 | 8 | 8 | 8 | **8.0** |
| multiplayer.go | 8 | 8 | 8 | 8 | **8.0** |
| map_importer.go | 8 | 8 | 8 | 8 | **8.0** |
| wesnoth.go | 9 | 9 | 9 | 8 | **8.8** |

**最需要改进**: `backup.go`（`_backup_info.json` 残留 + 竞态条件）
**最健康**: `extractor.go`, `saves.go`, `wesnoth.go`

---

## 八、前端组件健康度评分

| 组件 | 可行性 | 一致性 | 简洁性 | 可维护性 | 总分 |
|------|--------|--------|--------|----------|------|
| App.vue | 8 | 7 | 6 | 6 | **6.8** |
| Sidebar.vue | 9 | 9 | 9 | 8 | **8.8** |
| TitleBar.vue | 9 | 9 | 9 | 9 | **9.0** |
| ModsView.vue | 8 | 8 | 7 | 7 | **7.5** |
| MapsView.vue | 8 | 8 | 8 | 8 | **8.0** |
| SavesView.vue | 8 | 8 | 8 | 8 | **8.0** |
| DownloadsView.vue | 8 | 8 | 7 | 7 | **7.5** |
| BrowseView.vue | 8 | 8 | 7 | 7 | **7.5** |
| MultiplayerView.vue | 8 | 8 | 8 | 8 | **8.0** |
| ToolboxView.vue | 7 | 8 | 6 | 6 | **6.8** |
| SettingsView.vue | 8 | 8 | 8 | 8 | **8.0** |
| AboutView.vue | 9 | 9 | 8 | 8 | **8.5** |

**最需要改进**: `App.vue`（CSS 变量冗余）、`ToolboxView.vue`（过大、拖拽手柄内存泄漏风险）
**最健康**: `TitleBar.vue`, `Sidebar.vue`, `AboutView.vue`

---

## 九、项目整体评估

**整体健康度: 7.8 / 10**

**优势**:
- 代码量精简（~3800 行），功能完整
- 前后端绑定完整，无遗漏
- 安全意识好（危险扩展名过滤、zip-slip 防护、路径遍历保护）
- 镜像加速和断点续传实现可靠

**最需要改进的 Top 5**:
1. `backup.go` 的 `_backup_info.json` 残留到 mods/ 目录
2. `backup.go` 的 `RestoreBackup()` 竞态条件
3. 测试覆盖不足（仅 17 个测试，缺少备份/解包/冲突核心功能测试）
4. `ToolboxView.vue` 组件过大（>800 行），应拆分
5. `App.vue` 的 CSS 变量体系应简化，去掉兼容别名层
