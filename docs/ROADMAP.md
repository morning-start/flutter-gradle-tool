---
title: flutter-gradle-tool 路线图
doc_type: roadmap
version: 0.2.0
status: draft
updated: 2026-06-10
related_docs:
  - report.md
  - TODO.md
  - tech-decision.md
---

# 路线图：flutter-gradle-tool

## 文档关系

- [report.md](report.md)：产品方案、范围和实现约束。
- [TODO.md](TODO.md)：当前阶段的执行清单。
- [tech-decision.md](tech-decision.md)：库选型与自研边界。
- 本文档：阶段目标、里程碑、风险与评审节奏。

> 基于 Go 的 Flutter Gradle 加速 CLI 工具

---

## 战略愿景（1 年方向）

让 Flutter 开发者通过一条命令完成 Gradle 构建加速，消除国内网络环境下的 Gradle 下载瓶颈，成为 Flutter 生态中 Gradle 配置管理的标准辅助工具。

---

## 阶段总览

| 阶段 | 名称 | 时间 | 交付物 | 状态 |
|------|------|------|--------|------|
| P1 | 项目骨架与镜像管理核心 | 第 1-3 周 | CLI 框架 + 镜像源 CRUD | **进行中** |
| P2 | Gradle 配置操作 | 第 4-6 周 | init 命令 + wrapper/Maven 修改 | 进行中 |
| P3 | 辅助命令与平台适配 | 第 7-8 周 | doctor + cache + exec + Windows 兼容 | 待开始 |
| P4 | 高级功能与发布 | 第 9-11 周 | 测速 + 文档 + 二进制发布 | 待开始 |

---

## 阶段详情

### P1：项目骨架与镜像管理核心（第 1-3 周）

**目标**：搭建完整的 Go CLI 项目骨架，实现镜像源的列出、切换和查看功能，确保持久化和幂等性。

| 关键结果 | 衡量标准 |
|----------|----------|
| KR1: CLI 框架集成 | cobra 子命令路由正常，`fgt --help` 输出完整帮助信息 |
| KR2: 镜像源 CRUD | 内置 ≥4 个镜像源，`list`/`set`/`current` 三个子命令可用 |
| KR3: 持久化机制 | `.fgt-config` 文件读写正确，`current` 可反向推断 |
| KR4: 错误处理 | ≥8 种错误场景有明确退出码和提示信息 |
| KR5: 单元测试 | 核心函数覆盖率 ≥80% |

**交付物**：
- [x] Go 项目初始化（go.mod、目录结构）
- [ ] MirrorSource 数据结构与内置列表
- [ ] `mirror list` / `mirror set` / `mirror current` 子命令
- [ ] `.fgt-config` 持久化读写
- [ ] 退出码定义与错误提示
- [ ] 单元测试套件

---

### P2：Gradle 配置操作（第 4-6 周）

**目标**：实现 Gradle wrapper 分发 URL 的解析与替换，以及 Maven 仓库镜像的注入，完成 `init` 命令。

| 关键结果 | 衡量标准 |
|----------|----------|
| KR1: distributionUrl 解析 | 正确提取版本号和分发类型，覆盖 `-all`/`-bin`/`-src` |
| KR2: 镜像 URL 构建 | 基于任意内置源的 GradleURL 正确构建新 URL |
| KR3: Maven 镜像注入 | `build.gradle` 中正确插入/移除镜像仓库配置 |
| KR4: 幂等性 | 重复执行 `init` 不产生重复插入，已有镜像不重复添加 |
| KR5: Groovy DSL 支持 | 标准 Flutter 项目的 `build.gradle` 正确修改 |

**交付物**：
- [x] `internal/gradle/wrapper.go` — URL 解析与重构
- [x] `internal/gradle/maven.go` — Maven 镜像注入
- [x] `init` 子命令完整实现
- [ ] `build.gradle.kts` 支持规划（延后到 P4）

---

### P3：辅助命令与平台适配（第 7-8 周）

**目标**：补充 `doctor`、`cache`、`exec` 命令，完成 Windows 平台适配。

| 关键结果 | 衡量标准 |
|----------|----------|
| KR1: doctor 诊断 | 诊断 wrapper URL / Maven 镜像 / 持久化一致性 |
| KR2: cache 管理 | 查看 Gradle 缓存路径、大小；清理过期/全部缓存 |
| KR3: exec 执行 | 跨平台调用 wrapper 脚本，Windows 检测 `gradlew.bat` |
| KR4: Windows 兼容 | 路径/脚本/换行符/编码全部通过 |
| KR5: 跨平台测试 | Windows + macOS + Linux 三平台通过测试 |

**交付物**：
- [ ] `internal/cache/cache.go`
- [ ] `internal/doctor/doctor.go`
- [ ] `internal/gradle/build.go`（exec 逻辑）
- [ ] Windows 兼容性适配
- [ ] 跨平台 CI 测试

---

### P4：高级功能与发布（第 9-11 周）

**目标**：实现镜像源测速、交互式选择优化，完成文档和二进制发布。

| 关键结果 | 衡量标准 |
|----------|----------|
| KR1: 镜像测速 | `mirror test` 并发测速，超时 5s，排序输出 |
| KR2: 交互式选择 | 简单菜单展示镜像列表，后续可升级为 promptui |
| KR3: CI 集成文档 | 提供 GitHub Actions、Jenkins 示例 |
| KR4: 二进制发布 | GitHub Releases 发布 Windows/macOS/Linux × amd64/arm64 |

**交付物**：
- [ ] `mirror test` 子命令
- [ ] 交互式选择 UI 优化（可选引入 promptui）
- [ ] GitHub Actions CI/CD 配置
- [ ] GitHub Releases 发布脚本
- [ ] 用户文档（README、使用示例）

---

## 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 镜像 URL 变更或失效 | 中 | 高 | 内置源可配置+可扩展，定期验证脚本 |
| `build.gradle` 格式多样难以通用解析 | 高 | 中 | 初期仅支持标准 Flutter 模板，使用标记注释而非完整 AST |
| Windows 路径/编码兼容问题 | 中 | 中 | P3 阶段专项测试，CI 中加入 Windows runner |
| 个人开发者时间不足 | 中 | 高 | 优先 MVP（P1+P2），P3/P4 按需裁减 |
| Gradle wrapper 格式变化 | 低 | 高 | 预留 URL 解析接口，版本变化只需更新正则 |

---

## 评审节奏

- **周评审**：每周日检查本周 TODO 完成情况，调整下周计划
- **阶段评审**：每阶段结束时回顾 OKR 达成率
- **整体复盘**：P4 完成后进行全项目复盘
