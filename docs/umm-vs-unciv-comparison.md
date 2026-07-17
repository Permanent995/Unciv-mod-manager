# UMM 与 Unciv 内置功能对比分析

**日期**: 2026-07-16  
**Unciv 版本**: 4.21.1  
**分析范围**: Unciv 内置 vs UMM 实现的模组管理/诊断/冲突检测功能

---

## 一、Unciv 内置的模组管理架构

从 jar 反编译发现，Unciv 4.21.1 已经内置了完整的模组管理子系统：

```
ModManagementScreen（主界面）
├── ModInfoAndActionPane（模组详情/操作面板）
│   ├── update(Ruleset) → 调用 ModCompatibility.isAudioVisualMod()
│   │   └── 判断音视频模组 → ⚠️ 崩溃点
│   └── 下载/卸载/复制链接 操作
├── ModManagementOptions
│   ├── Filter: 按类型/状态过滤
│   └── SortType: 按名称/日期/星星/状态排序
└── 两栏 Tab：已安装 | 在线浏览
```

## 二、Unciv 内置的模组诊断架构

```
RulesetValidator（规则集验证器）
├── UniqueValidator（Unique 语法验证）
│   ├── checkUnique() — 验证单个 Unique
│   ├── checkUniques() — 遍历所有 Unique
│   └── checkUntypedUnique() — 处理无类型 Unique
│
├── ModCompatibility（模组兼容性判断）
│   ├── isAudioVisualMod() — ⚠️ 崩溃在此
│   ├── isExtensionMod() — ⚠️ 崩溃在此
│   └── isBaseRuleset() 
│
├── BaseRulesetValidator（基础规则集验证）
│
├── RulesetErrorList（错误列表）
│   └── getErrorText() — 格式化错误文本
│
└── RulesetErrorSeverity（严重级别 — 5级）
    ├── OK
    ├── WarningOptionsOnly  ← 仅在模组选项中显示
    ├── Warning
    ├── ErrorOptionsOnly    ← 仅在模组选项中显示
    └── Error
```

## 三、功能逐一对比

### 3.1 模组浏览与管理

| 功能 | Unciv 实现 | UMM 实现 | 重叠度 |
|------|-----------|---------|--------|
| 已安装模组列表 | `ModManagementScreen` 左栏 | `ModsView.vue` (428行, 34个后端调用) | 🔴 100% 重叠 |
| 模组详情(README/预览) | `ModInfoAndActionPane.update()` | `ModsView.vue` 右侧面板 | 🔴 100% 重叠 |
| 模组备份/还原 | 无 | `ModsView.vue` (BackupMod/RestoreBackup/DeleteBackup) | 🟢 UMM 独有 |
| 在线浏览 | 右栏 (GitHub API 搜索) | `BrowseView.vue` (221行, 26个后端调用) | 🔴 100% 重叠 |
| README 翻译 | 无 | `BrowseView.vue` (TranslateText, 3种后端) | 🟢 UMM 独有 |
| 下载模组 | `ModInfoAndActionPane` (直连) | `DownloadsView.vue` (271行, 34后端调用, 6镜像+断点续传+队列) | 🟡 UMM 增强 |
| 批量更新检查 | 调用 GitHub API | `updater.go` (CheckModUpdates + DownloadAllUpdates) | 🟡 实现不同 |
| 模组排序过滤 | `SortType` + `Filter` (5种排序, 8种分类) | `ModsView.vue` A-Z/类型/大小 排序+分类筛选 | 🟡 80% 重叠 |
| Unciv版本显示 | 无 | `Sidebar.vue` (GetUncivVersion 从 jar manifest) | 🟢 UMM 独有 |

### 3.2 模组诊断

| 功能 | Unciv 实现 | UMM 实现 | 重叠度 |
|------|-----------|---------|--------|
| Unique 废弃语法检查 | `UniqueValidator.checkUnique()` — 内置在 Unciv 引擎中，运行时自动检查 | `diagnose.go` + `deprecated.go` — 手动维护的规则列表 | 🔴 100% 重叠 |
| 严重级别 | `RulesetErrorSeverity` (OK/Warning/Error/OptionsOnly) | `DiagIssue.severity` ("error"/"warning") | 🔴 90% 重叠 |
| 错误定位 | `RulesetErrorList.getErrorText()` 显示文件和行号 | `DiagIssue.message` 人工描述 | 🟡 UMM 消息更友好 |
| 缺失类型引用检查 | `BaseRulesetValidator` (unitType/tech/resource) | `conflict.go` 步4 | 🔴 100% 重叠 |
| 初始化时自动运行 | ✅ Unciv 启动时自动检查 | ❌ UMM 需要用户手动点击 | — |

### 3.3 冲突检测

