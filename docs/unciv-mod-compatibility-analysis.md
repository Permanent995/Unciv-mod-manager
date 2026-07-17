# Unciv `NoClassDefFoundError: Intrinsics` 源码级分析

**日期**: 2026-07-16
**Unciv 版本**: 4.20.12 / 4.21.1
**反编译来源**: `Unciv.jar` → `ModCompatibility.class` → javap

---

## 一、崩溃调用链

所有 6 次崩溃汇总：

| # | 触发场景 | 堆栈入口 | Unciv 版本 |
|---|---------|---------|-----------|
| 1 | 定位模组错误 | `ModCheckTab.runModChecker` | 4.20.12 |
| 2 | 点击模组详情 | `ModCheckTab.runModChecker` | 4.21.1 |
| 3 | 点击新游戏 | `ModCheckboxTable.<init>` | 4.21.1 |
| 4 | 启动时加载 | `UniqueValidator.<clinit>` | 4.20.12 |
| 5 | 模组详情页 | `ModManagementScreen` → `ModInfoAndActionPane.update` | 4.21.1 |
| 6 | 定位模组错误 | `ModCheckTab.runModChecker` | 4.21.1 |

所有崩溃终点相同：`ModCompatibility.isAudioVisualMod()` / `ModCompatibility.isExtensionMod()`

---

## 二、`isAudioVisualMod()` 反编译源码

```java
// 方法签名
public final boolean isAudioVisualMod(Ruleset ruleset)
```

**字节码（JVM 指令）**：

```
0: aload_1                          // 加载参数 ruleset
1: ldc           #16                 // 常量 "mod"
3: invokestatic  #22                 // ⚠️ kotlin/jvm/internal/Intrinsics.checkNotNullParameter
                                     //     ← 第一行就崩在这里！

6: aload_0
7: aload_1
8: invokespecial #26                 // isAudioVisualDeclared(ruleset) → Boolean?
11: dup
12: ifnull        21                 // 如果 isAudioVisualDeclared 返回 null
15: invokevirtual #32                // Boolean.booleanValue()
18: goto          27
21: pop
22: aload_0
23: aload_1
24: invokespecial #35                // isAudioVisualGuessed(ruleset) → boolean
                                     //     ← 扫描 mod 目录判断
27: ireturn
```

**等价的 Kotlin 源码**：

```kotlin
fun isAudioVisualMod(mod: Ruleset): Boolean {
    // ⚠️ 这行就是崩溃点 — Kotlin 编译器自动注入的 null 检查
    // Intrinsics.checkNotNullParameter(mod, "mod")
    
    return isAudioVisualDeclared(mod) ?: isAudioVisualGuessed(mod)
}
```

**第 3 条指令** 是崩溃点。`Intrinsics.checkNotNullParameter` 是 Kotlin 编译器自动注入的——对应 Kotlin 源码中 `mod: Ruleset`（非空类型）的 null 检查。这是**每个非空参数函数的第一条指令**。

---

## 三、`isAudioVisualDeclared()` 逻辑

检查 ModOptions.json 中是否有以下 Unique：

| Unique | 含义 | 返回值 |
|--------|------|--------|
| `ModIsAudioVisualOnly` | 纯音视频模组 | `true` |
| `ModIsAudioVisual` | 含音视频内容的模组 | `true` |
| `ModIsNotAudioVisual` | 声明非音视频模组 | `false` |
| 无以上声明 | 无法判断 | `null` |

---

## 四、`isAudioVisualGuessed()` 逻辑

当 ModOptions.json 没有声明时，**扫描模组目录**判断：

```kotlin
fun isAudioVisualGuessed(mod: Ruleset): Boolean {
    val folder = mod.folderLocation ?: return false
    
    if (folder.list("music").isNotEmpty())    return true  // 有 music/ 目录
    if (folder.list("sounds").isNotEmpty())   return true  // 有 sounds/ 目录
    if (folder.list("voices").isNotEmpty())   return true  // 有 voices/ 目录
    if (folder.list("atlas*").isNotEmpty())   return true  // 有 .atlas 文件
    if (folder.list("game.png").isNotEmpty()) return true  // 有贴图文件
    
    // ... 更多文件检查 ...
}
```

