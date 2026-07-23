# UMM 跨平台方案

## 1. 平台概览

| 平台 | 方案 | 技术栈 | 状态 |
|------|------|--------|:---:|
| Windows | Wails v2 + Go + Vue 3 | 单 exe，~14MB | ✅ 已完成 v1.9.2 |
| macOS | Wails v2 + Go + Vue 3 | 同一代码库，交叉编译 | ⚠️ 未测试 |
| Linux | Wails v2 + Go + Vue 3 | 同一代码库，交叉编译 | ⚠️ 未测试 |
| Android | Fork Unciv + gomobile + Kotlin Compose | Go 编译为 .so，嵌入 Unciv | ❌ 规划中 |

## 2. 桌面端（Windows / macOS / Linux）

### 2.1 当前架构

```
UMM.exe (Wails runtime)
├── Go 后端（Wails 绑定 ~25 文件）
│   ├── 模组扫描 / 冲突检测 / 诊断
│   ├── 下载队列 + 镜像加速
│   ├── GitHub API / 自更新
│   └── 存档 / 地图 / 备份管理
├── Vue 3 前端（~6 组件）
│   ├── 模组库 / 在线浏览 / 下载
│   ├── 工具箱（冲突、诊断、崩溃）
│   ├── 设置（路径、迁移、主题）
│   └── 存档 / 联机 / 地图
└── Wails IPC（Go ↔ JS 自动绑定）
```

### 2.2 跨平台编译

Wails v2 原生支持 Windows / macOS / Linux，同一代码库：

```bash
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

前端零改动。Go 端 `game_link.go` 仅 Windows（Win32 API 检测进程），需平台适配。

### 2.3 桌面端迁移

桌面无沙箱限制，UMM 直接读写 `UncivPath/mods/`、`SaveFiles/`、`maps/`。跨版本迁移已实现：设置页 `MigrateUncivData(from, to)` 增量复制。

## 3. Android 端

### 3.1 核心限制

Android 11+ `/Android/data/<package>/` 是 UID 级别隔离，独立 app 无法访问其他 app 的私有目录。唯一可行方案：**Fork Unciv，内置 UMM 模块**。

### 3.2 方案架构

```
Unciv Android Fork (com.drj13.uncivmm)
│
├── Unciv 游戏引擎
│   ├── libGDX（Kotlin/Java）
│   ├── mods/      ← 同沙箱，UMM 直接读写
│   ├── SaveFiles/
│   └── ModListCache.json
│
└── UMM 模块
    ├── Go 核心（gomobile → .so）
    │   ├── conflict.go   ✅ 复用
    │   ├── diagnose.go   ✅ 复用
    │   ├── scanner.go    ✅ 复用
    │   ├── entity*.go    ✅ 复用
    │   └── deprecated.go ✅ 复用
    │
    ├── Kotlin/Compose UI
    │   ├── 模组库 + 冲突报告
    │   ├── 诊断 + 版本检查
    │   └── 下载队列 + 镜像
    │
    └── JNI 桥接
        └── Go .so ←→ Kotlin 调
```

### 3.3 Go → Android 编译

```bash
# 创建 gomobile-ready 包（裁剪 Wails 依赖）
mkdir ummcore
cp conflict.go diagnose.go scanner.go entity*.go deprecated.go vanilla_types.go ummcore/

# 编译为 Android .aar
gomobile bind -target=android -o umm.aar ./ummcore
```

Kotlin 端调用：

```kotlin
val core = Ummcore.new()
val reports = core.analyzeConflicts(modsPath, rulesetsPath)
```

### 3.4 数据迁移

| 场景 | 策略 |
|------|------|
| 官方 Unciv → Fork v1.0 | **唯一断档**。旧 Unciv 需自行导出 mods/ 到共享目录，Fork 导入。或发旧版 Unciv "数据导出"插件 |
| Fork v1.0 → v2.0 | 自动保留。同包名升级，Android 不删 `/data/data/` |
| Google Play → Sideload | 同包名：保留。换包名：丢失 |

### 3.5 签名问题

Fork 和官方 Unciv 包名不同 ⇒ 不同 app ⇒ 不同沙箱。签名只需要 Fork 自己一致（v1 签的 key 用于签 v2），升级时数据可保留。

## 4. 代码复用率

| 模块 | 桌面 | Android | 复用 |
|------|:---:|:---:|:---:|
| conflict.go | ✅ | ✅ | 100% |
| diagnose.go | ✅ | ✅ | 100% |
| scanner.go | ✅ | ✅ | 90%（路径适配） |
| entity*.go | ✅ | ✅ | 100% |
| deprecated.go | ✅ | ✅ | 100% |
| vanilla_types.go | ✅ | ✅ | 100% |
| updater.go | ✅ | ✅ | 80% |
| downloader.go | ✅ | ❌ | 0%（Android 原生实现） |
| mirror.go | ✅ | ⚠️ | 50%（URL 拼接逻辑） |
| github_api.go | ✅ | ⚠️ | 30%（Kotlin/OkHttp 替代） |
| selfupdate.go | ✅ | ❌ | 不需（Google Play） |
| game_link.go | ✅ | ❌ | 不需 |
| Vue 组件（6 个） | ✅ | ❌ | 0%（Compose 重写） |

**核心逻辑复用率：~70%**。冲突检测引擎零修改。

## 5. 工作量估算

| 阶段 | 内容 | 时间 |
|------|------|------|
| Phase 1 | Unciv fork + gomobile 集成 + Go .so 跑通 | 3-5 天 |
| Phase 2 | Compose UI（模组库 + 冲突 + 诊断） | 5-7 天 |
| Phase 3 | 下载队列 + 镜像 + 存档管理 | 3-4 天 |
| Phase 4 | 在线浏览 + 更新检查 + 设置 | 2-3 天 |
| Phase 5 | 测试 + 签名 + 发布 | 2-3 天 |
| **总计** | | **15-22 天** |

## 6. 注意事项

- Unciv GPL-3.0 ⇒ Fork 必须开源
- Android NDK + gomobile 环境搭建
- Go .so ~5-8MB，APK 增大 ~30%
- JNI 调用开销可忽略（冲突检测 CPU 密集，Go 快于 Kotlin）
- 首版仅 arm64-v8a（覆盖 95% 设备）
