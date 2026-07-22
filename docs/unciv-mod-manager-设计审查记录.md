# Unciv Mod Manager 方案审查记录

审查日期：2026-07-15
审查范围：LYT VPK 2.5.4 源码、Unciv 实际模组文件结构、GameSettings.json 实际内容、Wails v2 + Vue 3 技术可行性

---

## 参考架构：LYT VPK 2.5.4

| 项目 | 值 |
|------|-----|
| Wails 版本 | v2.10.2（go.mod 声明） |
| Go 模块名 | `vpk-manager` |
| 内部包数 | 5：`app`、`parser`、`network`、`platform/protocol`、`platform/urlregistry`、`minidump` |
| 前端框架 | 无，原生 JS + Vite 3，零框架依赖 |
| 前端文件数 | ~80 JS 文件，~15K-20K 行 |
| Wails 绑定 | 整个 `*App` struct 绑为 `any`，~120 方法暴露给前端 |
| 下载方案 | 5MB 分块 + `Truncate()` + `file.WriteAt()` 直接写偏移 |
| 配置存储 | JSON 文件在 `%AppData%/LytVPK/`，无数据库 |
| 并发池 | `panjf2000/ants/v2`，size = max(4, GOMAXPROCS) |
| 存档解析 | `l4d2-manager-next/pkg/valve/vpk`（替换为 `LaoYutang/l4d2-server-next/backend`） |
| 单例锁 | named pipe 互斥 |
| URL 协议 | `lytvpk://` 注册到 Windows 注册表 HKCU |

---

## 一、方案与实际的差异

### 1. GameSettings.json 结构完全不同（🔴 阻断级）

**方案假设**：`baseRuleset` 和 `mods` 在 JSON 顶层，管理器可以直接读写模组。

**实际情况**：
```json
{
  "tileSet": "5Hex",
  "unitSet": "5Hex",
  "lastGameSetup": {
    "gameParameters": {
      "baseRuleset": "Go-Astray",
      "mods": ["5Hex Tileset"]
    },
    "mapParameters": {
      "mods": ["5Hex Tileset", "unciv unrepentant cathay"]
    }
  }
}
```

- 规则集和扩展模组在深层嵌套对象里，不在顶层
- 这是"上次开局的配置"，不是"当前全局启用的模组"
- `gameParameters.mods` 和 `mapParameters.mods` 可以不同
- 只有 `tileSet`/`unitSet` 是全局实时的图形模组
- Unciv 没有"全局启禁用"概念，每次开局在游戏内选模组

**修复方案**：管理器改为**只读** GameSettings.json，不写。模组"开关"改为"标记关注"，存管理器自己的 `profiles.json`。玩家去游戏内选模组，管理器只做冲突分析。

### 2. Unciv 的 JSON 不是标准 JSON（🔴 阻断级）

**实际情况**：Go-Astray 的 `ModOptions.json` 有 trailing comma 和 `//` 注释：
```json
{
  "uniques": [ ... ],
  "isBaseRuleset": true,
}
```

**影响**：Go 标准库 `encoding/json` 无法解析。

**修复方案**：使用 `github.com/tidwall/gjson`（容错好，只读）作为解析器。

### 3. replaces 跨模组时仍需报告冲突（🔴 阻断级）

**修正**：同一模组内不同文明对同一实体做 `replaces` 不冲突（如 E&D 内多个文明各自替换 Courthouse）。但**不同模组**即使 `uniqueTo` 不同（如 Hodgepodge 和 unrepentant 都替换 Courthouse），加载顺序也会决定最终哪个生效——这是实打实的互盖问题，需要报告。

**修复方案**：replaces 检测逻辑加入 uniqueTo 维度，跨模组同目标替换仍然报告为互盖。

### 4. 下载方案改为 LYT 的 WriteAt 方案（🟡 优化）

**原方案**：分块→独立 .part 文件→io.Copy 合并（三步）
**LYT 方案**：`Truncate()` 预分配 → `file.WriteAt()` 直接写偏移量（一步，无合并）

`Truncate()` 是文件系统元数据操作，不占内存。每个 worker 持有一个 5MB buffer，下载完写盘就释放，不存在 OOM。

###  6. "懒作者"模组的兜底处理（🟡 容错）

**问题**：如果模组作者没写开源协议、Readme，甚至 ModOptions.json 信息不全怎么办？

**解决**：
- `name` 缺失 → 使用文件夹名作为显示名称
- `author` 缺失 → 显示"未提供"
- 无 `ModOptions.json` → 标记 `IsIncomplete: true`，分类为 `unclassified`，不报错跳过
- 检查 `README.md` 是否存在 → 记录到 `HasReadme` 字段，前端可用于提示"该模组无说明文档"
---

## 二、技术选型决策过程

| 决策项 | 备选 | 最终选择 | 理由 |
|--------|------|---------|------|
| 前端框架 | 原生 JS / Vue 3 | **Vue 3** | 组件化，比原生 JS 好维护 |
| 语言 | JavaScript / TypeScript | **TypeScript** | Wails 自动生成 .d.ts，跨语言类型安全 |
| UI 库 | 手写 CSS / Naive UI / Element Plus | **Element Plus** | 中文文档全、桌面端验证过、生态大 |
| 路由 | vue-router / v-if | **v-if 切换** | 6 页面平级无嵌套，桌面端无 URL 栏，无需路由（方案原定 vue-router，实际落地改为 v-if） |
| 状态管理 | Pinia / 组件 ref | **ref + prop** | 下载状态 Go 后端持有，EventsOn 推送增量更新，组件内 ref 足够（方案原定 Pinia，实际未使用） |
| 实体数据结构 | ECS 拆分组件 / 扁平 struct | **扁平 struct** | 量级小（几千实体），ECS 拆分徒增复杂度 |

---

## 三、Unciv 模组关键发现