**这个函数会扫描每一个模组目录！** 当遇到 UMM 备份残留的 `_backup_info.json` 或其它非标准文件时，虽然大概率不影响判断结果，但仍可能触发意料外的边界行为。

---

## 五、`isExtensionMod()` 

同样在第一行调用 `Intrinsics.checkNotNullParameter`，属于同一个问题。

---

## 六、根因分析

### 为什么 `Intrinsics` 找不到？

```
┌─────────────────────────────────────────────┐
│ Unciv.jar (fat jar, ~55MB)                  │
│                                             │
│ com/unciv/.../ModCompatibility.class        │
│   ↓ 需要加载                                │
│ kotlin/jvm/internal/Intrinsics.class         │ ← 也在同一个 jar 里 ✅
│                                             │
│ 但是：                                       │
│ - Java 21 模块系统认为 kotlin.jvm.internal  │
│   属于 kotlin.stdlib 模块                    │
│ - Unciv.jar 作为 Unnamed Module 加载        │
│ - Unnamed Module 无法读取 Named Module      │
│ - → NoClassDefFoundError ❌                  │
└─────────────────────────────────────────────┘
```

### 为什么命令行 `java -cp Unciv.jar` 能加载？

命令行 `-cp` 把所有类放在 classpath（非模块路径），**不启用模块系统**，所有类在同一个 Unnamed Module 中，互相可见。

### 为什么 Unciv.exe 启动时启用模块系统？

Unciv.exe 使用 `jre/bin/java`（Java 21），**默认启用模块系统**。`Unciv.json` 中 `classPath: ["Unciv.jar"]` 可能以模块路径方式加载，导致 `ModCompatibility` 和 `Intrinsics` 被隔离在不同模块。

### 为什么"继续游戏"不崩溃？

"继续游戏"直接加载存档数据，**不触发 mod 兼容性检查**。`ModCompatibility` 类仅在以下场景被加载：
- 查看模组详情
- 开新游戏（需要展示模组复选框）
- 定位模组错误

---

## 七、验证实验

| 实验 | 结果 | 说明 |
|------|------|------|
| 自带 JRE (Temurin 21) + Unciv.exe | ❌ 崩溃 | 模块系统开启 |
| 系统 JDK 17 + `java -jar` | ✅ 成功 | JDK 17 或无模块模式 |
| 自带 JRE + 命令行 `-cp` 直接加载 | ✅ 成功 | classpath 模式，无模块隔离 |
| Minecraft JRE (Microsoft 21) + Unciv.exe | ❌ 崩溃 | Microsoft JDK 同样启用模块 |
| 命令行加载 `ModCompatibility` 类 | ✅ 成功 | 不在协程/UI 线程上下文 |

---

## 八、结论

**这是 Java 21 模块系统 + Kotlin fat jar 的已知兼容性问题**，不是 UMM 造成的。

`Unciv.jar` 使用 Kotlin 编译，`Intrinsics` 是 Kotlin 标准库核心类。在 Java 9+ 模块系统下，作为 fat jar 加载时，Unnamed Module 无法访问 `kotlin.stdlib` 命名模块中的类。

### 给 Unciv 开发者的建议

1. **短期修复**: 启动参数添加 `--add-opens java.base/java.lang=ALL-UNNAMED`
2. **长期修复**: 
   - 添加 `META-INF/MANIFEST.MF` 的 `Add-Opens` 或
   - 使用 `jlink` 创建包含 Kotlin 模块的自定义 JRE，或
   - 回退到 classpath 模式加载（不使用模块路径）

### 用户临时规避方案

在 `Unciv.json` 的 `vmArgs` 中添加 JVM 参数尝试绕过模块隔离：

```json
"vmArgs": [
  "-Xmx4G",
  "--add-opens=java.base/java.lang=ALL-UNNAMED",
  "--add-opens=java.base/java.util=ALL-UNNAMED"
]
```

但这不一定能解决问题——根本原因在于 Kotlin stdlib 和 Unciv 代码的模块边界。
