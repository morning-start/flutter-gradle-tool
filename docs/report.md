---
title: Flutter Gradle 加速工具方案报告
doc_type: report
version: 0.2.0
status: draft
updated: 2026-06-10
related_docs:
  - ROADMAP.md
  - TODO.md
  - tech-decision.md
---

# 报告：Flutter Gradle 加速工具 —— 基于 Go 的二进制 CLI 方案

## 文档关系

- [ROADMAP.md](ROADMAP.md)：定义阶段目标、里程碑和风险。
- [TODO.md](TODO.md)：定义当前 P1 阶段的执行清单和依赖关系。
- [tech-decision.md](tech-decision.md)：定义库选型与自研边界。
- 本文档：定义产品定位、功能边界、实现约束和验收依据。

## 1. 项目背景

### 1.1 痛点分析

Flutter 项目在构建 Android 版本时，依赖 Gradle 构建系统。整个过程需要从远程仓库下载 Gradle 分发包和 Maven 依赖，在国内网络环境下存在以下问题：

- **Gradle 分发包下载缓慢**：`gradle-wrapper.properties` 中的 `distributionUrl` 默认指向 `services.gradle.org`（境外服务器），下载速度经常只有几十 KB/s，导致首次构建或升级 Gradle 版本时长时间等待。
- **Maven 依赖下载缓慢**：`build.gradle` 中配置的 Maven Central、Google 等仓库同样位于境外，大量依赖项（AndroidX、Flutter 引擎等）的下载耗时巨大。
- **手动配置繁琐且易错**：开发者需要手动修改 `distributionUrl`、替换仓库地址、切换镜像源，操作分散且难以统一管理。
- **团队协作不一致**：不同开发者的镜像源配置可能不同，导致构建行为不一致，CI 环境也需要单独配置。

### 1.2 现有解决方案及不足

| 方案                               | 优点      | 不足                                          |
| ---------------------------------- | --------- | --------------------------------------------- |
| 手动修改 gradle-wrapper.properties | 简单直接  | 每次新建项目都要改；切换源需重新编辑          |
| Shell/Python 脚本批量替换          | 可复用    | 跨平台兼容性差；需依赖 Python/Shell 环境      |
| CI 中硬编码镜像 URL                | CI 侧统一 | 本地开发仍需手动处理；镜像 URL 散落在各配置中 |
| Gradle 初始化脚本 (init.d)         | 全局生效  | 学习成本高；团队共享不便                      |

### 1.3 工具定位

`flutter-gradle-tool` 定位为 Flutter 开发者的本地和 CI 辅助工具，解决 Gradle 构建过程中的网络加速问题。核心价值在于：

- **一键初始化**：自动识别 Flutter 项目结构，替换镜像源
- **交互式切换**：无需记忆镜像 URL，通过菜单选择即可切换
- **跨平台一致**：同一套 CLI 在 Windows/macOS/Linux 上行为一致
- **CI 友好**：非交互模式可直接集成到 CI 流水线

### 1.4 技术选型

选择 Go 语言实现的原因：

- **单二进制分发**：编译后无需运行时依赖，下载即用
- **跨平台编译**：Go 交叉编译原生支持 Windows/macOS/Linux × amd64/arm64
- **标准库丰富**：文件操作、HTTP 客户端、文本解析均无需第三方库
- **启动速度快**：适合 CI 场景，冷启动在毫秒级

---

## 2. 工具设计概述

### 2.1 工具名称

`flutter-gradle-tool`（可简写为 `fgt`）

### 2.2 核心功能模块