来自 6 个样本模组的实际检查：
- 6 个模组中 **0 个** 有 `modDependencies`
- `replaces` 字段广泛使用，常与 `uniqueTo` 搭配
- Go-Astray 用点号命名风格：`Unit.Pioneer`、`Tech.Pipe making`
- `maps/` 顶级目录只有 `Autosave`（二进制 gzip），没有 .civ5map 文件
- JSON 实体确认字段：`name`、`unitType`、`cost`、`movement`、`strength`、`rangedStrength`、`range`、`requiredTech`、`obsoleteTech`、`upgradesTo`、`requiredResource`、`replaces`、`uniqueTo`、`promotions`、`uniques`、`attackSound`

MergeAction 五种操作类型（详见 [教程文档](C:\Users\drj13\unciv模组制作\UncivCN扩展JSON-MergeAction教程.md)）：无 action/CREATE_OR_REPLACE → 完整覆盖、TRY_INJECT → 字段级合并、REMOVE → 删除实体、REMOVE_FIELD → 移除字段、条件分支 if/then/else。

---

## 四、执行阶段调整记录

| 原始方案 | 调整后 | 原因 |
|---------|--------|------|
| Phase 1: 项目初始化 | Phase 1: 项目初始化 + 路径检测 | 路径检测是扫描模组的前提，不能放在 Day 5 |
| Phase 3: 路径检测 + 启动 | 合并为新的 Phase 3（启动 + 关于页） | 路径已提前到 Phase 1 |
| Phase 7: 集成测试 + 打包 | 合并为 Phase 6 | 减少阶段数，更均衡 |

---

## 五、核心原则：规则集互斥

**两个规则集永远不可能同时启用。**

- E&D 和 Go-Astray 是互斥的，玩家一次只能选一个
- 规则集之间的实体比较（同名/覆盖/替换）**毫无意义**
- 冲突检测引擎在任何阶段都不得比较两个规则集

### 对冲突检测的实际影响

1. **globalIndex 只放扩展**。规则集实体只缓存，用于类型存在性检查。
2. **类型存在性检查**：扩展引用的 `unitType`/`requiredTech`/`requiredResource`，检查是否存在于**全部已加载模组（规则集+扩展）的集合**中。不存在的才是风险。
3. **原版 unitType 白名单**：`Sword`/`Archery`/`Mounted` 等是 Civ5 游戏引擎硬编码类型，不在任何 JSON 文件中。必须单独维护白名单。科技和资源应按规则集 JSON 解析——parser 出问题则另修。
4. **不做按规则集拆分的兼容性报告**。扩展和规则集的匹配是玩家自己决定的事。

---

## 六、技术选型实际落地 vs 方案差异

| 决策项 | 方案原定 | 实际落地 | 原因 |
|--------|---------|---------|------|
| 路由 | vue-router | **v-if 切换** | 6 个页面平级无嵌套、桌面端无 URL 栏、下载进度靠 EventsOn 推送不需路由参数 |
| 状态管理 | Pinia | **组件内 ref + prop** | 下载状态由 Go 后端持有，前端 GetDownloadList 拉取 + EventsOn 增量更新，不需要全局 store |

## 七、Phase 4 实施中遇到的技术问题

### 1. Go 版本号不一致（🟡 环境问题）

**现象**：`go.mod` 声明 `go 1.25.0`，但实际安装 Go 1.24.4。`GOTOOLCHAIN=local` 时直接拒绝编译。

**原因**：Wails v2.13.0 自动升级了 go.mod 的 Go 版本指令。

**解决**：不加 `GOTOOLCHAIN=local`，Go 自动下载 1.25 工具链后编译通过。

### 2. Windows Defender 误报 Go exe 为病毒（🔴 阻断级）

**现象**：每次 `wails build` 产出的 `.exe` 被 Windows Defender 隔离。

**原因**：Go runtime 内存分配模式触发启发式扫描。

**解决**：Windows 安全中心 → 排除项，添加 `build/bin` 目录。

### 3. 镜像测速打根路径全部超时（🔴 功能缺陷）

**现象**：测速后只显示"直连"，所有镜像站不可用。下载走直连几十 KB/s。

**原因**：`TestMirrorsLatency()` 对镜像发 `HEAD https://ghproxy.com/`（根路径），大部分镜像不响应根路径 HEAD，全部超时被丢弃。

**修复**：改用 GET 真实探测文件（Unciv README）通过镜像，即 `GET https://ghproxy.com/https://raw.githubusercontent.com/...`。只有 200 才计入。新增 ghfast.top、kkgithub.com、hub.nuaa.cf 三个镜像，共 6 个。

### 4. 非直链 URL 下载 HTML 当 ZIP 解包失败（🔴 功能缺陷）

**现象**：用户粘贴 GitHub Releases 页面 URL，下载"成功"但解包报 `zip: not a valid zip file`。

**原因**：对非直链 GET 下载，服务器返回 HTML 网页。下载器未检查 Content-Type，直接写入 `.zip`。

**修复**：
- HEAD 阶段检查 `Content-Type`，含 `text/html` 则提示"请使用直链"
- HTTP 状态码检查（非 200 报错）
- 新增 `FetchReleases` API + 版本选择 UI，自动构建 `archive/...zip` 直链

### 5. 无边框窗口 CSS 拖拽（🟡 Wails 特定）

**现象**：`Frameless: true` 后窗口无法拖动。

**修复**：标题栏 `--wails-draggable: drag`，按钮区 `no-drag`。窗口控制调 `WindowMinimise()`/`WindowToggleMaximise()`/`Quit()`。

### 6. OnFileDrop 参数缺失（🟡 TS 类型）

**现象**：`vue-tsc` 报 `Expected 2 arguments, but got 1`。

