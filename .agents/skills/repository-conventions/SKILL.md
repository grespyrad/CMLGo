---
name: repository-conventions
description: Maintain portable repository instructions, skills and commit hooks when changing agent guidance or project workflows.
---

Use `AGENTS.md` for repository rules and `.agents/skills/<name>/SKILL.md` for
task-specific workflows. Keep these instructions usable by different agents.
Provider/editor instruction files and aliases are forbidden: Cursor MDC,
CLAUDE.md, GEMINI.md, proprietary rule directories and Copilot instructions.
Keep necessary guidance when migrating; avoid duplicate sources of rules.

Small shell wrappers and simple commands are allowed. Put substantial logic,
branches, retries, state handling and deployment orchestration in a program.
Do not convert existing tools just to change language.

Run `go run .agents/skills/repository-conventions/scripts/check.go` before commit.
The hook checks the Git index and non-ignored untracked paths, so deletions must
be staged. `-self-test` verifies rejection and allowed portable paths.