| 模块      | 功能                                                                                                                        | 适用场景                               |
| --------- | --------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| `init`    | 初始化项目：将 `gradle-wrapper.properties` 中的 `distributionUrl` 替换为镜像地址；可选修改 `build.gradle` 添加 Maven 镜像。 | 本地新项目初始化、旧项目迁移           |
| `mirror`  | 列出、选择、设置 Gradle 分发镜像和 Maven 镜像。支持官方源与多个国内镜像，可单独设置 wrapper 或 Maven 镜像。                 | 切换镜像源；在不同网络环境下选择最快源 |
| `cache`   | 查看 Gradle 缓存状态（路径、大小、各版本占用）；清理过期或全部缓存。                                                        | 本地磁盘空间管理；调试缓存问题         |
| `doctor`  | 诊断当前项目 Gradle 配置是否正确（wrapper URL 是否为所选镜像、Maven 镜像是否生效、缓存是否存在等）。                        | CI 构建前置检查；本地问题排查          |
| `exec`    | 在配置好的环境中执行任意 Gradle 命令（如 `./gradlew build` 或 `gradlew.bat build`），自动注入环境变量。                     | CI 中统一构建入口                      |
| `version` | 显示工具版本。                                                                                                              | 版本管理                               |
| `help`    | 显示帮助信息。                                                                                                              | 辅助使用                               |

### 2.3 镜像源列表设计

工具内置一个镜像源列表，包含官方源与常见国内镜像。每个镜像源包括：

- **名称**（如 `official`、`tencent`、`aliyun`、`huaweicloud`）
- **显示名称**（如 `Official (services.gradle.org)`、`Tencent Cloud`、`Aliyun`、`Huawei Cloud`）
- **Gradle 分发 URL 模板**：`{url}/gradle-{version}-{type}.zip`
- **Maven 仓库镜像 URL**：用于替换 Maven 中央仓库、Google 等

**初始内置镜像源**：

| 名称          | 显示名称                       | Gradle 分发 URL                             | Maven 镜像 URL                                                     |
| ------------- | ------------------------------ | ------------------------------------------- | ------------------------------------------------------------------ |
| `official`    | Official (services.gradle.org) | `https://services.gradle.org/distributions` | `https://repo.maven.apache.org/maven2/`（不做替换，使用默认）      |
| `tencent`     | Tencent Cloud                  | `https://mirrors.cloud.tencent.com/gradle`  | `https://mirrors.cloud.tencent.com/nexus/repository/maven-public/` |
| `aliyun`      | Aliyun                         | `https://mirrors.aliyun.com/maven/gradle`   | `https://maven.aliyun.com/repository/public`                       |
| `huaweicloud` | Huawei Cloud                   | `https://mirrors.huaweicloud.com/gradle`    | `https://mirrors.huaweicloud.com/repository/maven/`                |

> 注：官方源的 Maven 镜像不设置特殊值，即使用 Gradle 默认的仓库地址（Maven Central、Google 等）。对于其他镜像源，工具将自动将 Maven 仓库替换为对应的镜像地址。

### 2.4 命令行接口设计

```bash
fgt mirror [subcommand] [flags]

Subcommands:
  list            List all available mirror sources (with current selection marked)
  set             Set mirror source for wrapper and/or maven
  current         Show current mirror configuration for the project
  test            Test latency/availability of all mirror sources (optional)

Flags for `set`:
  --source        Mirror source name (e.g., official, tencent, aliyun)
  --wrapper-only  Only change gradle-wrapper.properties distributionUrl
  --maven-only    Only change Maven mirror in build.gradle
  --interactive   Interactive selection (default if no --source given)
  --project-dir   Path to Flutter project root (default: current directory)
  --ci            Non-interactive mode (must provide --source)
```

**示例用法**：

```bash
# 列出所有镜像源
fgt mirror list

# 交互式选择切换
fgt mirror set --interactive

# 直接切换到阿里云镜像（同时修改 wrapper 和 Maven）
fgt mirror set --source aliyun

# 仅修改 wrapper 分发镜像为腾讯云
fgt mirror set --source tencent --wrapper-only

# 仅修改 Maven 镜像为华为云
fgt mirror set --source huaweicloud --maven-only
```

### 2.5 工作流程示例

**本地开发**：

