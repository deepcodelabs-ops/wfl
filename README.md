# wfl — Workflow File System

**wfl** is a developer-intent layer for agent-based development. You write a `.wfl` file describing what a feature should do. Agents read it in focused chunks via an MCP server, implement the code, and write notes back — all without modifying your original intent.

---

## Why wfl

When agents write code, the developer's job shifts from writing code to communicating intent. The challenge is that agents need sufficient context to act correctly, but dumping an entire codebase into an agent's context is wasteful and imprecise.

wfl solves this with a structured file format where:
- **You write intent** — reviewable by teammates, renderable as Mermaid diagrams
- **Agents query it in chunks** — via MCP tools, getting only what's needed for the current task
- **Agents write back** — post-implementation notes in `@@impl` blocks
- **Nothing drifts silently** — intent blocks are hash-locked; breaking changes auto-flag stale implementations

---

## Quick Start

### 1. Install

```bash
go install github.com/rishirajsingh/wfl/cmd/wfl@latest
```

Or build from source:
```bash
git clone https://github.com/rishirajsingh/wfl
cd wfl
go build -o wfl ./cmd/wfl
```

### 2. Create a workflow file

```bash
wfl init user_auth.wfl
```

This scaffolds a `.wfl` file with a hash-locked intent block:

```
@@intent v1.0.0 flow:UserAuth
  # feat: initial UserAuth flow

  actor -> service: action()
@@end:sha256:<computed-hash>

@@impl v1.0.0 flow:UserAuth
  # agent-written after implementation
@@end

@@changelog flow:UserAuth
  v1.0.0 feat: initial UserAuth flow
@@end
```

### 3. Write your intent

Edit the `@@intent` block to describe your feature. After editing, recompute the hash:

```bash
wfl validate user_auth.wfl
```

If the hash is stale (because you edited the block), regenerate it:

```bash
wfl init --rehash user_auth.wfl   # coming in v2; for now: delete and re-init
```

> **For now:** when you edit an `@@intent` block, delete the old `@@end:sha256:...` line,
> parse the raw content manually and compute `sha256sum`, or use `wfl init` on a fresh file.
> A `wfl rehash` command is planned for v2.

A real workflow file looks like this:

```
@@intent v1.0.0 flow:UserAuth
  # feat: JWT-based login flow
  # depends: TokenStore

  client -> gateway: POST /login { email, password }
  gateway -> authService: validateCredentials()
    @context: src/auth/validator.go:42
    @api: internal/auth
    @db: users.credentials
  authService -> tokenStore: issueJWT(userId)
    @context: src/token/issuer.go:18
  gateway -> client: 200 { token, expiresAt }
@@end:sha256:<hash>

@@impl v1.0.0 flow:UserAuth
  # Implemented by agent on 2026-10-06
  # jwt.go:42 — ValidateCredentials reads bcrypt hash from users table
  # issuer.go:18 — HS256 signing, 24h expiry
  # Tests: auth_test.go covers happy path + wrong password + expired token
@@end

@@changelog flow:UserAuth
  v1.0.0 feat: initial JWT auth flow
@@end
```

### 4. Preview as a diagram

```bash
wfl render user_auth.wfl
```

Produces `user_auth.md` with a Mermaid sequence diagram — paste it into GitHub, GitLab, or any Markdown viewer.

### 5. Check drift status

```bash
wfl status user_auth.wfl
```

```
FLOW      INTENT  IMPL    STATUS
UserAuth  v1.0.0  v1.0.0  in-sync
```

### 6. Register as an MCP server

Run this once from the root of any project to register `wfl` in Claude Code, Cursor, and VS Code/Copilot simultaneously:

```bash
bash /path/to/wfl/scripts/register-mcp.sh
```

The script:
- Finds the `.wfl` file(s) in your current project directory
- Writes `mcpServers.wfl` into `.claude/settings.json`, `.cursor/mcp.json`, and `.vscode/mcp.json`
- Merges safely with any existing config (idempotent — safe to re-run)
- Creates the config directories automatically if they do not exist

After running, reload the editor window to pick up the new server. Agents connected via MCP can then call `workflow.getIntent`, `workflow.getContext`, `workflow.writeImpl`, and the other tools described below.

To register manually instead, add to the appropriate config file:

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

---

## The `.wfl` File Format

A `.wfl` file contains named blocks separated by `@@` markers.

### Block Types

| Block | Owner | Hash-locked | Purpose |
|-------|-------|-------------|---------|
| `@@intent` | Developer | Yes | Describes what the feature should do |
| `@@impl` | Agent | No | Agent writes implementation notes back here |
| `@@changelog` | Append-only | Past entries locked | Version history for the flow |

### Annotations inside `@@intent`

| Annotation | Meaning |
|------------|---------|
| `@context: path/to/file.go:42` | Points to a specific line of code; agent receives a ±20-line excerpt |
| `@api: internal/auth` | Names an API the flow calls (metadata, no file resolution) |
| `@db: users.credentials` | Names a database table/collection the flow touches |
| `# depends: OtherFlow` | Declares a dependency on another flow in the same file |

### Versioning

Versions follow semver. The bump kind carries meaning:

| Bump | Meaning | Agent effect |
|------|---------|--------------|
| `patch` (v1.0.0 → v1.0.1) | Clarification, no structural change | Agent can proceed |
| `minor` (v1.0.0 → v1.1.0) | New scope added, non-breaking | Agent should re-read intent |
| `major` (v1.0.0 → v2.0.0) | **BREAKING CHANGE** | Agent is blocked; `@@impl` marked stale until developer bumps its version to acknowledge |

### Hash lock

The `@@end:sha256:<hash>` line on `@@intent` blocks is a SHA256 of the block's raw content. If you or an agent modifies the intent block, the hash will no longer match and `wfl validate` will exit with an error. This prevents accidental or silent mutation of developer intent.

