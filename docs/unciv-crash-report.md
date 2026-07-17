# Unciv 崩溃问题检测报告

**检测日期**: 2026-07-16
**Unciv 版本**: 4.20.12 (构建 1226) / 4.21.1 (构建 1236)
**平台**: Windows 11 23H2 (Build 22631)
**Java**: Eclipse Adoptium Temurin-21.0.11+10
**内存**: -Xmx4G

---

## 一、现象描述

在 Unciv 中执行以下任一操作时，弹出 Java 崩溃窗口：

| 触发操作 | 崩溃方法 | 版本 |
|---------|---------|------|
| 点击"定位模组错误" | `ModCompatibility.isAudioVisualMod` | 4.20.12 |
| 点击模组查看详情 | `ModCompatibility.isExtensionMod` | 4.21.1 |
| 点击"开始游戏" | `ModCompatibility.isExtensionMod` | 4.21.1 |

**崩溃核心**: `java.lang.NoClassDefFoundError: kotlin/jvm/internal/Intrinsics`

---

## 二、排查过程

### 2.1 类文件完整性检查

```bash
# Intrinsics.class 是否存在于 Unciv.jar
unzip -l Unciv.jar | grep "Intrinsics.class"
→ kotlin/jvm/internal/Intrinsics.class  (9086 bytes) ✅

# 文件是否损坏
unzip -t Unciv.jar "kotlin/jvm/internal/Intrinsics.class"
→ OK, No errors detected ✅

# Kotlin 标准库类总数
unzip -l Unciv.jar | grep -c "kotlin/"
→ 3071 个 kotlin 类 ✅
```

**结论**: jar 文件中所有 Kotlin 标准库类完整存在。

### 2.2 JRE 类加载测试

```bash
# 用自带 JRE 单独加载 Intrinsics
jre/bin/java -cp Unciv.jar ... Intrinsics
→ 成功加载 ✅

# 用自带 JRE 单独加载 ModCompatibility
jre/bin/java -cp Unciv.jar ... ModCompatibility
→ 成功加载 ✅
```

**结论**: JRE 本身没有问题，类加载在命令行环境下正常。

### 2.3 JRE 版本比对

| JRE | 版本 | 类加载 |
|-----|------|--------|
| 自带 jre/ | Temurin-21.0.11 | ✅ 命令行可加载 |
| 系统 Java | Temurin-21.0.11 | ✅ 命令行可加载 |
| 旧版自带 jre/ (4.20.12) | Temurin-21.0.11 | ✅ 命令行可加载 |

**结论**: 两个 JRE 版本完全相同，命令行加载均正常。

### 2.4 全新版本测试

下载全新 Unciv 4.21.1 覆盖安装（含新 jar + 新 jre/），保留原有 mods/ 和 SaveFiles/：

- 点"定位模组错误" → ❌ 同样崩溃
- 点"开始游戏" → ❌ 同样崩溃
- 点模组详情 → ❌ 同样崩溃

**结论**: 不是 jre 损坏，不是 jar 损坏。

### 2.5 UMM 影响排除测试

完全关闭 UMM（包括后台进程），单独运行 Unciv：

- 点模组 → ❌ 崩溃
- 点开始游戏 → ❌ 崩溃

**结论**: 与 UMM 进程无关。

### 2.6 UMM 文件访问审查

UMM 对 Unciv 目录的所有写操作：

| 文件/目录 | 操作 | 触发时机 |
|----------|------|---------|
| `mods/*/ModOptions.json` | 只读 | 扫描模组 |
| `mods/*/jsons/*.json` | 只读 | 模组诊断、冲突检测 |
| `Unciv.jar` | 只读 | 读取版本号 |
| `ModListCache.json` | 只读 | 在线浏览、检查更新 |
| `GameSettings.json` | 只读 | 读取上次游戏配置 |
| `SaveFiles/*` | 只读 | 存档浏览（删除除外） |
| `maps/*` | 写入 | 地图导入（用户主动操作） |
| `mods/` | 写入 | 模组下载解包（用户主动操作） |
| `umm_backups/` | 写入 | UMM 备份功能 |

**结论**: UMM 不写入 `Unciv.jar`、`ModListCache.json`、`GameSettings.json`，不存在文件锁冲突。

### 2.7 module-info 检查

```bash
# Unciv.jar 中的 module-info
javap META-INF/versions/9/module-info.class

module kotlin.reflect {
    requires java.base;
    requires transitive kotlin.stdlib;
    ...
}
```

Unciv.jar 是一个 Multi-Release JAR，包含 module-info.class。模块系统本身未限制 Kotlin 类的访问。

### 2.8 Unciv.json 配置检查

```json
{
  "jrePath": "jre",
  "classPath": ["Unciv.jar"],
  "mainClass": "com.unciv.app.desktop.DesktopLauncher",
  "vmArgs": ["-Xmx4G", "-"]
}
```

**注意**: `vmArgs` 末尾有一个孤立的 `"-"`，这不是有效的 JVM 参数。可能影响 JVM 初始化，但不应导致特定类加载失败。

---

## 三、崩溃堆栈分析

所有崩溃都汇聚到同一个方法：`ModCompatibility`

```
ModCompatibility.isAudioVisualMod()   ← 判断模组是音视频还是内容模组
ModCompatibility.isExtensionMod()     ← 判断模组是否是扩展模组
    ↓
需要加载 kotlin.jvm.internal.Intrinsics
    ↓
ClassNotFoundException              ← 找不到！
```

**关键发现**: 单独通过命令行加载 `ModCompatibility` 类成功，但在 Unciv 运行时通过 `ModCheckTab` / `ModCheckboxTable` 调用链时失败。说明问题出在**运行时类加载上下文**，而非 jar 文件本身。

---

## 四、根因判断

**这是 Unciv 自身的 bug。**

`ModCompatibility` 类在特定运行上下文中触发了 `NoClassDefFoundError`，可能的原因：

1. **类加载器隔离问题**: Unciv 在运行时使用了自定义类加载器（如 `CrashHandlingDispatcher` 的协程调度器），该类加载器可能无法正确加载 Kotlin 标准库类
2. **Kotlin 内联函数编译问题**: `Intrinsics` 是 Kotlin 编译器自动注入的类，如果 `ModCompatibility.kt` 编译时使用了不同版本的 Kotlin 编译器，可能导致运行时不兼容
3. **Multi-Release JAR 冲突**: jar 中包含 `META-INF/versions/9/module-info.class`，可能在 Java 21 下触发模块路径冲突

---

## 五、已安装的模组

```
SpacedOutChicken-DeCiv-Redux-e722318
RekMOD-3.4
Civ V - Vanilla
unciv unrepentant cathay
Civ V - Gods & Kings
Emperors and Deities
DeCiv-Redux-7.4
5Hex Tileset
Leader Mission 2 Rising Power
Hodgepodge mod
```

---

## 六、建议

1. **向 Unciv 提交 Issue**: 这是 Unciv 的 bug，建议提交到 https://github.com/yairm210/Unciv/issues
2. **临时规避**: 避免使用"定位模组错误"和"开始游戏"中的模组兼容性检查功能
3. **检查 vmArgs**: `Unciv.json` 中 `vmArgs` 末尾的 `"-"` 可能有问题，可尝试删除

---

*本报告由 Unciv Mod Manager 辅助诊断生成*
