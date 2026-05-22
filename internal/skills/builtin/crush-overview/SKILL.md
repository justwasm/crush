---
name: crush-overview
description: Use when the user wants a guided walkthrough of Crush's major product features, extension points, and key parts of the codebase.
user-invocable: true
---

# Crush Overview

Use this skill when the user is new to Crush and wants a concise tour of what
the project does, how it is organized, and which capabilities matter most.

## What Crush Is

Crush is a terminal-based AI coding assistant written in Go. It combines LLM
providers, local tools, code intelligence, and user-defined extensions into a
single interactive CLI and TUI.

## Major Features To Cover

Walk through the project in this order unless the user asks for a different
path:

1. **Model and provider support**
   - Crush works with multiple providers and model families.
   - Users can switch models during a session without losing context.
   - Providers can be built in or configured through `crush.json`.

2. **Session-based workflow**
   - Crush keeps work organized around sessions and project context.
   - Sessions are persisted, so users can return to prior work.
   - Multiple sessions can coexist for different tasks.

3. **Tool-using agent**
   - The agent can inspect files, edit code, run shell commands, search the
     repository, and call external MCP tools.
   - Permissions control which tools can run automatically.
   - Hooks can intercept tool calls before execution.

4. **Code intelligence and context**
   - Crush can use LSP servers for richer code understanding.
   - It reads project context files such as `AGENTS.md`, `CRUSH.md`,
     `CLAUDE.md`, and `GEMINI.md`.
   - It tracks touched files, prompt history, and other session context.

5. **Extensibility**
   - Skills add reusable instructions for common tasks.
   - MCP servers add external tools over `stdio`, `http`, or `sse`.
   - Hooks let users gate, rewrite, or annotate tool calls.

6. **Terminal-first user experience**
   - Crush has both CLI commands and a Bubble Tea TUI.
   - It supports model management, sessions, stats, login flows, and project
     initialization from the terminal.

7. **Configuration and customization**
   - `crush.json` controls models, providers, MCP, LSP, hooks, permissions,
     tools, and options.
   - Local config can override global config.
   - Users can disable tools or skills they do not want the agent to use.

## Codebase Map

When the user wants to connect features to implementation, point them here:

- `main.go` and `internal/cmd/`: CLI entry points and commands.
- `internal/app/`: top-level wiring for config, DB, agents, LSP, MCP, and
  events.
- `internal/agent/`: conversation loop, prompt assembly, and tool execution.
- `internal/agent/tools/`: builtin tool implementations and MCP integration.
- `internal/config/`: config loading, validation, and provider setup.
- `internal/hooks/`: hook execution and aggregation.
- `internal/lsp/`: language server management.
- `internal/skills/`: skill discovery, builtin skill embedding, and tracking.
- `internal/session/` and `internal/db/`: session persistence backed by SQLite.
- `internal/ui/`: terminal interface.

## How To Present The Walkthrough

- Start with a high-level summary before diving into details.
- Keep the explanation grounded in user-visible features first, then map those
  features to the codebase.
- Call out extension points explicitly: skills, hooks, MCP, providers, and LSP.
- End by asking which area the user wants to explore next.
