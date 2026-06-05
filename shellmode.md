# Bash Mode Design

## Architecture

Bash mode (`!command` / `$ command`) is a lightweight way to execute shell commands directly from the chat bar without going through the LLM agent.

### Execution flow

1. User types `!ls` → `runBashCommand("ls")`
2. Pending bash tool card rendered in chat (spinner)
3. `shell.Run()` executes via mvdan/sh in a goroutine
4. On completion: pending card removed, command persisted to DB via `CreateMessage`
5. PubSub event triggers UI re-render with the completed bash tool card

### Persistence model

Each bash command is persisted as a **single User message** with three parts:

```
User
├── TextContent: "$ ls"               ← command text (for display/reload)
├── ToolCall: {Name: "bash", ...}      ← triggers bash tool card UI rendering
└── ToolResult: {Content: "file1\n"}   ← command output
```

## Design Quirks

### 1. User message as a container for ToolCall + ToolResult

**Problem**: `message.Service` only supports creating messages by role (User, Assistant, Tool, System). There's no way to persist a standalone ToolResult. The ToolCall and ToolResult must be attached to a parent message.

**Solution**: Pack all three (TextContent + ToolCall + ToolResult) into a single User message.

**Consequence**: `ToAIMessage` must unpack this into `User → Assistant(ToolCall) → Tool(ToolResult)` for the LLM, because Anthropic etc. require Tool messages to reference a preceding Assistant ToolCall.

### 2. Synthetic Assistant message for provider compatibility

**Problem**: Anthropic, OpenAI, and other providers validate that every `ToolResult` has a matching `ToolCall` from an `Assistant` message. Since bash mode's ToolCall is user-initiated (not LLM-generated), there is no real Assistant message.

**Solution**: `ToAIMessage()` emits a synthetic `Assistant: ToolCall{ID: "shell_xxx"}` as a compatibility shim. It has no text, no LLM involvement — just satisfies the provider protocol.

**Tradeoff**: The Assistant message is semantically wrong (the LLM didn't make this tool call), but functionally harmless.

### Future: Split into User + Tool Messages

An alternative approach would persist the bash command as **two separate messages**:

```
User: TextContent("$ ls")
Tool: ToolResult{ToolCallID: "shell_xxx", Content: "file1\n"}
```

This would eliminate the need for the synthetic Assistant shim in `ToAIMessage()` — the Tool message could carry the result directly, which the standard user → tool conversation already supports.

**Cost**:
- `message.Service` needs a standalone Tool message creation path (currently all bash commands go through `CreateMessage` with Role=User)
- `ShouldRenderUserMessage` would need a new mechanism (metadata flag or Attachment) to distinguish "bash command user bubble" from "normal user input", since the User message would no longer carry ToolCalls as a rendering signal
- UI rendering would need to correlate the User and Tool messages (by shared tool call ID) to render the bash card in the correct position
- More moving parts for what is ultimately an edge case feature

### 3. Output truncation

**Problem**: Shell command output is persisted verbatim. `cat hugefile` or `curl` large responses can bloat the session DB.

**No current fix**: A `MaxBytes` limit on persisted output would prevent DB bloat. This is a future improvement.

### 4. No `BlockFuncs` in `runBashExecution`

**Problem**: `shell.Run()` in bash mode passes `BlockFuncs: nil`, unlike the agent's bash tool which always passes block funcs. Dangerous commands (`rm -rf /`, `curl | sh`) are not intercepted.

**Rationale**: Bash mode is a user-initiated action in the TUI, not an LLM tool call. The user is executing commands directly, so the same protections may not apply. This is a deliberate design choice, but worth noting.
