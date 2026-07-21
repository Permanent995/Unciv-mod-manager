# UMM 设计规范

> 本文档以 **UIUX Pro Max** 规范为核心框架，定义 Unciv Mod Manager 的统一设计语言。
> 所有新建页面必须严格遵循此规范，保证全局风格一致。

---

## 目录

1. [设计优先级框架](#1-设计优先级框架)
2. [配色系统](#2-配色系统)
3. [字体系统](#3-字体系统)
4. [间距系统](#4-间距系统)
5. [图标规范](#5-图标规范)
6. [动画与过渡](#6-动画与过渡)
7. [可访问性](#7-可访问性)
8. [交互与触控](#8-交互与触控)
9. [性能](#9-性能)
10. [布局与响应式](#10-布局与响应式)
11. [组件样式](#11-组件样式)
12. [页面结构约定](#12-页面结构约定)
13. [设计反模式](#13-设计反模式)
14. [CSS 变量引用规则](#14-css-变量引用规则)
15. [常用 class 命名约定](#15-常用-class-命名约定)
16. [交付前检查清单](#16-交付前检查清单)

---

## 1. 设计优先级框架

所有设计决策按以下优先级权衡，高优先级项不可为低优先级让步：

| 优先级 | 类别 | 说明 |
|--------|------|------|
| P1 — 关键 | 可访问性 | 对比度、焦点环、标签 |
| P2 — 关键 | 交互与触控 | 触控目标、反馈、禁用态 |
| P3 — 高 | 性能 | 动画合成器、布局偏移、加载态 |
| P4 — 高 | 布局与响应式 | 对齐、间距、最小尺寸、溢出 |
| P5 — 中 | 排版与颜色 | 字体层级、行高、色板一致性 |
| P6 — 中 | 动画 | 时长、缓动、尊重 reduced-motion |
| P7 — 中 | 风格选择 | 设计模式一致性 |
| P8 — 低 | 图表与数据 | 配色、可访问性 |

---

## 2. 配色系统

所有色值通过 CSS 变量定义在 `App.vue` 的 `:root` / `[data-theme="dark"]` 中，组件内**禁止硬编码色值**，一律引用变量。

### 2.1 语义色（项目令牌）

| 变量 | 亮色值 | 暗色值 | 用途 |
|------|--------|--------|------|
| `--accent` | `#4f46e5` | 同左 | 主色：按钮、链接、选中态、焦点环 |
| `--accent-hover` | `#6366f1` | 同左 | 主色 hover 态 |
| `--success` | `#10b981` | 同左 | 成功／可用 |
| `--warning` | `#f59e0b` | 同左 | 警告／排队／注意 |
| `--danger` | `#ef4444` | 同左 | 危险／错误／删除 |

### 2.2 背景色

| 变量 | 亮色 | 暗色 | 用途 |
|------|------|------|------|
| `--bg-primary` | `#ffffff` | `#0f172a` | 页面主背景 |
| `--bg-secondary` | `#f8fafc` | `#1e293b` | 次要背景（侧栏面板、分类条） |
| `--bg-sidebar` | `#f1f5f9` | `#1e293b` | 侧边栏背景 |
| `--bg-card` | `#f1f5f9` | `#1e293b` | 卡片、工具区块 |
| `--bg-input` | `#ffffff` | `#1e293b` | 输入框背景 |
| `--bg-hover` | `#e2e8f0` | `#334155` | 通用 hover 态 |
| `--bg-active` | `#cbd5e1` | `#475569` | 选中／激活态 |
| `--sidebar-active` | `#e2e8f0` | `#334155` | 侧栏导航选中 |

### 2.3 文字色

| 变量 | 亮色 | 暗色 | 用途 | 对比度参考 |
|------|------|------|------|-----------|
| `--text-primary` | `#1e293b` | 同左 | 正文、标题 | 对白底 >13:1 |
| `--text-secondary` | `#475569` | 同左 | 次要文字、按钮文字 | 对白底 7:1 |
| `--text-muted` | `#94a3b8` | 同左 | 辅助文字、占位符、时间戳 | 仅供装饰性文字使用 |

### 2.4 边框与阴影

| 变量 | 亮色 | 暗色 | 用途 |
|------|------|------|------|
| `--border-color` | `rgba(203,213,225,0.5)` | `rgba(51,65,85,0.5)` | 分割线、卡片边框、输入框边框 |
| `--card-shadow` | `0 1px 2px 0 rgba(0,0,0,0.05)` | `none` | 卡片和工具区块阴影 |

### 2.5 暗色主题

通过 `[data-theme="dark"]` 选择器覆盖背景色和边框/阴影变量。语义色和文字色不覆盖（亮色值已在暗色背景满足对比度要求）。

---

## 3. 字体系统

### 3.1 字体栈

**主字体：**
```css
font-family: 'Microsoft YaHei', '微软雅黑', -apple-system, BlinkMacSystemFont,
             'Segoe UI', Roboto, sans-serif;
```
Windows 首选微软雅黑，macOS/Web 回退系统字体。

**等宽字体（代码、路径、堆栈）：**
```css
font-family: 'Cascadia Code', 'Fira Code', monospace;
```

### 3.2 字号与字重

| 层级 | 字号 | 行高 | 字重 | 场景 |
|------|------|------|------|------|
| Page title / h1 | `24px` | `1.4` | `600` (Semibold) | 页面标题 |
| Section h1 | `22px` | `1.4` | `700` (Bold) | 存档页标题 |
| Section h2 | `18px` | `1.5` | `600` (Semibold) | 工具区块标题、详细面板 |
| Block title | `15px` | `1.5` | `600` (Semibold) | 卡片标题 (`.card-title`) |
| Body | `15px` | `1.6` | `400` | 页面正文（`body` 默认） |
| Secondary | `14px` | `1.5` | `400` | 侧栏标签、输入框文字 |
| UI label | `13px` | `1.4` | `500` / `600` | 按钮、表单标签、小标题 |
| Small | `12px` | `1.4` | `400` | 描述文字、时间戳 |
| X-small | `11px` | `1.4` | `400` | 辅助信息、存档子文本 |
| Mini | `10px` | `1.4` | — | 版本号标签 |

### 3.3 行宽

- 正文段落每行 **65–75 字符** 为宜
- 超出此宽度应考虑增加内边距或限制 max-width

---

## 4. 间距系统

采用 4px 网格基础，所有间距值为 4 的倍数或常见分数。

### 4.1 页面布局

| 位置 | 值 |
|------|-----|
| 页面内边距（`.main-content`） | `20px` |
| 页面标题下边距 | `20px`（h1）、`14px`（含 subtitle） |
| 标题与副标题间距 | `4px` |
| 卡片间距（`.settings-grid`） | `12px` |
| 侧栏默认宽度 | `220px`（可拖拽 120-400px） |

### 4.2 卡片与区块

| 位置 | 值 |
|------|-----|
| 卡片内边距（`.card`） | `18px` |
| 工具区块内边距（`.tool-section`） | `16px` |
| 卡片标题与描述间距 | `2px` |
| 描述与内容间距 | `10px` |
| 输入框内边距 | `7-8px 10-12px` |

### 4.3 组件间距

| 场景 | 值 |
|------|-----|
| 按钮内边距（大） | `8px 20px` |
| 按钮内边距（中） | `6px 14px` |
| 按钮内边距（小） | `4px 10px` |
| 侧栏导航项内边距 | `12px 20px` |
| 侧栏图标与文字间距 | `12px` |
| Tab 间距 | `4px` |
| Flex 默认间距 | `8px`（gap） |
| Flex 大间距 | `14px`（gap） |

### 4.4 z-index 层级

| 层级 | 值 | 用途 |
|------|----|------|
| 基础内容 | `auto` | 正常文档流 |
| 浮动元素 | `10` | 拖动手柄、粘性标题 |
| 弹出层 | `30` | 下拉菜单、tooltip |
| 模态遮挡 | `50` | 模态框遮罩 |
| 通知 | `100` | Toast、全局通知 |

---

## 5. 图标规范

### 5.1 图标源

**禁止使用 Emoji 作为 UI 图标**（如 📦 🔍 🧰 ℹ️）。统一使用 SVG 图标集：

- 推荐 [Heroicons](https://heroicons.com)（MIT，24×24 outline 风格）
- 备选 [Lucide](https://lucide.dev)（ISC，兼容 Heroicons 风格）

### 5.2 技术要求

- SVG viewBox：`0 0 24 24`
- 图标尺寸：`width: 20px; height: 20px`（侧栏导航）、`width: 24px; height: 24px`（标题/大图标）
- 填充色通过 `currentColor` 继承文字色
- hover 态使用 `transition: color 0.2s`，禁止缩放过渡（避免布局偏移）

### 5.3 过渡期间约定

当前项目仍使用 Emoji。**改造计划**：按照优先级逐步替换，新页面从设计起直接使用 SVG 图标，旧页面在下次重构时替换。

---

## 6. 动画与过渡

### 6.1 时长规范

| 类型 | 时长 | 适用场景 |
|------|------|---------|
| 微交互 | `150-200ms` | hover、active、focus 等即时反馈 |
| 常规过渡 | `200-300ms` | 面板展开/收起、切换 |
| 复杂过渡 | `≤400ms` | 页面切换、大型布局变化 |
| 退出动画 | 进入时长的 `60-70%` | 弹窗关闭、删除项消失 |

### 6.2 动画属性

- **优先**使用 `transform` 和 `opacity` 做动画（GPU 合成）
- **禁止**对 `width`、`height`、`top`、`left` 做动画（触发重排）
- 缓动函数：`ease` 或 `cubic-bezier(0.4, 0, 0.2, 1)`

### 6.3 实际项目约定

| 场景 | 当前值 | 说明 |
|------|--------|------|
| 进度条填充 | `transition: width 0.3s` | 例外：进度条必须用 width |
| 卡片 hover 阴影 | `transition: box-shadow 0.2s` | 仅亮色生效 |
| 导航选中态 | `transition: background 0.2s` | |
| Spinner | `animation: spin 0.8s linear infinite` | |

### 6.4 尊重 `prefers-reduced-motion`

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 7. 可访问性

### 7.1 对比度

- 正文（`--text-primary`）与背景对比度 **≥ 4.5:1**（当前值 >13:1 ✅）
- 次要文字（`--text-secondary`）与背景对比度 **≥ 3:1**（当前值 ≈7:1 ✅）
- 仅装饰性文字可使用 `--text-muted`
- 错误提示不应仅靠颜色区分，需配合图标或文字（如 `错误的URL` + 红色边框）

### 7.2 焦点环

- 所有可交互元素（按钮、输入框、链接、可点击项）须有可见的 `:focus-visible` 样式
- 标准：`outline: 2px solid var(--accent); outline-offset: 2px`
- 使用 `:focus-visible` 而非 `:focus`，避免鼠标点击时也显示焦点环

### 7.3 标签与语义

- 表单输入项必须关联 `<label>`（for/id 配对或包裹）
- 图标按钮须提供 `aria-label`
- Tab 顺序（tabindex）须与视觉顺序一致

### 7.4 当前项目不符合项（待修复）

- 输入框 `focus` 缺少 `outline`，仅 `border-color` 变化 — 需加 `outline: 2px solid var(--accent); outline-offset: 1px`
- 列表项（存档、导航）点击态仅靠背景色区分 — 需确保没有在纯色旁使用纯色
- 无 `prefers-reduced-motion` 支持

---

## 8. 交互与触控

### 8.1 触控目标

- 桌面端最小触控目标：**24×24px**（按钮/可点击区域）
- 所有可点击元素须设置 `cursor: pointer`
- 紧密排列按钮之间至少 `8px` 间距

### 8.2 反馈规范

| 状态 | 视觉反馈 |
|------|---------|
| 默认 | 正常样式 |
| `:hover` | 背景/边框/文字色变化，150-200ms 过渡 |
| `:active` | 压暗效果（`opacity: 0.85` 或深色背景） |
| `:disabled` | `opacity: 0.5-0.6` + `cursor: not-allowed` |
| loading | 按钮文字变 "加载中..." + disabled |
| 错误 | 错误文字靠近触发控件（字段下方或附近） |

### 8.3 禁用态约定

- 禁用按钮不触发 hover/active 样式
- 异步操作（下载、分析、扫描）开始时立即切换为 disabled
- 操作完成后恢复

---

## 9. 性能

### 9.1 布局偏移

- 异步加载内容须预留占位空间，避免加载后页面跳动
- 进度条等动态元素须有固定高度（`4px`）
- Toast 提示采用 fixed/absolute 定位，不占用文档流

### 9.2 加载态

- 耗时操作（>300ms）须显示加载指示器
- 按钮内loading替换文字，不在按钮外额外显示
- 数据列表为空时显示 `.empty-state` 占位

---

## 10. 布局与响应式

### 10.1 页面约束

- 桌面端内容区最大宽度：`1100px`（工具箱）、`640px`（设置页）
- 无水平滚动条（`overflow: hidden` 由布局保证）
- 使用 `min-height: 0` 防止 flex 溢出

### 10.2 页面高度

- 主内容区撑满视口剩余高度（`flex: 1; overflow: hidden`）
- 滚动发生在内部容器，而非整个页面
- 列表/条目区域最大高度受视口约束（如 `.mirror-list` 的 `max-height: 200px`）

---

## 11. 组件样式

### 11.1 设计模式

本项目以 **Bento Grid（卡片化布局）** 为主要设计模式：
- 设置页：垂直排列的独立卡片
- 工具箱：区块式 tool-section + 内部面板布局
- 存档页：master-detail 双栏布局

不使用毛玻璃（Glassmorphism）和新拟态（Neumorphism），保持扁平清晰。

### 11.2 按钮

| 类型 | 背景 | 文字色 | 边框 | 圆角 | hover | disabled |
|------|------|--------|------|------|-------|---------|
| Primary (`btn-primary`) | `var(--accent)` | `#fff` | 无 | `4px` | opacity 0.85 | opacity 0.6 |
| Outline (`btn-outline`) | 透明 | `var(--text-secondary)` | `1px solid var(--border-color)` | `4px` | 边框+文字→ accent | opacity 0.5 |
| Ghost (`btn-retest`) | 透明 | `var(--text-secondary)` | `1px solid var(--border-color)` | `4px` | 边框+文字→ accent | opacity 0.5 |
| Small (`btn-sm`) | 透明 | `var(--accent)` | `1px solid var(--accent)` | `3px` | accent 背景+白字 | opacity 0.5 |
| Small danger | 透明 | `var(--danger)` | `1px solid var(--danger)` | `3px` | danger 背景+白字 | opacity 0.5 |
| Title bar (`tb-btn`) | 透明 | `var(--text-secondary)` | 无 | `4px` | border 背景 | — |
| Title close | 透明 | `var(--text-secondary)` | 无 | `4px` | 红 `#e81123` 底白字 | — |
| Tab (`tab-btn`) | `var(--bg-card)` | `var(--text-secondary)` | `1px solid var(--border-color)` | `6px` | 文字→ primary | — |
| Tab active | `var(--accent)` | `#fff` | `var(--accent)` | `6px` | — | — |

### 11.3 卡片

| 属性 | 值 |
|------|-----|
| 背景 | `var(--bg-card)` |
| 圆角 | `10px`（设置卡片）、`8px`（工具区块） |
| 边框 | `1px solid var(--border-color)` |
| 阴影 | `var(--card-shadow)` |
| 内边距 | `18px`（设置卡片）、`16px`（工具区块） |
| hover 增强 | `box-shadow: 0 2px 8px rgba(0,0,0,0.08)`（仅亮色） |
| 布局 | `display: flex; gap: 14px` |

### 11.4 输入框 (Input / Select)

| 属性 | 值 |
|------|-----|
| 背景 | `var(--bg-input)` |
| 边框 | `1px solid var(--border-color)` |
| 圆角 | `4px` |
| 内边距 | `6-8px 10-12px` |
| 文字色 | `var(--text-primary)` |
| 字号 | `13px` 或 `14px` |
| focus | `border-color: var(--accent); outline: 2px solid var(--accent); outline-offset: 1px` |

Select 和 input 样式统一。

### 11.5 侧边栏导航

| 属性 | 值 |
|------|-----|
| 背景 | `var(--bg-sidebar)` |
| 分隔线 | `1px solid var(--border-color)` |
| 导航项内边距 | `12px 20px` |
| 图标宽+间距 | 图标 `20px` + `12px` gap |
| hover | `background: var(--border-color)` |
| active | `background: var(--sidebar-active)` + 左 `3px solid var(--accent)` |
| 启动按钮 | 通栏 `100%`，accent 底白字，`font-size: 14px; font-weight: 600` |

### 11.6 Tab 栏

| 属性 | 值 |
|------|-----|
| 容器 | `display: flex; gap: 4px` |
| Tab | `padding: 8px 20px; border-radius: 6px` |
| 背景 | `var(--bg-card)` |
| 边框 | `1px solid var(--border-color)` |
| active | `background: var(--accent); color: #fff; border-color: var(--accent)` |

### 11.7 进度条

| 属性 | 值 |
|------|-----|
| 容器 | `height: 4px; background: var(--bg-secondary); border-radius: 2px; overflow: hidden` |
| 填充 | `height: 100%; background: var(--accent); transition: width 0.3s` |
| 暂停态 | 填充色变 `var(--warning)` |
| 最小宽度 | `2%` |

### 11.8 标签 / Badge

| 属性 | 值 |
|------|-----|
| 内边距 | `1-2px 5-8px` |
| 圆角 | `3px` |
| 字号 | `10-12px` |
| 字重 | `600` |
| accent 标签 | `background: var(--accent); color: #fff` |
| warning 标签 | `background: var(--warning); color: #fff` |

### 11.9 提示条（Toast / Banner）

| 类型 | 背景 | 文字色 | 特殊 |
|------|------|--------|------|
| 错误 | `rgba(255,107,107,0.1)` | `var(--danger)` | — |
| 成功 | `rgba(78,205,196,0.1)` | `#4ecdc4` | — |
| 诊断 | `rgba(255,107,107,0.08)` | — | 左 3px danger 边框 |
| 建议 | `rgba(74,158,255,0.08)` | — | 左 3px accent 边框 |
| 内边距 | `10-12px` | | |
| 圆角 | `4px` | | |

### 11.10 空状态 / 加载状态

| 属性 | 值 |
|------|-----|
| 对齐 | `text-align: center` |
| 内边距 | `40px` 上下 |
| 文字色 | `var(--text-secondary)` 或 `var(--text-muted)` |
| Spinner | `32×32px`，border `3px solid var(--border-color)`，top `var(--accent)`，`border-radius: 50%`，`animation: spin 0.8s linear infinite` |

### 11.11 条目列表（entries-panel / 冲突报告风格）

| 属性 | 值 |
|------|-----|
| 条目内边距 | `8px` |
| 条目圆角 | `4px` |
| 条目间距 | `4px` |
| 左侧标记 | `3px solid`，色值按级别：safe→success, risk→warning, override→danger, incompatible→#8c8c8c |
| 条目背景 | 各级别对应 `rgba` 浅色背景 |

---

## 12. 页面结构约定

### 12.1 标准页面骨架

```html
<div class="xxx-view">
  <div class="view-header">
    <h1>页面标题</h1>
    <p class="subtitle">描述文字</p>
  </div>
  <!-- 页面内容 -->
</div>
```

### 12.2 工具页面（工具箱风格）

三层结构：Tab 栏 → Section Header（标题 + 操作按钮）→ 内容区（`.tool-section`）。

内容区统一圆角 `8px`、`var(--bg-card)` 背景、`var(--card-shadow)` 阴影。

### 12.3 设置页面（卡片风格）

设置页使用 `.settings-grid`（`flex-direction: column; gap: 12px; max-width: 640px`），每项为 `.card`。

### 12.4 存档页面（master-detail 风格）

双栏布局：左栏列表（`width: 270px; flex-shrink: 0`）+ 右栏详情（`flex: 1`），间距 `14px`。

---

## 13. 设计反模式

| 反模式 | 说明 | 替代方案 |
|--------|------|---------|
| 🚫 Emoji 做图标 | emoji 在各平台渲染不一致，无法自定义颜色 | 使用 Heroicons / Lucide SVG |
| 🚫 仅靠颜色区分状态 | 色盲用户无法感知 | 配合图标、文字、边框标记 |
| 🚫 直接修改 width/height 做动画 | 触发重排，性能差 | 使用 transform: scale() |
| 🚫 无焦点样式 | 键盘用户无法导航 | 加 `:focus-visible` outline |
| 🚫 硬编码色值 | 无法支持暗色模式 | 统一用 CSS 变量 |
| 🚫 模板化 Bootstrap/Tailwind 默认样式 | 缺乏品牌个性 | 先定义设计令牌再构建 |

---

## 14. CSS 变量引用规则

1. **禁止在组件中硬编码色值**（`#xxx`、`rgb`、`rgba`），一律引用 `var(--xxx)`。
2. `var()` 回退值仅在使用第三方库或极端兼容时允许。
3. 所有 CSS 变量集中在 `App.vue` 的 `:root` 和 `[data-theme="dark"]` 中定义，组件只消费不生产。
4. 组件间共享样式通过变量传递，不通过全局 class（`.btn-primary` 等语义 class 例外）。
5. 暗色模式通过 `toggleTheme()` 切换 `data-theme` 属性实现。

---

## 15. 常用 class 命名约定

| Class | 用途 |
|-------|------|
| `.view-header` | 页面标题行（flex + h1 + 操作按钮） |
| `.section-header` | 区块标题行（flex，左 title + 右 button） |
| `.btn-primary` | 主操作按钮 |
| `.btn-outline` | 次要按钮 |
| `.btn-sm` | 小按钮 |
| `.btn-analysis` | 工具区操作按钮（同 primary） |
| `.card` / `.card-body` / `.card-title` / `.card-desc` | 设置卡片 |
| `.tool-section` | 工具箱内容区 |
| `.tab-bar` / `.tab-btn` | Tab 导航 |
| `.error-banner` | 错误提示 |
| `.empty-state` / `.loading-state` | 空/加载占位 |
| `.progress-bar` / `.fill` | 进度条容器/填充 |
| `.dl-bar` / `.dl-input` / `.dl-name` / `.dl-task` | 下载区组件 |
| `.badge-civ` / `.mod-tag` | 标签 |
| `.level-sidebar` / `.cat-sidebar` / `.entries-panel` | 三栏报告布局 |
| `.save-path-bar` | 下载保存路径条 |

---

## 16. 交付前检查清单

### 视觉质量
- [ ] 无 emoji 作为 UI 图标（均使用 SVG）
- [ ] 所有图标来自同一图标集（Heroicons / Lucide）
- [ ] hover 状态不引起布局偏移（禁止缩放过渡）

### 交互
- [ ] 所有可点击元素有 `cursor: pointer`
- [ ] hover 态有清晰视觉反馈（150-200ms）
- [ ] 过渡动画使用 transform/opacity（进度条 width 例外）
- [ ] `:focus-visible` 焦点环已实现
- [ ] 异步操作按钮加载时 disabled

### 浅色 / 暗色模式
- [ ] 浅色模式文本对比度 ≥ 4.5:1
- [ ] 边框在两种模式下均可见
- [ ] 两种模式实际预览验证通过

### 布局
- [ ] 无内容被固定元素遮挡
- [ ] 无水平滚动条
- [ ] 缺省内容有 empty-state 占位

### 可访问性
- [ ] 所有表单输入有 `<label>`
- [ ] 图标按钮提供 `aria-label`
- [ ] 不仅依赖颜色传递信息
- [ ] 尊重 `prefers-reduced-motion`

### 代码规范
- [ ] 无硬编码色值
- [ ] 样式引用 CSS 变量而非直接写值
- [ ] 新 class 名符合本文档第 15 节的命名约定

---

> **规则：每新建一个页面，必须先阅读本 design.md，严格按其中的配色、字体、间距、组件样式、可访问性标准进行设计，确保全项目视觉一致性。**