---

## CLI Reference

```
wfl init <file.wfl> [--flow-name <Name>]
    Scaffold a new workflow file with a computed hash.
    Creates the target directory automatically if it does not exist.
    --flow-name overrides the name derived from the filename.

wfl add-flow <file.wfl> <FlowName>
    Append a new flow skeleton (@@intent / @@impl / @@changelog) to an
    existing .wfl file. The new intent block is scaffolded with a
    correct hash and a visual # === FlowName === separator.

wfl rehash <file.wfl>
    Recompute and update all @@intent hash locks in-place.
    Run this after editing any @@intent block content — replaces the
    manual sha256sum + paste workflow entirely.

wfl check-context <file.wfl>
    Verify that every @context annotation path exists and that the
    referenced line number is within the file's length.
    Reports [OK], [MISSING], or [STALE] per annotation. Exits 1 on any failure.

wfl validate <file.wfl>   Verify all @@intent hash locks (exit 1 on mismatch)
wfl status <file.wfl>     Show intent/impl versions and drift status for all flows
wfl render <file.wfl>     Render all flows as Mermaid diagrams to <file>.md
wfl serve <file.wfl>      Start MCP server in stdio mode pointed at the workflow file
```

---

## MCP Tool Reference

When running as an MCP server (`wfl serve`), agents can call these tools:

| Tool | Input | Returns |
|------|-------|---------|
| `workflow.listFlows` | `workflow_file_path` | All flows with version and drift status |
| `workflow.getIntent` | `flow_name`, `workflow_file_path` | Intent block content, version, hash verification status |
| `workflow.getImpl` | `flow_name`, `workflow_file_path` | Impl block content, or `implMissing: true` |
| `workflow.getContext` | `flow_name`, `workflow_file_path` | Resolved file excerpts for all `@context` annotations; `@api`/`@db` metadata |
| `workflow.getSubtasks` | `flow_name`, `workflow_file_path` | Ordered subtask list decomposed from intent arrows |
| `workflow.getDependencies` | `flow_name`, `workflow_file_path` | Other flows this flow depends on |
| `workflow.getStatus` | `flow_name`, `workflow_file_path` | Drift status (in-sync / impl-missing / stale / hash-mismatch) |
| `workflow.writeImpl` | `flow_name`, `workflow_file_path`, `content` | Writes content to `@@impl` block; refuses if stale or if target is an intent block |

### Agent usage pattern

A typical agent session with wfl looks like this:

```
1. workflow.listFlows         → discover what flows exist
2. workflow.getIntent         → read the developer's intent
3. workflow.getContext        → resolve @context pointers to actual code
4. workflow.getSubtasks       → decompose into parallel workstreams
   └── dispatch subagents per subtask group
5. workflow.writeImpl         → record what was implemented
6. workflow.getStatus         → confirm in-sync before reporting done
```

---

## Multi-flow files

A single `.wfl` file can contain multiple flows:

```
@@intent v1.0.0 flow:UserAuth
  ...
@@end:sha256:...

@@impl v1.0.0 flow:UserAuth
@@end

@@changelog flow:UserAuth
  v1.0.0 feat: initial
@@end

@@intent v1.0.0 flow:TokenRefresh
  # depends: UserAuth
  ...
@@end:sha256:...

@@impl v1.0.0 flow:TokenRefresh
@@end

@@changelog flow:TokenRefresh
  v1.0.0 feat: initial
@@end
```

Use `workflow.listFlows` to discover all flows, `workflow.getDependencies` to understand the execution order, and dispatch subagents per flow in parallel where dependencies allow.

---

## Updating intent (breaking change workflow)

When you need to change how a feature works:

1. Edit the `@@intent` block content
2. Recompute the SHA256 hash of the new raw content
3. Update the `@@end:sha256:...` line
4. Bump the version — use `major` if the change breaks the existing implementation
5. Add an entry to `@@changelog`

```
@@intent v2.0.0 flow:UserAuth
  # feat: migrate from JWT to OAuth2
  # BREAKING: replaced direct credential flow with OAuth provider redirect
  ...
@@end:sha256:<new-hash>

@@changelog flow:UserAuth
  v2.0.0 feat: migrate to OAuth2 — BREAKING dropped direct credentials
  v1.0.0 feat: initial JWT auth flow
@@end
```

The `@@impl` block at `v1.0.0` is now automatically flagged as **stale**. Agents will be blocked from writing to it until you bump the `@@impl` version to acknowledge the breaking change:

```
@@impl v2.0.0 flow:UserAuth
  # ready for agent implementation
@@end
```

---

## Pre-commit hook (optional)

Add this to `.git/hooks/pre-commit` to prevent committing with broken hashes:

```bash
#!/bin/sh
for f in $(git diff --cached --name-only | grep '\.wfl$'); do
  wfl validate "$f" || exit 1
done
```

---

## Project structure

```
wfl/
  cmd/wfl/main.go            CLI entry point
  internal/
    parser/                  .wfl file parser (line scanner)
    validator/               Hash verification, stale detection
    renderer/                Mermaid diagram output
    context/                 @context annotation resolver
    mcp/                     MCP server and 8 tool definitions
    cli/                     CLI command implementations
  pkg/types/types.go         Shared types
  testdata/                  Sample .wfl files
  docs/                      Detailed documentation
```

---

## See also

- [docs/vision.md](docs/vision.md) — Why this tool exists and where it is going
- [docs/file-format.md](docs/file-format.md) — Complete `.wfl` format specification
- [docs/mcp-tools.md](docs/mcp-tools.md) — MCP tool reference with full input/output schemas
- [docs/agent-guide.md](docs/agent-guide.md) — How agents should use wfl to implement features
