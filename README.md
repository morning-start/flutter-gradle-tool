---
title: flutter-gradle-tool
doc_type: readme
version: 0.2.0
status: draft
updated: 2026-06-10
related_docs:
  - docs/report.md
  - docs/ROADMAP.md
  - docs/TODO.md
  - docs/tech-decision.md
---

# flutter-gradle-tool

[![CI](https://github.com/morning-start/flutter-gradle-tool/actions/workflows/ci.yml/badge.svg)](https://github.com/morning-start/flutter-gradle-tool/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/morning-start/flutter-gradle-tool)](go.mod)

CLI for speeding up Flutter Android builds by managing Gradle wrapper and Maven mirror settings.

## Installation

### From source

```bash
go install github.com/morning-start/flutter-gradle-tool/cmd/fgt@latest
```

### Binary release

Download the latest binary from [GitHub Releases](https://github.com/morning-start/flutter-gradle-tool/releases).

## Quick Start

```bash
# List available mirror sources
fgt mirror list

# Initialize a Flutter project with Aliyun mirrors
fgt init --source aliyun

# Interactive selection
fgt init --interactive

# Test mirror availability
fgt mirror test

# Check current Gradle health
fgt doctor

# Run Gradle build through wrapper (auto-detects gradlew / gradlew.bat)
fgt exec build

# Clean Gradle cache
fgt cache clean --all
```

## Commands

| Command | Description |
|---------|-------------|
| `fgt mirror list` | List built-in mirror sources |
| `fgt mirror set` | Set active mirror source |
| `fgt mirror current` | Show current mirror source |
| `fgt mirror test` | Test mirror availability |
| `fgt init` | Initialize Gradle mirror settings for a Flutter project |
| `fgt doctor` | Diagnose the current Flutter Gradle setup |
| `fgt cache [inspect\|clean]` | Inspect or clean Gradle cache |
| `fgt exec <tasks...>` | Run Gradle tasks through the project wrapper |

## Built-in Mirrors

- `official` — Official (services.gradle.org)
- `tencent` — Tencent Cloud
- `aliyun` — Aliyun
- `huaweicloud` — Huawei Cloud

## Development

- [docs/report.md](docs/report.md) — Product requirements and design
- [docs/ROADMAP.md](docs/ROADMAP.md) — Roadmap and milestones
- [docs/TODO.md](docs/TODO.md) — Current tasks
- [docs/tech-decision.md](docs/tech-decision.md) — Technical decisions

Build locally:

```bash
# Windows
build.bat

# Linux / macOS
go build -o fgt ./cmd/fgt/...
```

## License

[MIT](LICENSE)