**原因**：Wails runtime `OnFileDrop(callback, useDropTarget)` 需要第二个 boolean 参数。

**修复**：加第二个参数 `true`。

### 7. 懒作者模组信息残缺（🟡 容错）

**现象**：部分模组 `ModOptions.json` 缺少 name/author 或根本不存在。

**修复**：
- name 空 → 用文件夹名
- author 空 → "未提供"
- 无 ModOptions.json → `IsIncomplete: true`，不崩溃不跳过
- 检查 README.md 存在性 → `HasReadme`

### 8. AppConfig 字段追加导致前端类型不匹配（🟡 联动问题）

**现象**：Go 端 `AppConfig` 加 `Theme`/`SidebarWidth` 字段后，`SettingsView.vue` 初始化字面量报 TS2345。

**原因**：Wails 自动更新 `models.ts` 类型定义，但 Vue 组件硬编码的初始值未同步。

**修复**：每次 Go struct 加字段，检查所有前端初始化字面量，补齐默认值。

### 9. GitHub API zipball_url 403（🔴 阻断级）

**现象**：通过 GitHub API 获取 Release 列表后，用返回的 `zipball_url`（`api.github.com/repos/.../zipball/v1.0.0`）下载返回 403。

**原因**：GitHub API 的 zipball 端点需要认证 token 或正确处理重定向到 `codeload.github.com`。国内网络环境下重定向链容易断开。

**修复**：放弃 API zipball，改为拼接 CDN 直链 `https://github.com/user/repo/archive/refs/tags/vX.Y.Z.zip`，无需认证，走镜像代理畅通。

### 10. 模组大小字段经常为空（🟡 数据不完整）

**现象**：按大小排序没有实际效果，因为 `ModOptions.json` 里 `modSize` 字段大部分模组不填。

**修复**：解析 ModOptions.json 后调用 `scanModDirectory()` 一次遍历完成实际大小计算，`modSize` 为 0 时自动回退到扫描结果。

### 11. README 背景色主题适配（🟡 UI 细节）

**现象**：亮暗主题下 README 和代码块的 `--bg-card` 变量在亮色模式下太淡（接近白色），可读性差。

**修复**：README 展示区和崩溃堆栈改用固定深色背景 `#1a1a24` + 浅灰文字 `#d4d4d4`，两主题下统一可读。

### 12. 下载任务 map 永不清理（🔴 内存泄漏）

**现象**：`dlTasks` map 中已完成/失败的任务永不删除，长时间运行后累计内存占用。

**修复**：`StartDownload` 入口加 `pruneOldTasks()`，累计超过 50 条任务时清除已完成/失败任务，释放 `speedSamples` 切片和文件引用。

### 13. 下载临时 ZIP 残留（🟡 磁盘浪费）

**现象**：`%TEMP%/unciv-mm-downloads/` 中的 ZIP 文件解包后不删除，持续占用磁盘。

**修复**：新增 `CleanupTempFile()` 方法，前端解包成功后调用删除临时 ZIP。

### 14. Google 翻译国内不可用（🟡 功能取舍 → ✅ 已解决）

**最终方案**：不依赖 Google。照搬 LYT 的微软/Yandex 免费方案（借 Android App 公钥签名），另加自定义 OpenAI 兼容 API（DeepSeek 等）。设置页可切换三种翻译提供者。

### 15. 微软翻译 429 限流（🟡 运行时）

**现象**：微软翻译免费额度 200 万字符/月，频繁使用触发 429。

**对策**：Yandex 1000 万字符/月更宽松，建议设置页切换。自定义 DeepSeek API ¥1/百万 token 不限量。

### 16. 地图扫描只认 .civ5map 后缀（🔴 功能缺陷）

**现象**：Unciv 地图文件常不带后缀（如 `欧罗巴洲`），`ScanMaps` 只匹配 `.civ5map` 导致扫不到。

**修复**：`maps/` 和 `mods/*/maps/` 下所有非目录文件都纳入扫描，排除 `backup` 目录。

## 八、后续功能增强

### 17. GitHub 作者提取（🟡 数据补全）

**现象**：大量模组的 `ModOptions.json` 不填 `author` 字段，列表显示"未提供"。

**修复**：`parseModInfo` 中若 author 为空且 `modUrl` 含 `github.com/`，自动提取 owner 作为作者回退。

### 18. Wesnoth .map 格式支持（✅ 已实现）

原本判断"工程量巨大不做"，实际只需 40 个地形码映射表，~70 行代码。拖入 .map 文件自动转换+原文件备份。

---

## 九、执行阶段最终调整

| 阶段 | 原始预估 | 实际结果 |
|------|---------|---------|
| Phase 1 项目初始化 + 路径检测 | Day 1 | ✅ |
| Phase 2 模组扫描与管理 | Day 2-3 | ✅ |
| Phase 3 启动 Unciv + 设置 | Day 4 | ✅ |
| Phase 4 下载 + 解包 + 地图导入 | Day 5-7 | ✅ |
| Phase 5 冲突检测 | Day 8-10 | ✅ |
| Phase 6 崩溃报告 + 打包 | Day 11-14 | ✅ |
| 无边框窗口 + 主题 | 未预计 | ✅ |
| 在线浏览 + 翻译 + Wesnoth | 未预计 | ✅ |
| 图标 + 字体 + 队列优化 | 未预计 | ✅ |
| **总计** | **14 天** | **已完成** |

---

## Session 2 (2026-07-16) 新增审查记录

### 21. Unciv 自带 JRE 类加载异常（🔴 外部依赖问题）

**现象**：
- 点击"定位模组错误"时 `NoClassDefFoundError: kotlin.jvm.internal.Intrinsics`
- 启动时偶发 `NoClassDefFoundError: UniqueType$UniqueParameterErrorSeverity`
- 两个类在 `Unciv.jar` 内均存在（`unzip -t` 验证通过），且 jar 完整性校验 OK
- 使用系统 Java（同样 Temurin-21.0.11）运行完全正常

