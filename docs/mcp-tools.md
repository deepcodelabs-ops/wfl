# MCP Tools Reference

When `wfl serve <file.wfl>` is running, the following tools are available to any MCP-compatible agent.

All tools re-parse and re-validate the `.wfl` file on every call. There is no caching — the file is the authoritative source of truth at all times. This means edits to the file are always reflected without restarting the server.

---

## Registering the server

### Claude Code (`~/.claude/mcp.json`)

```json
{
  "mcpServers": {
    "wfl": {
      "command": "/path/to/wfl",
      "args": ["serve", "/path/to/your/feature.wfl"]
    }
  }
}
```

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "wfl": {
      "command": "/path/to/wfl",
      "args": ["serve", "/absolute/path/to/feature.wfl"]
    }
  }
}
```

The `serve` command uses **stdio transport** — the MCP client launches `wfl` as a subprocess and communicates over stdin/stdout. The path to the `.wfl` file must be **absolute**.

---

## Tool: `workflow.listFlows`

Lists all flows in a `.wfl` file with their version and drift status.

**Use this first** when connecting to a workflow file — it gives you the map of what exists before reading individual flows.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output

```json
{
  "workflowFile": "/path/to/feature.wfl",
  "flows": [
    {
      "FlowName": "UserAuth",
      "IntentVersion": "1.0.0",
      "ImplVersion": "1.0.0",
      "HashVerified": true,
      "IsStale": false,
      "Drift": "in-sync"
    },
    {
      "FlowName": "TokenRefresh",
      "IntentVersion": "1.0.0",
      "ImplVersion": "",
      "HashVerified": true,
      "IsStale": false,
      "Drift": "impl-missing"
    }
  ]
}
```

### Drift values

| Value | Meaning |
|-------|---------|
| `in-sync` | Intent and impl are on compatible versions |
| `impl-missing` | No `@@impl` block exists yet |
| `stale` | Intent has a major version bump relative to impl |
| `hash-mismatch` | Intent block has been tampered — **hard error** |

---

## Tool: `workflow.getIntent`

Returns the raw content of the `@@intent` block for a named flow.

**Hard error** if the hash does not verify — the tool returns an error result and will not return the content of a tampered intent block.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name (e.g., `UserAuth`) |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output (success)

```json
{
  "flowName": "UserAuth",
  "version": "1.0.0",
  "hashVerified": true,
  "content": "  # feat: initial auth flow\n\n  client -> gateway: POST /login\n  ..."
}
```

### Output (error — hash mismatch)

```json
{
  "isError": true,
  "content": "hash verification failed:\n  @@intent block for flow \"UserAuth\" has been tampered — hash mismatch"
}
```

---

## Tool: `workflow.getImpl`

Returns the raw content of the `@@impl` block for a named flow.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output (impl exists)

```json
{
  "flowName": "UserAuth",
  "version": "1.0.0",
  "content": "  # Implemented 2026-10-06\n  # src/auth/validator.go:42 ..."
}
```

### Output (impl missing)

```json
{
  "flowName": "UserAuth",
  "implMissing": true
}
```

---

## Tool: `workflow.getContext`

Resolves all `@context` annotations in the intent block to actual file excerpts. Returns `@api` and `@db` annotations as pass-through metadata.

This is the key tool for giving an implementing agent real codebase context without reading whole files. Each `@context` pointer returns ±20 lines around the referenced line.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output

```json
{
  "flowName": "UserAuth",
  "contexts": [
    {
      "annotation": "src/auth/validator.go:42",
      "filePath": "/absolute/path/src/auth/validator.go",
      "lineNumber": 42,
      "excerpt": [
        "// ValidateCredentials checks username + bcrypt hash",
        "func ValidateCredentials(db *sql.DB, email, password string) (User, error) {",
        "    var user User",
        "    ..."
      ],
      "exists": true
    }
  ],
  "metadata": {
    "api": ["internal/auth"],
    "db": ["users.credentials"]
  }
}
```

If a referenced file does not exist, `exists` is `false` and `excerpt` is empty — the tool does not error, allowing the agent to proceed and note the missing file.

---

## Tool: `workflow.getSubtasks`

Decomposes the intent block's flow arrows into an ordered list of subtasks. Each subtask maps to one arrow.

Use this when an orchestrating agent wants to dispatch work to subagents. Subtasks with the same `Order` value can be run in parallel.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output

```json
{
  "flowName": "UserAuth",
  "subtasks": [
    {
      "ID": "UserAuth-1",
      "Description": "client -> gateway: POST /login",
      "From": "client",
      "To": "gateway",
      "Label": "POST /login",
      "Order": 1
    },
    {
      "ID": "UserAuth-2",
      "Description": "gateway -> authService: validateCredentials()",
      "From": "gateway",
      "To": "authService",
      "Label": "validateCredentials()",
      "Order": 2
    }
  ]
}
```

---

## Tool: `workflow.getDependencies`

Returns other flows that this flow depends on. Dependencies are declared with `# depends: <FlowName>` comments in the intent block.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output