```bash
# 首次初始化，交互式选择镜像源
fgt init --interactive

# 或直接指定源
fgt init --source aliyun

# 后期需要切换源（例如官方源用于调试）
fgt mirror set --source official

# 查看当前配置
fgt mirror current

# 构建（工具自动适配 gradlew / gradlew.bat）
fgt exec build
```

**CI 环境**：

```yaml
- name: Setup Gradle with tool
  run: |
    wget .../fgt-linux-amd64 -O fgt
    chmod +x fgt
    # CI 中先初始化，再指定镜像源
    ./fgt init
    ./fgt mirror set --source tencent --ci
    ./fgt exec build
```

> 注：`fgt exec` 在 Windows 上自动调用 `gradlew.bat`，在 Unix 上调用 `./gradlew`。`build` 为 Gradle 构建的常用 task，也可指定其他 task 如 `assembleRelease`。

---

## 3. 实现指引（Go 语言）

### 3.1 项目目录结构

```
flutter-gradle-tool/
├── cmd/
│   └── fgt/
│       └── main.go                   # 程序入口，路由各子命令
├── internal/
│   ├── mirror/
│   │   ├── source.go                 # 镜像源数据结构 + 内置列表
│   │   ├── config.go                 # 读写 .fgt-config 持久化文件、反向推断镜像源
│   │   ├── test.go                   # 镜像源测速逻辑
│   ├── gradle/
│   │   ├── wrapper.go                # 解析/修改 gradle-wrapper.properties
│   │   ├── maven.go                  # 修改 build.gradle 镜像配置
│   │   └── build.go                  # exec 功能，自动检测平台选择 gradlew/gradlew.bat
│   ├── errors/
│   │   └── exitcode.go               # 统一退出码定义
│   ├── doctor/
│   │   └── doctor.go                 # 诊断逻辑
│   └── cache/
│       └── cache.go                  # 缓存管理逻辑
├── go.mod
├── go.sum
└── docs/
    └── report.md                     # 本报告
```

### 3.2 数据结构定义

在 `internal/mirror/source.go` 中定义：

```go
type Source struct {
    Name        string // 内部标识，如 "aliyun"
    DisplayName string // 显示名称
    GradleURL   string // Gradle 分发基础 URL，不含版本和尾部斜杠
    MavenURL    string // Maven 仓库镜像 URL，为空表示使用默认
}
```

预定义全局列表：

```go
var BuiltinSources = []Source{
    {Name: "official", DisplayName: "Official (services.gradle.org)", GradleURL: "https://services.gradle.org/distributions", MavenURL: ""},
    {Name: "tencent", DisplayName: "Tencent Cloud", GradleURL: "https://mirrors.cloud.tencent.com/gradle", MavenURL: "https://mirrors.cloud.tencent.com/nexus/repository/maven-public/"},
    {Name: "aliyun", DisplayName: "Aliyun", GradleURL: "https://mirrors.aliyun.com/maven/gradle", MavenURL: "https://maven.aliyun.com/repository/public"},
    {Name: "huaweicloud", DisplayName: "Huawei Cloud", GradleURL: "https://mirrors.huaweicloud.com/gradle", MavenURL: "https://mirrors.huaweicloud.com/repository/maven/"},
}
```

### 3.3 关键依赖

| 依赖                             | 用途         | 备注                                |
| -------------------------------- | ------------ | ----------------------------------- |
| `github.com/spf13/cobra`         | CLI 框架     | 子命令路由、标志解析                |
| `github.com/manifoldco/promptui` | 交互式选择   | 可选增强，用于更复杂的镜像选择体验  |
| `go.uber.org/zap` / `log`        | 日志输出     | 可考虑标准库 log 以减小体积         |
| 标准库 `net/http`                | HTTP 请求    | 镜像源测速                          |
| 标准库 `os/exec`                 | 外部命令执行 | `exec` 子命令调用 gradlew           |
| 标准库 `regexp`                  | 正则解析     | 解析 distributionUrl 提取版本和类型 |