**排查过程**：
1. 确认 `Unciv.jar` 中 `UniqueType$UniqueParameterErrorSeverity.class` 存在 ✅
2. 确认 `Unciv.jar` 中 `kotlin/jvm/internal/Intrinsics.class` 存在 ✅
3. 确认 `jre/bin/java --version` = Temurin-21.0.11+10，与系统 Java 版本一致
4. 确认 `jre/lib/modules` 99MB，模块数 50（完整 JDK，非 jlink 裁剪版）
5. 确认 `Unciv.json` classPath 仅为 `["Unciv.jar"]`，无外部干扰
6. 系统 Java 运行 `java -jar Unciv.jar` 无任何问题

**结论**：自带 `jre/` 的模块系统元数据损坏或存在特定环境下的类加载器 bug，**与 UMM 无关**。

**建议**：修改 `Unciv.json` 将 `jrePath` 置空以使用系统 Java，或重新下载 Unciv 获取新的 `jre/`。

### 22. 存档模组列表读取限制（🔴 功能缺陷 → ✅ 已修复）

**现象**：存档详情页显示"无扩展模组"，但 Unciv 游戏内可见模组列表。

**原因**：`gameParameters.mods` 位于存档文件尾部（434KB 文件的 433939 字节处），而代码只读取前 96KB（98304 字节），截断导致 gjson 找不到该字段。

**修复**：读取限制从 96KB 提升至 2MB（2097152），覆盖所有已知存档大小。

### 23. 存档模组字段搜索路径（🟡 容错增强）

**现象**：部分存档的 mods 不在 `gameParameters.mods` 而在 `lastGameSetup.gameParameters.mods`。

**修复**：三层兜底搜索：`mods` → `gameParameters.mods` → `lastGameSetup.gameParameters.mods`。

### 24. 侧边栏游戏版本显示（✅ 新增功能）

**新增**：`GetUncivVersion()` 方法从 `Unciv.jar` 的 `META-INF/MANIFEST.MF` 读取 `Specification-Version` 字段，降级从 `GameSettings.json` 的 `createdWithVersion` 字段读取。侧边栏底部显示 `🕹️ Unciv x.x.x`。

### 25. LYT 配色完整适配（✅ UI 改进 → ⚠️ 后续简化）

**改动（初始）**：
- `:root` 定义基础文字色：`--text-primary: #1e293b`, `--text-secondary: #475569`, `--text-muted: #94a3b8`
- `[data-theme="light"]` 仅设置背景白，文字继承 :root
- `[data-theme="dark"]` 仅覆盖背景和边框，文字不变
- 兼容别名层将 LYT 变量名映射到 UMM 现有组件变量（`--bg-primary` → `--bg-app` 等）
- `--accent` 在别名中覆盖为 `#4f46e5`（LYT 的 indigo primary），因为 UMM 用 `--accent` 作为按钮/主色调

