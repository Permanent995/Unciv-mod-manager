# Unciv Mod Manager

一个专为 Unciv 设计的现代化模组管理工具。

该项目通篇使用AI生成，vibecoding产品，本人只在AI陷入困境时进行少量修改。
在此感谢lytvpk作者的分享，我从他的项目中学到了很多，这个umm也是参考了他很多的设计思想,以及unciv原版的很多功能也放上来了，在做这个项目的时候unciv的定位模组错误功能还没那么强，可是现在已经比较完备了，有点出乎我的意料
## 🚀 功能特性

### 模组管理
- **智能扫描**: 自动扫描 `mods/` 目录，读取 `ModOptions.json` 元数据
- **分类展示**: 按规则集、扩展、图形、音频、地图等自动分类
- **在线浏览**: 浏览 GitHub 上 1000+ Unciv 模组，按热度/名称/更新时间排序
- **模组诊断**: 检测废弃 unique 语法、缺失类型引用、自我升级循环等问题
- **冲突检测**: 三阶段覆盖分析（同名实体、replaces 链、MergeAction 感知）
- **批量更新检查**: 基于 Unciv 的 `ModListCache.json` 零 API 检测更新

### 下载与加速
- **镜像加速**: 内置 ghproxy/kkgithub/nuaa 等多条线路，自动测速选最快
- **断点续传**: Truncate+WriteAt 分块下载，暂停继续不丢进度
- **下载队列**: 最多 2 并行 + 排队，进度实时推送
- **自动解包**: 下载完成自动解压至 `mods/`，展平嵌套根目录

### 联机检查
- **快照导出/比对**: 生成模组清单，双方比对确保 Mod 一致

### 存档浏览
- **存档管理**: 浏览 `SaveFiles/` 目录，查看存档详情、文明、回合
- **模组列表**: 显示存档启用了哪些模组
- **删除存档**: 支持删除无用存档

### 工具箱
- **冲突检测**: 实体级覆盖分析 + 类型存在性检查 + 差异对比
- **崩溃报告**: 自动匹配 `lasterror.txt` 中的错误模式
- **模组诊断**: 废弃 unique 语法检查、依赖验证

### 其他
- **一键启动 Unciv**: 侧边栏直接启动 Unciv.exe/jar
- **地图管理**: `.civ5map` 文件识别、Wesnoth 格式转换、剪贴板导入
- **README 翻译**: 支持微软/Yandex/自定义 AI 翻译

## 🛠️ 技术架构

### 后端 (Go)
- **框架**: Wails v2
- **模组扫描**: `gjson` 解析 JSON，容错 trailing comma
- **镜像代理**: 多线路并发测速，自动回退
- **并发下载**: goroutine + channel 信号量（限 3 worker）

### 前端 (Vue 3 + TypeScript)
- **框架**: Vue 3 + Vite + TypeScript
- **UI**: 原生 CSS 变量主题系统，无第三方 UI 库
- **实时通信**: Wails EventsOn 推送下载进度
- **跨语言类型安全**: Wails 自动生成 `.d.ts` 绑定

## 📦 安装使用

### 直接运行
下载 `build/bin/unciv-mod-manager.exe` 双击运行，首次启动选择 Unciv 目录即可。

### 自行编译
```bash
# 安装 Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式
wails dev

# 生产构建
wails build
```

### 使用说明
1. **选择目录**: 首次启动选择 Unciv 根目录（需包含 `mods/` 和 `GameSettings.json`）
2. **扫描模组**: 进入「模组库」查看已安装模组
3. **浏览下载**: 进入「模组发现」搜索在线模组
4. **冲突检测**: 工具箱 → 冲突检测，分析模组兼容性
5. **联机检查**: 导出快照 → 与联机对象比对确保一致

## 📄 开源协议

本项目以 GNU General Public License v3.0 only 授权发布。

分发二进制、修改版或衍生版本时，请遵守 GPLv3 关于对应源码、版权声明、许可证文本和修改说明等要求。第三方依赖与资源遵循各自许可证。