### 3.4 修改 `gradle-wrapper.properties` 的逻辑

修改函数接受 `MirrorSource` 参数，根据 `GradleURL` 和从当前 `gradle-wrapper.properties` 中读取到的版本号、分发类型（`-all`、`-bin`、`-src`），构造新的 `distributionUrl`。

**约束**：

- 必须保留原文件中的版本号和类型后缀（例如 `gradle-8.5-all.zip`）。
- 替换前应解析原 URL 提取版本和类型，而不是直接字符串替换整个 URL，以避免因镜像 URL 结构差异导致错误。
- 如果解析失败（原 URL 格式不标准），则报错并提示用户手动检查。
- 如文件不存在，应输出清晰的错误提示，而非 panic。
- 多次执行应保证幂等——若已是目标镜像 URL，则跳过不做修改。

### 3.5 修改 Maven 镜像的逻辑

在 `build.gradle` 中修改 Maven 仓库配置时，根据所选 `MirrorSource` 的 `MavenURL` 来替换或插入仓库地址。

**策略**：

- 如果 `MavenURL` 为空（如官方源），则移除之前由工具添加的镜像配置，恢复为默认。
- 如果 `MavenURL` 非空，则在 `buildscript.repositories` 和 `allprojects.repositories` 的开头插入镜像地址，保留原有仓库作为 fallback。

**约束**：

- 应能识别之前由工具添加的镜像配置，避免重复插入。
- 使用标记注释（如 `// Added by fgt`）来定位和更新，而非每次都简单追加。
- `build.gradle.kts` 与 `build.gradle` 都应支持。实现上按文件后缀分发到对应解析器，避免把 Groovy 语法写入 Kotlin DSL。
- 如果某个 DSL 文件不存在，应继续尝试同目录下的另一种 DSL 文件，不应因为缺少一个文件而阻断初始化流程。

### 3.6 交互式选择实现

优先使用标准库 `bufio`/`fmt` 实现简单菜单。若后续需要搜索、分页或更复杂的展示，可再引入 `github.com/manifoldco/promptui`。

**约束**：

- 在 `--ci` 模式下必须禁用交互，如果缺少 `--source` 参数则报错退出（退出码 1）。
- 交互式提示应包含镜像源的简要信息，测速结果属于后续增强。
- 应高亮显示当前正在使用的镜像源，减少误操作。
- 切换前提示用户确认，避免手滑操作。

### 3.7 镜像源测速（可选高级功能）

实现 `mirror test` 子命令，对每个镜像源的 Gradle 分发 URL 发送 HEAD 请求，测量响应时间。使用 Go 的 `http.Client` 设置超时（如 5 秒），并发测试所有源，输出排序后的列表。

**注意**：

- 测速功能需要网络访问，在 CI 环境中可能被防火墙限制，应允许跳过。
- 并发请求应使用 `sync.WaitGroup` 或 `errgroup` 控制，避免资源耗尽。
- 测速结果应包含成功/失败状态，方便用户判断源是否可用。

### 3.8 持久化当前项目使用的镜像源

在项目根目录下创建一个隐藏文件 `.fgt-config`，记录当前选中的镜像源名称。这样 `mirror current` 可以快速读取，`doctor` 可以对比实际配置文件是否与此记录一致。

**约束**：

- 该文件应加入 `.gitignore`，避免提交到仓库。
- 如果文件丢失，`current` 命令应通过分析 `gradle-wrapper.properties` 中的 URL 反向推断使用的镜像源（通过字符串匹配内置源列表）。
- 文件格式统一使用 JSON，示例为 `{"source":"aliyun"}`，便于后续扩展字段。

### 3.9 Windows 兼容性