| 功能 | Unciv 实现 | UMM 实现 | 重叠度 |
|------|-----------|---------|--------|
| 同名实体覆盖 | ❓ 未在反编译代码中找到独立实现 | `conflict.go` 步3（加载顺序分析） | 🟢 UMM 独有 |
| replaces 链分析 | ❓ 未找到 | `conflict.go` 步4b（跨模组 replaces） | 🟢 UMM 独有 |
| MergeAction 感知 | ❓ 内置引擎处理，不暴露给用户 | `conflict.go` 步2b（TRY_INJECT vs CREATE_OR_REPLACE） | 🟢 UMM 独有 |
| Diff 差异计算 | ❌ 无 | `conflict.go` 步3b（数值字段对比） | 🟢 UMM 独有 |

### 3.4 辅助工具

| 功能 | Unciv 实现 | UMM 实现 | 重叠度 |
|------|-----------|---------|--------|
| 崩溃报告 | ❌ 只有 CrashScreen 和 lasterror.txt | `crash_reporter.go` (9种错误模式) | 🟢 UMM 独有 |
| 存档浏览 | ❌ 只能加载/保存 | `SavesView.vue` (详情/模组列表/删除) | 🟢 UMM 独有 |
| 联机检查 | ❌ 只有内置 MP 系统 | `MultiplayerView.vue` (快照导出/比对) | 🟢 UMM 独有 |
| 地图管理 | ❌ 只能通过 mod 加载 | `MapsView.vue` (扫描/导入/Wesnoth转换) | 🟢 UMM 独有 |
| 下载加速 | ❌ 只有直连 | `downloader.go` (6镜像+测速+断点续传) | 🟢 UMM 独有 |
| README 翻译 | ❌ 无 | `translate.go` (微软/Yandex/自定义AI) | 🟢 UMM 独有 |

---

## 四、总结

### 完全重叠的功能（UMM 应考虑移除/精简）

```
🔴 模组浏览与管理 — Unciv 4.x 已内置完整的 ModManagementScreen
   - 已安装模组列表 + 详情
   - 在线浏览 + 搜索
   - 排序过滤
   - 下载安装

🔴 Unique 废弃语法检查 — Unciv 内置 UniqueValidator，运行时自动运行
   - 比 UMM 的硬编码规则列表更准确（直接从引擎读取 @Deprecated 注解）
   - 自动更新（跟随 Unciv 版本，不需要手动同步）

🔴 缺失类型引用检查 — BaseRulesetValidator 已实现
```

### UMM 独有的价值（应保留）

```
🟢 冲突检测引擎 — 三步检测（同名实体+replaces链+MergeAction感知+Diff差异）
   Unciv 没有独立的用户界面来展示这些

🟢 崩溃报告 — 9种错误模式匹配，人工友好提示
   Unciv 只有原始 lasterror.txt

🟢 存档浏览 — 查看存档详情/模组列表/删除
   Unciv 只能加载/保存

🟢 下载加速 — 6个镜像站测速+断点续传
   Unciv 只有直连下载

🟢 联机检查 — 快照导出/比对
   Unciv 的内置 MP 不做模组一致性检查

🟢 地图管理 — 导入/Wesnoth转换
   Unciv 只能通过模组加载地图
```

### 建议

| 优先级 | 操作 | 原因 |
|--------|------|------|
| 🔴 P0 | 移除 `diagnose.go` 的 Unique 废弃检查 | Unciv 内置且更准确，手动同步规则无法跟上 Unciv 版本迭代 |
| 🔴 P0 | 移除 `deprecated.go` 规则列表 | 已过时且不完整（仅 ~15 条，Unciv 有数百条 @Deprecated 注解） |
| 🟡 P1 | 模组管理功能降级 | ModsView/BrowseView 作为 Unciv 内置界面的补充，而非替代 |
| 🟢 P2 | 保留下载加速 | 镜像下载是 UMM 核心价值 |
| 🟢 P2 | 保留冲突检测 | Unciv 没有可视化冲突报告 |
| 🟢 P2 | 保留存档/联机/地图/崩溃报告 | 全部是 Unciv 不具备的功能 |

---

## 五、技术债务

1. **deprecated.go 规则已过时**：仅 ~15 条规则，无法覆盖 Unciv 4.x 的所有 @Deprecated unique（数以百计）。每个 Unciv 新版本都可能增删废弃语法，UMM 无法跟上。

2. **diagnose.go 的逻辑是 UniqueValidator 的子集**：`checkDeprecatedIn()` 做简单的字符串匹配（`strings.Contains`），而 Unciv 的 `UniqueValidator.checkUnique()` 做完整语法解析（参数类型匹配、条件分支检查等）。

3. **维护负担**：每发布一个新 Unciv 版本，都需要手动审查 Kotlin 源码中的 @Deprecated 注解来更新 deprecatedRules 列表。
