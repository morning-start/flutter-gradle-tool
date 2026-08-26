---
title: flutter-gradle-tool
doc_type: readme
version: 0.3.0
status: draft
updated: 2026-08-26
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

Solves two core problems for Flutter developers in China (or any slow-network environment):

1. **Gradle distribution download is slow** — `distributionUrl` defaults to `services.gradle.org` (overseas), causing multi-minute waits on first build.
2. **Maven dependency download is slow** — `google()`, `mavenCentral()` etc. are overseas repositories.

`fgt` replaces these with domestic mirrors (Aliyun, Tencent Cloud, Huawei Cloud) or a local mise-managed distribution — **zero network download**.

## Installation

### From source

```bash
go install github.com/morning-start/flutter-gradle-tool/cmd/fgt@latest
```

### Binary release

Download the latest binary from [GitHub Releases](https://github.com/morning-start/flutter-gradle-tool/releases).

Available for: `linux` / `darwin` / `windows` × `amd64` / `arm64`.

### Via Scoop (Windows)

```powershell
# After adding the bucket:
scoop install fgt
```

## Quick Start

### Option A: Use mise-managed local Gradle (recommended, zero network)

```bash
# 1. Install mise (https://mise.jdx.dev)
curl https://mise.run | sh

# 2. Install Gradle via mise
mise use gradle@8.5

# 3. Configure Flutter project to use local distribution
cd your-flutter-project
fgt init --mise

# 4. Build — no network download needed
fgt exec build
```

### Option B: Use domestic mirrors

```bash
# Initialize with Aliyun mirrors (both wrapper + Maven)
cd your-flutter-project
fgt init --source aliyun

# Or interactive selection
fgt init --interactive

# Build
fgt exec build
```

### Option C: CI/CD integration

```yaml
# GitHub Actions example
- name: Setup Gradle mirrors
  run: |
    # Download fgt binary
    curl -fsSL https://github.com/morning-start/flutter-gradle-tool/releases/latest/download/fgt_linux_amd64 -o fgt
    chmod +x fgt
    # Configure mirrors (non-interactive)
    ./fgt init --source tencent --ci
    # Build
    ./fgt exec assembleRelease
```

```groovy
// Jenkinsfile example
stage('Build') {
    steps {
        sh '''
            curl -fsSL https://github.com/morning-start/flutter-gradle-tool/releases/latest/download/fgt_linux_amd64 -o fgt
            chmod +x fgt
            ./fgt init --source aliyun --ci
            ./fgt exec assembleRelease
        '''
    }
}
```

## Commands

| Command | Description |
|---------|-------------|
| `fgt mirror list` | List built-in mirror sources |
| `fgt mirror set` | Set active mirror source |
| `fgt mirror current` | Show current mirror source |
| `fgt mirror test` | Test mirror latency/availability |
| `fgt init` | Initialize Gradle mirror settings for a Flutter project |
| `fgt init --mise` | Use mise-managed local Gradle distribution (zero network) |
| `fgt doctor` | Diagnose the current Flutter Gradle setup |
| `fgt cache` | Inspect Gradle cache |
| `fgt cache clean --all` | Clean Gradle cache directories |
| `fgt exec <tasks...>` | Run Gradle tasks through the project wrapper |

### `fgt init` flags

| Flag | Short | Description |
|------|-------|-------------|
| `--source <name>` | `-s` | Mirror source name (official, tencent, aliyun, huaweicloud) |
| `--mise` | | Use mise-managed local Gradle distribution |
| `--interactive` | `-i` | Interactive mirror selection |
| `--ci` | | Non-interactive mode (requires `--source`) |
| `--wrapper-only` | `-w` | Only modify gradle-wrapper.properties |
| `--maven-only` | `-m` | Only modify build.gradle Maven mirrors |
| `--project-dir <path>` | | Flutter project android directory (default: `./android`) |

### `fgt mirror set` flags

| Flag | Short | Description |
|------|-------|-------------|
| `--source <name>` | `-s` | Mirror source name |
| `--interactive` | `-i` | Interactive selection |
| `--ci` | | Non-interactive mode |
| `--wrapper-only` | `-w` | Only change wrapper mirror |
| `--maven-only` | `-m` | Only change Maven mirror |

## Built-in Mirrors

| Name | Gradle URL | Maven URL |
|------|-----------|-----------|
| `official` | `services.gradle.org/distributions` | *(default)* |
| `tencent` | `mirrors.cloud.tencent.com/gradle` | `mirrors.cloud.tencent.com/nexus/repository/maven-public/` |
| `aliyun` | `mirrors.aliyun.com/maven/gradle` | `maven.aliyun.com/repository/public` |
| `huaweicloud` | `mirrors.huaweicloud.com/gradle` | `mirrors.huaweicloud.com/repository/maven/` |

## How It Works

### Mirror mode (`fgt init --source <mirror>`)

1. Reads `gradle-wrapper.properties`, extracts version and distribution type (`-all` / `-bin`)
2. Replaces `distributionUrl` with the selected mirror's URL
3. Injects Maven mirror repository into `build.gradle` / `build.gradle.kts` (with `// Added by fgt` marker)
4. Saves selection to `.fgt-config` (auto-added to `.gitignore`)

### Mise mode (`fgt init --mise`)

1. Detects if `mise` is installed and manages a Gradle version
2. Locates the local distribution zip file
3. Replaces `distributionUrl` with a `file://` URL pointing to the local zip
4. **Zero network download** — Gradle Wrapper uses the local file directly

### Idempotency

All operations are idempotent — running `fgt init` multiple times produces the same result. If the target mirror is already configured, no files are modified.

## Project Structure

```
flutter-gradle-tool/
├── cmd/fgt/                  # CLI entry point
│   ├── main.go               # Program entry
│   ├── root.go               # Root command & subcommand registration
│   ├── init.go               # `init` command (mirror + mise)
│   ├── mirror.go             # `mirror` command group
│   ├── cache.go              # `cache` command
│   ├── doctor.go             # `doctor` command
│   ├── exec.go               # `exec` command
│   └── interactive.go        # Interactive selection UI
├── internal/
│   ├── mirror/               # Mirror source data & config persistence
│   ├── gradle/               # Wrapper/Maven file manipulation
│   ├── mise/                 # mise integration
│   ├── doctor/               # Diagnostic logic
│   ├── cache/                # Cache inspection & cleanup
│   └── errors/               # Error types & exit codes
├── docs/                     # Documentation
├── .github/workflows/        # CI/CD
├── .goreleaser.yaml          # Release configuration
└── go.mod
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GRADLE_USER_HOME` | Gradle cache directory | `~/.gradle` |
| `FGT_TEST_CONCURRENCY` | Mirror test concurrency | `4` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Project directory not found |
| 2 | gradle-wrapper.properties parse error |
| 3 | Network unreachable (mirror test) |
| 4 | Unknown mirror source name |
| 5 | `--ci` mode requires `--source` |
| 6 | File write permission error |
| 7 | Unknown command |

## Development

### Build

```bash
# Windows
build.bat

# Linux / macOS
go build -o fgt ./cmd/fgt/...
```

### Test

```bash
go test ./...
```

### Release

Releases are automated via GitHub Actions + GoReleaser. Push a `v*` tag to trigger:

```bash
git tag v0.3.0
git push origin v0.3.0
```

## Documentation

- [docs/report.md](docs/report.md) — Product requirements and design
- [docs/ROADMAP.md](docs/ROADMAP.md) — Roadmap and milestones
- [docs/TODO.md](docs/TODO.md) — Current tasks
- [docs/tech-decision.md](docs/tech-decision.md) — Technical decisions

## License

[MIT](LICENSE)