> **后续简化（Session 4 前）**：别名层 + LYT 集共 43 个变量经 grep 验证，只有 18 个被组件引用。删除所有未引用变量和整个别名层，保留单层 18 变量系统。`--accent` 直接定义为 `#4f46e5`，不再通过别名覆盖。见 [App.vue](frontend/src/App.vue#L96-L138)。

### 26. 关于页面重做（✅ UI 改进）

**改动**：仿 LYT VPK 风格：
- Hero 区（logo + kicker + 标题 + 描述）
- 双列网格（项目信息 + 操作按钮）
- 底部功能特性卡片
- GitHub 链接统一更新为 `Permanent995/unciv-mod-manager`

### 27. 检查更新 404 处理（✅ 功能修复）

**现象**：GitHub 仓库无 Release 时，`/releases/latest` 返回 404，UMM 显示"检查失败"。

**修复**：`selfupdate.go` 中在 `fetchJSON` 返回 404 时，返回"当前已是最新版本"而非错误。

### 28. 测试文件补充（✅ 质量提升）

新增 `app_test.go`，17 个测试覆盖：
- 语义化版本解析与比较
- 原生类型判断
- 文件分类
- 冲突检测（4 个子测试：TRY_INJECT 值差异/相同/CREATE_OR_REPLACE/无 mergeAction）
- 字符串截断
- 废弃规则检测
- 镜像列表完整性
- 速度格式化
- 任务 ID 生成
- owner/repo URL 解析
- JSON 预处理
- 存档元数据解析（空文件容错）
- Wesnoth 地形映射
- URL 编码
- 镜像 URL 转换

### 29. 模组诊断过早显示"通过"（✅ 功能修复）

**现象**：进入工具箱→模组诊断时，`diagIssues` 初始为空数组，两个 `v-if` 同时为 true，导致在诊断运行前就显示"✅ 所有模组通过自检"。

**修复**：增加 `diagDone` ref，仅在诊断完成后显示通过消息。

### 30. 下载队列安装按钮时机（⚠️ 临时方案 → ✅ Session 9 根治）

**现象**：点击下载后立即显示"安装更新"按钮，但下载尚未完成。

**Session 2 临时修复**：移除自动显示安装按钮，改为显示手动安装指引。

**Session 9 根治**：分离 `downloadQueued` / `downloadCompleted` 状态，监听 `download:complete` 事件，下载真正完成时才显示安装按钮（见 #62）。

---

## Session 3 (2026-07-20) — Phase 2: 镜像源健康检测 + 自动回退 + BrowseView 修复

### 32. 镜像系统中心化（✅ 新增模块 mirror.go）

**改动**：
- 新建 `internal/app/mirror.go`，集中管理镜像列表和 URL 构造
- `defaultMirrors()` 返回 6 个内置镜像：ghproxy.com、mirror.ghproxy.com、gh.api.99988866.xyz、ghfast.top、kkgithub.com、hub.nuaa.cf
- `getAllMirrors()` 合并内置镜像 + 用户自定义镜像（去重）
- `GetMirrorHealth()` 并发 HEAD 探测所有镜像，返回延迟/存活/最后检测时间
- `TestSingleMirror(url)` 供前端测试单个自定义镜像

### 33. 镜像故障切换（✅ 修复 downloader.go）

**改动**：
- `dlTask` 新增 `mirrorCandidates []string` 字段，存储剩余候选镜像
- `StartDownloadWithMirror()` 预计算候选列表（auto/手动指定/直连三种模式）
- `tryNextMirror()` 方法切换到列表下一个镜像
- `runDownload()` 中四种失败场景均触发回退：

| 失败场景 | 修复前 | 修复后 |
|---------|--------|--------|
| HEAD 连接失败 | `failTask` 直接返回 | `tryNextMirror()` → `goto retryHead` |
| HEAD 返回非200 | `failTask` 直接返回 | 同上 |
| 返回 HTML | `failTask` 直接返回 | 同上 |
| 下载内容失败 | 已有回退（保留） | 已有回退（保留） |

### 34. GitHub API 镜像遍历（✅ 替换三处硬编码）

- `FetchReleases()`：直连 → `getAllMirrors()` 遍历 → 报错
- `searchOnlineModsAPI()`：直连 → `getAllMirrors()` 遍历 → 报错
- `FetchReadme()`：动态从 `getAllMirrors()` 构建 URL 列表，并发取最快成功者
- `CheckSelfUpdate()`：直连 → `getAllMirrors()` 遍历 → 报错
- 删除 `github_api.go` 中的 `applyMirror()`（已移至 mirror.go）

### 35. 镜像设置 UI（✅ 新增 SettingsView 区块）

- 自动/手动模式切换
- 手动模式下拉选择可用镜像（不可用镜像置灰）
- 健康列表显示所有镜像的 label/延迟/状态/最后检测时间
- 自定义镜像：输入 URL → 测试 → 添加
- 删除自定义镜像
- 重新测试所有镜像按钮

### 36. DownloadsView 镜像模式适配（✅）

- `onMounted` 读取 `config.mirrorMode`
- auto 模式显示"自动（故障切换）"，禁用下拉
- manual 模式保持现有选择行为

### 37. BrowseView 新增下载和版本功能（✅ 修复）

**问题**：模组在线浏览点开后不能下载、看不到仓库地址和版本。

**修复**：
- 点击模组后显示可打开的 GitHub 链接 `🌐 https://github.com/owner/repo`
- 点击「查看版本」调用 `FetchReleases`，展示所有 Release（tag、名称、日期）
- 每个 Release 旁有「下载」按钮，走镜像线路，添加到下载队列
- 关闭详情时自动清空版本列表

### 38. 测试扩充（✅ 17 → 44 个测试）

**新增文件** `internal/app/entity_test.go`（27 个测试）：
- `TestCategorizeMod`（9 场景）：全部 7 个 topic 分支 + isBaseRuleset + 空兜底
- `TestWalkEntity`（6 场景）：flat 实体、tech 类型注册、column-based techs 展开、空名跳过、非对象跳过、replaces/upgradesTo 字段
- `TestParseEntities`（3 场景）：数组解析、空数组、非数组 JSON
- `TestMirrorURL`（5 组合）、`TestBuildMirrorDownloadURL`、`TestExtractHost`（5 URL）、`TestPreprocessUncivJSON_EdgeCases`（7 边界）

**扩展** `internal/app/app_test.go`（6 个冲突检测场景）：
- Zero Strength 不标记为差异
- Cost-only 差异
- 两边零值安全
- 空 vs TRY_INJECT → override
- 两边空相同 → override
- 两边空不同 → override（带详情）

**其他新增**：
- `TestDefaultMirrors`（原 `TestGetMirrors`，因 `getMirrors()` 被删除而改名）、`TestApplyMirror`（5 组合 + null mode）

### 39. 审查发现的 Bug：runDownload HEAD 失败未触发镜像回退（🔴 已修复）

**现象**：`runDownload()` 中 HEAD 请求的三个失败路径直接 `failTask` 返回，未调用 `tryNextMirror()`。

**影响**：如果当前镜像服务器无响应（HEAD 超时/非200/返回 HTML），下载任务直接失败，不会尝试下一个镜像。

**修复**：三个 HEAD 失败路径均先调 `tryNextMirror()`，有候选则重置上下文 `goto retryHead`。

**发现方式**：代码审查时逐路径检查 `runDownload()` 逻辑流。

---

## Session 4 (2026-07-20) — 代码去重与清理

### 40. 镜像系统去重（✅ 代码清理）

**问题**：代码审查发现方案中存在多处重复实现：

| 重复项 | 涉及文件 | 问题 |
|--------|---------|------|
| `TestMirrorsLatency` vs `GetMirrorHealth` | downloader.go / mirror.go | 两套独立探测逻辑，返回值格式不同但功能相同 |
| `BuildDownloadURL("mirror"模式)` vs `applyMirror` | downloader.go / mirror.go | URL 剥离 + 拼接逻辑完全一致 |
| `probeSuffix` 引用 `probeURL` | mirror.go → downloader.go | 跨文件常量依赖，隐蔽的编译安全风险 |
| `safeDomains` 硬编码镜像域名 | downloader.go | 新增自定义镜像后仍触发 URL 警告 |
| `getMirrors()` 转发 | downloader.go | 直接调 `defaultMirrors()`，多余间接层 |

**修复**：

**① TestMirrorsLatency → GetMirrorHealth 委托**

`TestMirrorsLatency()` 改为调用 `GetMirrorHealth()` 并转换返回格式。探测逻辑只保留一份在 mirror.go。

**② BuildDownloadURL → applyMirror 委托**

`BuildDownloadURL` 的 `"mirror"` 分支改为调 `applyMirror(rawURL, "mirror", cfg.MirrorURL)`，URL 剥离逻辑只保留一份。

**③ probeURL 移入 mirror.go**

将 `probeURL` 常量从 downloader.go 移到 mirror.go，`probeSuffix` 引用包内常量，消除跨文件依赖。

**④ safeDomains 动态化**

`ValidateDownloadURL()` 的允许域名列表中，镜像部分改为从 `getAllMirrors()` 运行时构建。GitHub 基础设施域名保持硬编码。

**⑤ getMirrors() 删除**

直接删除 `getMirrors()`，测试改为 `TestDefaultMirrors` 调 `defaultMirrors()`。

### 41. GetMirrorHealth 缓存（✅ 性能优化）

**问题**：SettingsView 和 DownloadsView 挂载时各自调 `GetMirrorHealth()`，短时间内重复发起 6+ 次 HTTP 请求。

**修复**：`mirror.go` 增加包级缓存变量，30 秒 TTL。命中缓存时直接返回副本，不发起网络请求。

### 42. 前端下载页改用 GetMirrorHealth（✅ 前端简化）

**问题**：DownloadsView.vue 仍调用被废弃的 `TestMirrorsLatency()`，返回 `map[string]int64` 后手动解析 URL hostname。

**修复**：改用 `GetMirrorHealth()` 直接取 MirrorInfo 数组，移除了 `new URL(url).hostname` 解析步骤。

### 43. CSS 变量体系简化（✅ 代码清理）

**问题**：历史累积了两套 CSS 变量定义——43 个 LYT 风格变量（`--primary-*`、`--gray-*`、`--shadow-*` 等）加 10 行 UMM 兼容别名覆盖，共 53 行。大多数变量未被任何组件引用。

**验证方式**：
```bash
grep -roh 'var(--[a-z0-9_-]*' frontend/src/ | sort | uniq -c | sort -rn
```

**修复**：
- 删除全部 43 个 LYT 未引用变量
- 删除全部别名覆盖层
- 保留实际被引用的 18 个变量，统一放在 `:root, [data-theme="light"]` 和 `[data-theme="dark"]` 两组中
- `--accent` 直接定义为 `#4f46e5`，不再通过别名覆盖
- 验证：`vue-tsc --noEmit` 通过，视觉无变化

---

## Session 5 (2026-07-20) — 消灭轮子与死代码

### 44. 清理重复造轮子（✅ 代码清理）

| 函数 | 问题 | 修复 |
|------|------|------|
| `urlEncode()` [github_api.go:391] | 手动编码只处理空格和冒号，漏了 `+` `&` `%` 等 | 改用 `url.QueryEscape()` |
| `base64Encode()` [scanner.go:278] | 一行封装 `base64.StdEncoding.EncodeToString()` | 调用处直接内联标准库 |
| `buildMirrorDownloadURL()` [mirror.go:205] | 单行委托 `mirrorURL()`，三个调用处多一层间接 | 删除，调用处直调 `mirrorURL()` |

### 45. 删除死代码（✅ 代码清理）

三个函数已被 `scanModDirectory()` 取代，但原函数未删除：

- `hasOnlyImageFiles()` — scanner.go
- `hasMusicFiles()` — scanner.go
- `dirSize()` — scanner.go

零调用者，已全部删除。

### 46. probeSuffix 包级变量内联（✅ 简化）

`probeSuffix` 是 `strings.TrimPrefix(probeURL, "https://")` 的缓存，只在 `GetMirrorHealth()` 和 `TestSingleMirror()` 中各用一次。内联为表达式后删除包级变量，减少全局状态。

---

## Session 6 (2026-07-20) — 审查报告问题确认与文档清理

### 47. 审查报告 15 项状态确认（✅ 整理）

对照审查报告 `umm-review-report.md` 逐项核查当前代码：

| 组 | 已修 | 无需处理 | 待修 |
|---|------|---------|------|
| 严重问题 1-3 | backup 残留、竞态条件（2） | Unciv.json 外部问题（1） | — |
| 一般问题 4-11 | dlTasks 并发、dirSize、重复遍历、错误边界、CSS 冗余（6） | GetUncivVersion 开 jar（1） | speedSamples 上限（1） |
| 小问题 12-15 | readModVersion 路径（1） | extractor 覆盖设计如此（1） | ToolboxView 拖拽监听（1） |
| **合计** | **8** | **3** | **2** |

部分修：测试覆盖不足（17→44 测试）、UMMVersion 硬编码值已更新。

### 48. 文档清理（✅ 整理）

- 删除无用文档 3 个：`umm-vs-unciv-comparison.md`、`unciv-crash-report.md`、`unciv-mod-compatibility-analysis.md`
- 恢复 LYT VPK 架构分析表到设计审查记录头部
- `umm-review-report.md` 所有问题标注状态 + 更新整体评分 7.8→8.5

---

## Session 7 (2026-07-20) — 自定义下载 + 设置卡片化 + 存档 Bug 修复

### 49. 工具箱新增「自定义下载」标签页（✅ 新增功能）

**需求**：在工具箱加 PCL 风格的自定义文件下载工具，支持输入任意 URL 下载文件。

**改动**：
- `internal/app/downloader.go` 新增两个 Go 方法：
  - `SelectDownloadDirectory()` — 调 Wails `OpenDirectoryDialog` 让用户选择保存目录
  - `SaveDownloadedFile(srcPath, destDir)` — 将临时文件复制到用户指定目录
- `frontend/ToolboxView.vue` 新增第 4 个 tab「📥 自定义下载」：
  - **保存路径条**（PCL 风格）：`📁 D:\Downloads [...]`
  - URL 输入框 + 文件名自动提取 + 下载按钮
  - 下载任务列表：下载中（进度条）/ 暂停 / 已完成 / 失败
  - 自动保存到用户设定目录
- 复用现有 `StartDownloadWithMirror`、`GetDownloadList`、EventsOn 进度推送
- 参考 PCL 开源源码（tangge233/PCL2、PCL-Community/PCL2-CE），确认百宝箱功能在开源版中为空桩

### 50. 设置页面卡片式 UI 重构（✅ UI 改进）

**问题**：设置页原为扁平列表式布局，各组设置之间视觉区分不足。

**改动**：每项设置改为独立卡片：
- 左侧图标 + 右侧内容区
- 悬停阴影效果
- 统一的 `.card` / `.card-title` / `.card-desc` 组件样式

### 51. 存档列表 nil 切片崩溃修复（🔴 已修复）

**问题**：`ListSaveArchives()` 在备份目录存在但为空时返回 `nil` 切片，JSON 序列化为 `null`，前端 `all.filter(...)` 抛 `TypeError: Cannot read properties of null`

**修复**：`saves.go` 返回值前加 `if archives == nil` 检查，确保始终返回空数组而非 nil。

---

## Session 8 (2026-07-21) — UI 标题颜色统一 + 窗口拖拽修复尝试

### 52. README 正文从代码块风格改为文档风格（✅ UI 改进）

**问题**：`.readme-text` 使用 `var(--code-bg)`（深色底）+ `var(--code-text)`（浅灰字），在亮色模式下是黑底灰字，与页面其他文字风格不统一。

**改动**：
- `BrowseView.vue`: `.readme-text` 改为 `background: var(--bg-secondary); color: var(--text-primary); border: var(--border-color)`
- 下同 ModsView 的 `.readme-text`

### 53. 小标题统一从 `--text-secondary` 提升为 `--text-primary`（✅ UI 改进）

**问题**：页面中各区域小标题（"📦 版本""📖 README""📝 中文翻译""📦 备份管理""📦 备份存档"等）使用 `--text-secondary`，偏淡，不够突出。

**改动**（涉及 5 个文件）：

| 文件 | 选择器 | 旧值 | 新值 |
|------|--------|------|------|
| BrowseView.vue | `.detail-section-hdr h3` | `--text-secondary` | `--text-primary` |
| BrowseView.vue | `.detail-readme h3` | `--text-secondary` | `--text-primary` |
| BrowseView.vue | `.bcat-item` | `--text-secondary` | `--text-primary` |
| ModsView.vue | `.readme-header h3` | 无（继承body） | `var(--text-primary)`（显式） |
| ModsView.vue | `.translated-section h3` | `--text-secondary` | `--text-primary` |
| ModsView.vue | `.backup-section summary` | `--text-secondary` | `--text-primary` |
| SavesView.vue | `.archive-section summary` | `--text-secondary` | `--text-primary` |

### 54. `.view-header h1` 全部显式声明颜色（✅ 代码规范化）

**问题**：8 个视图的 `h1` 标题（如"模组库"）没有显式设置 `color`，依赖从 `body` 继承 `--text-primary`，属于隐式硬编码。

**改动**：在 8 个文件的 `.view-header h1` 规则中统一添加 `color: var(--text-primary)`：
- BrowseView.vue、DownloadsView.vue、HelpView.vue、MapsView.vue、MultiplayerView.vue、SavesView.vue、SettingsView.vue（已有）、ToolboxView.vue

ModsView.vue 已在 #53 中一并处理。

### 55. 无边框窗口拖拽修复尝试（🟡 待验证）

**问题**：`Frameless: true` 后窗口无法拖动。标题栏 CSS 设了 `--wails-draggable: drag` 但无效。

**排查过程**：

| 版本 | 方案 | 结果 |
|------|------|------|
| 原始 | `body { --wails-draggable: no-drag }` + `.title-bar { --wails-draggable: drag }` | ❌ 拖不动 |
| 第1次 | 去掉 body 的 no-drag，只保留 title-bar 的 drag | ❌ 拖不动 |
| 第2次 | 改为 `.app-root { drag }` + `.app-body { no-drag }` | ❌ 拖不动 |
| 第3次 | 改用 `data-wails-drag` HTML 属性（Wails 社区方案） | ❌ 更糟，连边框拖动都没了 |
| 第4次 | 去掉 `data-wails-drag`，仅留 `.title-bar { --wails-draggable: drag }` | ❌ 待测试 |

**参考**：Wails 官方示例使用 `data-wails-drag` 属性而非 CSS 变量。当前代码使用 `--wails-draggable: drag` CSS 变量（Wails v2.13.0），两种方案均未验证通过。

**当前状态**：仅 `.title-bar` 保留 `--wails-draggable: drag`，其余地方无拖拽相关代码。

### 56. 用户自定义亮色主题色（✅ 用户配置）

用户在 App.vue 中调整了亮色模式色值：
- `--bg-sidebar`: `#b9d6f3` → `#c4d2e1`
- `--bg-card`: `#c2dbf4`
- `--text-muted`: `#7a8ba8` → `#a4b1c7`
- `--code-text`: `#d4d4d4` → `#eedfdf`

暗色模式同时调整为浅灰色系（`--bg-primary: #c9cdd7` 等），营造类似"褪色"效果。

---

## Session 9 (2026-07-22) — 自更新系统重构：下载→安装→校验全链路

### 审查背景

自更新模块（`selfupdate.go`）存在多项设计缺陷，导致功能从未真正端到端跑通。本 Session 对照 lytvpk 的实现进行了系统性修复。

### 57. 版本号未随 tag 更新（✅ 已修复）

**现象**：Git 最新 tag 为 `v1.9`，最新 commit 日志为 `v1.9: ...`，但 `app.go` 中 `UMMVersion = "1.8.0"`。About 页显示 v1.8.0，检查更新永远提示有新版本。

**根因**：发版时 tag 打了、exe 传了，但源码常量忘记更新。

**修复**：`UMMVersion` → `"1.9.0"`。

### 58. 前端安装按钮未接上（✅ 已修复）

**现象**：Go 端 `InstallSelfUpdate()` 已实现自动解压替换逻辑，但 `AboutView.vue` 下载完成后只显示"到 build/bin/ 手动解压覆盖"，从未调用安装函数。整个自更新流程在最后一步断掉。

**修复**：
- 下载完成后显示「⬆ 安装更新」按钮，点击调用 `InstallSelfUpdate()`
- 安装成功后显示「🔁 退出并重启 UMM」按钮，调 `Quit()` 退出
- 旧版备份为 `.bak`，异常时可手动回滚

### 59. Release 资产匹配失败（✅ 已修复）

**现象**：GitHub Release 上传的是裸 `unciv-mod-manager.exe`，但 `CheckSelfUpdate` 只匹配 `*windows*.zip`。资产匹配为空 → 回退到 `zipball_url`（源码 zip）→ `InstallSelfUpdate` 在源码中找不到 `.exe` → 永远安装失败。

**修复**（`CheckSelfUpdate`）：
- 资产匹配优先级：`.zip` → `.exe` → `zipball_url`
- 移除无意义的 "windows" 子串匹配

### 60. 安装逻辑只处理 zip（✅ 已修复）

**现象**：`InstallSelfUpdate` 硬编码 `zip.OpenReader()`，裸 exe 下载成功但安装时报 `zip: not a valid zip file`。

**修复**（`InstallSelfUpdate`）：
- zip 分支：解压 → 找 exe → 提取到 `.new` → 替换
- exe 分支：直接复制到安装目录 `.new` → 替换
- `DownloadSelfUpdate` 文件名根据 URL 后缀动态选择 `.zip` 或 `.exe`

### 61. 检查更新镜像回退完全无效（✅ 已修复）

**现象**：API 直连失败后，代码尝试 9 个镜像走 `mirrorURL(apiURL, m)` 代理 GitHub API。但镜像只代理文件下载，不代理 `api.github.com`，9 连败后报错。

**修复**：参照 lytvpk 的 **302 redirect 方式**：
1. 请求 `<镜像>/https://github.com/<repo>/releases/latest`
2. GitHub 302 重定向到 `/releases/tag/vX.Y.Z`
3. 从最终 URL 解析 tag → 构造下载地址
4. 9 个镜像逐个尝试，任意一个成功即可

新增 `fetchLatestTagViaMirror()` 函数，仅在 API 直连失败时触发。API 直连作为优先路径（可获得 Release Notes 和精确资产 URL），镜像 redirect 作为保底。

### 62. 下载-安装两段割裂（✅ 已修复）

**现象**：点击下载后立即显示安装按钮（`downloadOk = true`），但此时文件还在下载队列中。用户点了安装会报"未找到更新文件"。

**修复**：
- `downloadQueued` 和 `downloadCompleted` 分离为两个状态
- `onMounted` 注册 `EventsOn('download:complete')` 监听器
- 仅当后端推送 `download:complete` 且 `filename` 匹配 `umm_update.zip` / `.exe` 时，才显示安装按钮
- `onUnmounted` 时 `EventsOff` 清理监听器

### 63. CI 自动构建 + 上传（✅ 新增）

**问题**：发版需手动编译 exe + 手动上传到 GitHub Release。

**新增** `.github/workflows/release.yml`：
- 触发条件：`push tags: v*`
- 环境：`windows-latest` + Go 1.22 + Node 18
- 步骤：`npm ci` → `wails build` → 上传 `unciv-mod-manager.exe` 到 Release
- 此后发版只需 `git tag v1.10 && git push --tags`

### 64. SHA256 文件完整性校验（❌ 已撤销）

**初版**：新增 `checksums.txt` + `pendingChecksumURL` + `fetchChecksumForFile()` / `fileSHA256()` / `fetchBody()` 三个辅助函数。

**撤销原因**：lytvpk 未做校验，一个 mod 管理器搞 SHA256 是过度设计。Single exe, no fuss.

### 65. 仍待解决的问题（Session 10 收尾）

| # | 问题 | 优先级 | 状态 | 说明 |
|---|------|--------|------|------|
| 65a | 检查更新无离线缓存 | 🟡 | ✅ | 网络断开时读取 `selfupdate_cache.json`，显示上次检查时间 |
| 65b | 镜像回退下载地址是猜的 | 🟡 | 📝 | 已加注释标注命名约定，CI 上传文件名必须为 `unciv-mod-manager.exe` |
| 65c | speedSamples 无上限 | 🟡 | ✅ | 加 200 条硬上限，超量时丢弃旧样本 |
| 65d | ToolboxView 拖拽监听 | 🟡 | ✅ | `startCatDrag` 逻辑正确，mousedown/move/up 完整，80-400px 约束 |
| 65e | 无边框窗口拖拽 | 🟡 | ✅ | `.title-actions` 加了 `--wails-draggable: no-drag`，按钮区不拦截拖拽 |

---

## 十、参考文档

- LYT VPK 源码：`c:\Users\drj13\Desktop\ak\lytvpk-2.5.4\`
- Unciv 模组目录：`c:\Users\drj13\Desktop\官方unciv文件\mods\`
- GameSettings.json：`c:\Users\drj13\Desktop\官方unciv文件\GameSettings.json`
- ModListCache.json：`c:\Users\drj13\Desktop\官方unciv文件\ModListCache.json`
- MergeAction 教程：`C:\Users\drj13\unciv模组制作\UncivCN扩展JSON-MergeAction教程.md`
- Wails v2 官方文档：https://wails.io/docs/
