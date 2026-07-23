# UMM Android 移植可行性评估

## 1. 现状

### 1.1 桌面版 UMM 架构

| 层 | 技术 | 文件数 |
|----|------|--------|
| 运行时 | Wails v2.13 + Go 1.25 | ~25 .go 文件 |
| 前端 | Vue 3 + TypeScript + Vite | ~6 .vue 组件 |
| 通信 | Wails IPC（Go ↔ JS） | 自动生成绑定 |
| 打包 | NSIS（Windows .exe） | 单文件 ~14MB |

### 1.2 Unciv Android 架构（来自 APK 分析）

```
Unciv-signed.apk
├── classes.dex (×4)      — Kotlin 编译产物
├── lib/arm64-v8a/libgdx.so — libGDX 原生库
├── assets/                 — 游戏资源（图片、atlas、字体）
├── AndroidManifest.xml     — package: com.unciv.app
└── META-INF/               — 签名信息
```

- 游戏引擎：libGDX（跨平台）
- 模组存储：`/Android/data/com.unciv.app/files/mods/`
- 存档存储：`/Android/data/com.unciv.app/files/SaveFiles/`

---

## 2. Android 核心限制

### 2.1 分区存储（Scoped Storage）

Android 10+ 引入分区存储，Android 11+ 严格执行：

```
/Android/data/<package>/
├── files/      ← 仅本 app 可读写
├── cache/
└── ...

其他 app 无法访问此目录
```

**结论**：独立 UMM app **无法**访问 Unciv 的 `files/mods/`。

### 2.2 权限方案评估

| 方案 | 可行性 | 原因 |
|------|:------:|------|
| SAF 文件选择器 | ❌ | Android 11+ 文件选择器不显示 `/Android/data/` |
| `MANAGE_EXTERNAL_STORAGE` | ❌ | 全盘权限拿不到其他 app 的私有目录 |
| Root / ZYGISK | ⚠️ | 技术上可行，用户基数极小 |
| `sharedUserId` + 同签名 | ⚠️ | 需 Unciv 官方配合修改 manifest |
| **Fork Unciv 内置模块** | ✅ | 同进程同目录，零权限问题 |

---

## 3. 推荐方案：Fork Unciv 内置 UMM 模块

### 3.1 架构

```
Unciv Android (forked)
│
├── 原版 Unciv（libGDX / Kotlin）
│   ├── 游戏主逻辑
│   ├── 模组下载（GitHub API → mods/）
│   └── 存档管理（SaveFiles/）
│
└── UMM 模块
    ├── Go 核心逻辑（编译为 .so 通过 JNI 调用）
    │   ├── 冲突检测引擎
    │   ├── 模组诊断（RulesetValidator 兼容）
    │   ├── 文件分类解析
    │   └── 版本比较
    │
    └── Kotlin/Compose UI（集成在 Unciv 界面中）
        ├── 模组浏览（复用 ModListCache）
        ├── 冲突分析报告
        ├── 多版本存档管理
        ├── 下载队列 + 镜像加速
        └── 诊断 + 兼容性检查
```

### 3.2 Go → Android 编译

使用 `gomobile` 将 Go 代码编译为 Android 原生库：

```bash
gomobile bind -target=android -o umm.aar ./internal/app
```

| 模块 | 复用情况 |
|------|----------|
| `conflict.go` — 冲突检测 | ✅ 直接复用（纯逻辑，无 Wails 依赖） |
| `diagnose.go` — 模组诊断 | ✅ 直接复用 |
| `scanner.go` — 模组扫描 | ✅ 复用（需适配 Android 路径） |
| `entity*.go` — 实体解析 | ✅ 直接复用 |
| `deprecated.go` — 废弃检测 | ✅ 直接复用 |
| `vanilla_types.go` — 原版类型 | ✅ 直接复用 |
| `updater.go` — 模组更新检查 | ✅ 复用（需 ModListCache） |
| `github_api.go` — GitHub API | ⚠️ 部分复用（Android 用 Kotlin/OkHttp 替代更合适） |
| `downloader.go` — 下载队列 | ⚠️ 需要 Android 原生实现 |
| `mirror.go` — 镜像加速 | ⚠️ URL 拼接逻辑可复用 |
| `selfupdate.go` — 自更新 | ❌ 不需要（通过 Google Play 更新） |
| `game_link.go` — 桌面路径 | ❌ 不需要 |
| `map_importer.go` — 地图导入 | ✅ 可复用 |
| App.vue / 各 View | ❌ 用 Kotlin Compose 重写 |

**核心逻辑复用率**：~70% 的 Go 代码无需修改即可复用。

### 3.3 UI 方案

| 方案 | 优势 | 劣势 |
|------|------|------|
| **Kotlin Compose** | Unciv 同技术栈，集成最自然 | 需重写全部 UI |
| WebView + Vue | 前端 100% 复用 | 启动慢、内存高、体验不如原生 |
| Flutter | 统一 UI 层 | 引入新框架，包体积大增 |

**推荐 Compose**：Unciv 本身用 Kotlin，UI 融入设置页或独立 Tab 最不违和。

---

## 4. 功能映射

| 桌面 UMM 功能 | Android 可行性 |
|---------------|:---:|
| 模组库（本地扫描） | ✅ 同目录 |
| 在线浏览（ModListCache） | ✅ 已有缓存 |
| 冲突检测（覆盖分析） | ✅ 核心引擎复用 |
| 模组诊断 | ✅ 核心引擎复用 |
| 一键迁移（跨版本） | ✅ /Android/data 内自由操作 |
| 下载队列 + 镜像加速 | ✅ 需 Android 原生下载器 |
| 自定义下载 | ✅ |
| 存档管理 | ✅ 同目录 |
| 多版本 Unciv 检测 | ❌ Android 单版本 |
| 自更新 | ❌ Google Play 接管 |
| 翻译 README | ✅ API 层无变化 |
| 崩溃报告查看 | ⚠️ 依赖 Unciv 的日志 |
| 联机快照比对 | ⚠️ 需分析 Unciv MP 协议 |

---

## 5. 工作量估算

| 阶段 | 内容 | 预估 |
|------|------|------|
| Phase 1 | Unciv fork + gomobile 集成 + 冲突检测跑通 | 3-5 天 |
| Phase 2 | Compose UI（模组库 + 冲突报告 + 诊断） | 5-7 天 |
| Phase 3 | 下载队列 + 镜像加速 + 存档管理 | 3-4 天 |
| Phase 4 | 在线浏览 + 更新检查 + 迁移 | 2-3 天 |
| Phase 5 | 测试 + 发布 | 2-3 天 |
| **总计** | | **~15-22 天** |

---

## 6. 风险和注意事项

1. **Unciv 许可证**：Unciv 使用 GPL-3.0，Fork 必须开源
2. **编译环境**：gomobile 需要 Android NDK，Go 1.25 兼容性需验证
3. **包体积**：Go .so 文件 ~5-8MB（arm64），APK 增大 ~30%
4. **性能**：JNI 调用开销可忽略（冲突检测是 CPU 密集型，Go 比 Kotlin 快）
5. **Unciv 版本迁移**：Google Play 版不支持模块注入，需要自行签名发布

---

## 7. 结论

**可行。** 推荐 Fork Unciv + gomobile + Compose 方案。

核心优势：70% 的 Go 逻辑直接复用，不需要从零开发冲突检测引擎。最大工作量在 Compose UI 层和下载模块的 Android 化。
