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

CLI for speeding up Flutter Android builds by managing Gradle wrapper and Maven mirror settings.

## Planned Commands

```bash
fgt mirror list
fgt mirror set --source aliyun
fgt mirror current
fgt mirror test
fgt init --source aliyun
fgt doctor
fgt cache
fgt exec build
```

## Built-in Mirrors

- `official`
- `tencent`
- `aliyun`
- `huaweicloud`

## Status

Implementation has started. The current plan and status live in:

- [docs/report.md](docs/report.md)
- [docs/ROADMAP.md](docs/ROADMAP.md)
- [docs/TODO.md](docs/TODO.md)
- [docs/tech-decision.md](docs/tech-decision.md)