| 差异点              | 处理方式                                                                                         |
| ------------------- | ------------------------------------------------------------------------------------------------ |
| 路径分隔符          | 使用 `filepath.Join` 而非字符串拼接                                                              |
| Gradle wrapper 脚本 | `exec` 时检测 `gradlew.bat` 优先于 `gradlew`                                                     |
| 隐藏文件            | Windows 上使用 `ATTRIB +H` 或直接以点开头命名（Go 的 os 包创建 `.fgt-config` 在 Windows 也正常） |
| 换行符              | 读写配置文件时使用平台原生换行符或统一使用 LF                                                    |
| 终端编码            | 交互式提示时避免依赖 UTF-8 特殊字符，使用 ASCII 友好符号                                         |

### 3.10 错误处理策略

| 错误场景                                     | 行为                         | 退出码      |
| -------------------------------------------- | ---------------------------- | ----------- |
| Flutter 项目目录不存在                       | 打印错误并退出               | 1           |
| `gradle-wrapper.properties` 不存在或无法解析 | 打印错误并建议手动检查       | 2           |
| 网络不可达（测速时）                         | 标记该源为不可用，继续其他源 | 0（非致命） |
| 无网络（执行 `mirror test` 全部失败）        | 打印提示并建议检查网络       | 3           |
| `--source` 指定的镜像源名称不存在            | 列出可用源并退出             | 4           |
| `--ci` 模式缺少 `--source`                   | 报错并提示必须指定源         | 5           |
| `build.gradle` 写入权限不足                  | 提示以管理员/root 权限重试   | 6           |
| 未知子命令                                   | 打印帮助信息并退出           | 7           |

### 3.11 测试要点

- 单元测试：验证 `gradle-wrapper.properties` 的 URL 解析与重构是否正确（覆盖正常 URL、异常 URL、不同版本号、不同分发类型）。
- 单元测试：验证 `build.gradle` 插入/移除镜像配置的逻辑（使用临时文件，覆盖空文件、已有工具标记、多种仓库顺序）。
- 单元测试：验证镜像源名称查找的模糊匹配（大小写不敏感、空格忽略）。
- 集成测试：准备多个 Flutter 项目，依次切换到每个内置镜像源，执行 `fgt doctor` 确认无报错，并实际运行 `fgt exec dependencies` 验证依赖下载正常。
- 跨平台测试：在 Windows（PowerShell/cmd）、macOS（zsh）、Linux（bash）上验证行为一致。

---

## 4. 使用注意事项

### 4.1 对于开发者

1. **切换镜像源后需要清缓存吗？** 不需要。不同镜像源提供的 Gradle 分发包内容一致，缓存基于版本号，切换源不会导致重复下载（只要版本相同）。Maven 依赖也类似，不同镜像的 artifact 校验和应相同，通常无需清理。
2. **官方源作为兜底**：如果国内镜像暂时同步延迟或缺失某些新版本，可以切换回 `official` 源，但下载速度会变慢。工具应提示用户注意。
3. **交互式切换时避免误操作**：工具会显示当前使用的源并提示确认。在 CI 中务必使用 `--source` 显式指定。
4. **多模块项目**：对于带有多个 `android` 模块的 Flutter 项目（如 monorepo），需手动指定 `--project-dir` 或确保在当前模块目录执行。

### 4.2 对于团队/CI 管理员

1. **锁定镜像源版本**：可以在团队文档中约定统一使用某个镜像源（如阿里云），并在 CI 脚本中固定指定，避免因镜像不同导致的不一致。
2. **私有镜像源支持**：如果团队内部有私有 Gradle 分发缓存，可以通过配置文件扩展镜像源列表。未来可支持从 JSON/YAML 文件加载自定义源。
3. **CI 安全性**：建议将工具二进制文件托管在内部制品库（如 Nexus、JFrog），而非从公网下载，防止供应链攻击。

### 4.3 故障排除