```json
{
  "flowName": "TokenRefresh",
  "dependencies": ["UserAuth", "TokenStore"]
}
```

An empty `dependencies` array means the flow has no declared dependencies and can be implemented independently.

---

## Tool: `workflow.getStatus`

Returns the full drift status for a single named flow.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |

### Output

```json
{
  "FlowName": "UserAuth",
  "IntentVersion": "1.0.0",
  "ImplVersion": "1.0.0",
  "HashVerified": true,
  "IsStale": false,
  "Drift": "in-sync"
}
```

---

## Tool: `workflow.writeImpl`

Writes content to the `@@impl` block for a named flow.

This is the only mutating tool. All writes are atomic (write to `.wfl.tmp`, then rename).

### Safety guards (enforced server-side)

1. **Cannot write to intent blocks** — if `flow_name` matches an `@@intent` block, the tool returns `WRITE_REFUSED`
2. **Cannot write when stale** — if the intent block has a major version bump relative to impl, the tool returns `STALE_INTENT`. The developer must first bump the `@@impl` version header to acknowledge the breaking change.

### Input

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `flow_name` | string | Yes | Flow name |
| `workflow_file_path` | string | Yes | Absolute path to the `.wfl` file |
| `content` | string | Yes | Content to write into the `@@impl` block |

### Output (success)

```json
{
  "success": true,
  "flowName": "UserAuth",
  "message": "@@impl block updated successfully"
}
```

### Output (stale error)

```json
{
  "isError": true,
  "content": "STALE_INTENT: the intent block has a breaking major version bump. Developer must bump the @@impl version to acknowledge before agent can proceed."
}
```

### Output (write refused)

```json
{
  "isError": true,
  "content": "WRITE_REFUSED: agents may not write to @@intent blocks"
}
```

---

## Error handling

All tools return errors as MCP tool-level errors (not protocol errors). The response will have `isError: true` and a descriptive message in `content`. Your agent should check for `isError` before using the result.

Hard errors (hash mismatch, parse failure) are surfaced clearly. Soft states (impl missing, file not found for `@context`) are returned as structured data with boolean flags so the agent can decide how to proceed.

---

## Recommended agent invocation sequence

```
1. workflow.listFlows(workflow_file_path)
   → Discover all flows and their status

2. For each flow that needs implementation:
   a. workflow.getDependencies(flow_name, ...)
      → Build dependency graph; determine order
   b. workflow.getIntent(flow_name, ...)
      → Read the developer's specification
   c. workflow.getContext(flow_name, ...)
      → Get actual code excerpts for @context pointers
   d. workflow.getSubtasks(flow_name, ...)
      → Decompose into subtasks; dispatch parallel subagents if applicable

3. After implementation:
   workflow.writeImpl(flow_name, ..., content)
   → Record what was built, any deviations, test locations

4. Verify:
   workflow.getStatus(flow_name, ...)
   → Confirm Drift == "in-sync" before reporting done
```