| 问题现象                                    | 可能原因                          | 解决方式                                                         |
| ------------------------------------------- | --------------------------------- | ---------------------------------------------------------------- |
| 切换源后 `gradle-wrapper.properties` 未更新 | 项目路径不正确或文件权限问题      | 运行 `fgt doctor` 检查；确保在 Flutter 项目根目录执行            |
| 切换 Maven 镜像后构建仍访问国外仓库         | `build.gradle` 中镜像未排在最前面 | 工具应确保插入的镜像仓库在 `repositories` 列表顶部；手动检查顺序 |
| 交互式选择时出现乱码                        | 终端编码问题                      | 使用 UTF-8 终端；或通过 `--source` 参数直接指定                  |
| `fgt exec` 报"找不到文件"                   | Windows 上未检测到 `gradlew.bat`  | 确保 Flutter 项目的 `android/` 目录中已存在 `gradlew.bat`       |
| 测速全部超时/失败                           | 网络环境受限（如企业代理）        | 直接跳过测速，手动指定镜像源                                    |

---

## 5. 非功能性需求

### 5.1 性能指标

- **工具自身启动耗时**：< 100ms（冷启动，不含 Gradle 执行）
- **镜像源测速并发数**：默认 4 并发，可通过环境变量 `FGT_TEST_CONCURRENCY` 覆盖
- **配置文件读写**：< 10ms
- **二进制体积**：< 10MB（压缩后 < 5MB）

### 5.2 兼容性要求

- **操作系统**：Windows 10+、macOS 12+、Linux（glibc 2.17+）
- **架构**：amd64、arm64
- **Flutter 版本**：Flutter 3.x+（支持的 Gradle wrapper 版本范围 7.x ~ 8.x）
- **Go 版本**：Go 1.21+（开发编译用）

### 5.3 安全考虑

- 工具不处理任何凭据或令牌，不涉及用户敏感信息。
- `exec` 子命令不自动注入代理凭据到环境变量，避免泄漏风险。
- 二进制发布前建议进行完整性校验（SHA256）。

### 5.4 更新机制

初期建议手动下载更新（GitHub Releases），后续可考虑：

- 内置 `self-update` 子命令，自动检测并下载最新版本
- 或集成到包管理器（Homebrew、Scoop、APT）

---

## 6. 总结与后续扩展

### 6.1 核心价值总结

`flutter-gradle-tool` 通过 CLI 工具的形式，将 Flutter 项目中 Gradle 构建的网络加速操作从"手动分散配置"升级为"一键统一管理"，解决了以下关键问题：

| 问题              | 方案效果                                           |
| ----------------- | -------------------------------------------------- |
| Gradle 分发下载慢 | 自动替换为国内镜像，下载速度从 KB/s 提升到 MB/s 级 |
| 镜像切换繁琐      | 交互式 / 命令式一键切换，无需记忆 URL              |
| 团队配置不一致    | 可通过 `.fgt-config` + CI 配置确保一致性           |
| 跨平台差异        | 单二进制 + 平台自适应，行为统一                    |

### 6.2 与现有方案对比

| 对比维度     | 手动修改 | Shell 脚本   | Gradle init.d | fgt             |
| ------------ | -------- | ------------ | ------------- | --------------- |
| 学习成本     | 低       | 中           | 高            | 低              |
| 跨平台       | ✅       | ❌（需兼容） | ✅            | ✅              |
| 镜像切换速度 | 慢       | 中           | 中            | 快              |
| CI 集成难易  | 繁琐     | 中           | 繁琐          | 易              |
| 可扩展性     | 无       | 可定制       | 可定制        | 内置 + 配置文件 |

### 6.3 可能的扩展方向

- **用户自定义镜像源**：支持从 `~/.fgt/sources.json` 加载自定义源，允许添加企业内部镜像。
- **自动选优**：在 `init` 或 `mirror set` 时增加 `--auto` 标志，自动测速所有源并选择最快的一个。
- **细粒度 Maven 镜像**：支持单独设置 Google、JCenter、Central 的不同镜像地址，而非统一替换。
- **插件系统**：允许通过外部插件扩展其他 Gradle 加速策略（如缓存共享、远程构建缓存）。
- **Docker 镜像**：提供预装工具的 Docker 镜像，适用于基于容器的 CI 环境。
