# .wfl File Format Specification

## Overview

A `.wfl` file is a plain text file containing one or more named **flows**. Each flow has three blocks: an `@@intent` block (developer-written, hash-locked), an `@@impl` block (agent-written, mutable), and is tracked in a `@@changelog` block.

Files use UTF-8 encoding. Line endings are normalized to LF (`\n`) on read — CRLF files are supported transparently.

---

## Block structure

Every block follows this pattern:

```
@@<kind> v<version> flow:<FlowName>
<content lines>
@@end[:<hash>]
```

The three block kinds are:

| Kind | Version required | Hash on `@@end` | Mutable by |
|------|-----------------|-----------------|------------|
| `intent` | Yes | **Required** | Developer only |
| `impl` | Yes | No | Agent (via `workflow.writeImpl`) |
| `changelog` | No | No | Append-only |

---

## `@@intent` block

```
@@intent v<semver> flow:<FlowName>
  [content]
@@end:sha256:<64-char-hex>
```

The intent block is the developer's specification of what a feature should do. It contains:

- **Comment lines** starting with `#` — notes, conventional commit messages, dependency declarations
- **Flow arrows** — `from -> to: label` lines describing interactions
- **Annotations** — `@context`, `@api`, `@db` lines providing context pointers

### Conventional commit in comments

Use the first comment line to describe the intent version using conventional commit format:

```
@@intent v1.0.0 flow:UserAuth
  # feat: initial JWT-based login flow
  ...
```

When bumping version, update this line:

```
@@intent v2.0.0 flow:UserAuth
  # feat!: migrate to OAuth2 provider
  # BREAKING: removed direct credential validation
  ...
```

### Flow arrows

```
  client -> gateway: POST /login
  gateway -> authService: validateCredentials()
  authService -> tokenStore: issueJWT(userId)
  gateway -> client: 200 + token
```

Format: `  <from> -> <to>[: <label>]`

- Leading whitespace (2+ spaces) is required to distinguish arrows from block headers
- The label after `:` is optional but recommended
- Participants are any string without whitespace (e.g., `client`, `authService`, `db`)
- Multiple arrows on consecutive lines are rendered in sequence

### Conditional / branching arrows

The format has no native branching syntax. Use square brackets in the label to express conditions — this is the standard convention:

```
  gateway -> authService: [password valid] issueJWT()
  gateway -> client: [password invalid] 401 Unauthorized
```

For a check-and-skip pattern (gate logic), use a semicolon-separated label:

```
  Redis -> service: [key exists] skip; [key absent] proceed
```

Both forms render cleanly as Mermaid labels and give implementing agents unambiguous branching intent. Do not use prose like "if the key exists, skip the upsert" in a label — keep it terse and bracket-delimited.

### Annotations

Annotations appear on lines immediately following the arrow they annotate, indented further:

```
  gateway -> authService: validateCredentials()
    @context: src/auth/validator.go:42
    @api: internal/auth
    @db: users.credentials
```

| Annotation | Format | Effect |
|------------|--------|--------|
| `@context` | `@context: <relative-path>:<line>` | Agent receives ±20 lines of that file around that line number |
| `@api` | `@api: <name>` | Metadata tag — no file resolution; passed through to agent as-is |
| `@db` | `@db: <name>` | Metadata tag — no file resolution; passed through to agent as-is |

`@context` paths are relative to the directory containing the `.wfl` file.

### Dependency declaration

Declare inter-flow dependencies in comments:

```
  # depends: TokenStore
  # depends: UserProfile
```

These are read by `workflow.getDependencies` to help orchestrating agents determine execution order.

### Hash lock

The `@@end` line for intent blocks must include a SHA256 hash:

```
@@end:sha256:78f60cf556b77da34f377e6b230a4f0cd49d244352bbd47d570b61d8f2d9d24b
```

**Hash computation rule:**
1. Take all lines between the `@@intent` header and the `@@end` line
2. Join them with `\n` (a single newline between each line, no leading or trailing newline)
3. Encode the result as UTF-8
4. Compute SHA256
5. Hex-encode the 32-byte digest

Example in Go:
```go
import "crypto/sha256"
import "encoding/hex"

h := sha256.Sum256([]byte(rawContent))
hash := hex.EncodeToString(h[:])
```

If the hash does not match when `wfl validate` is run, the file is rejected. This makes silent tampering detectable.

---

## `@@impl` block

```
@@impl v<semver> flow:<FlowName>
  [content]
@@end
```

The impl block is written by agents after implementation. It is free-form text — there is no parsing beyond the block boundaries.

Convention for agent-written impl notes:

```
@@impl v1.0.0 flow:UserAuth
  # Implemented 2026-10-06
  # src/auth/validator.go:42 — ValidateCredentials compares bcrypt hash
  # src/token/issuer.go:18  — HS256 JWT, 24h expiry, userId in sub claim
  # Tests: auth_test.go covers happy path, wrong password, expired token
  # Deviation: added rate limiting middleware not in original intent
@@end
```

### Version on impl blocks

The `@@impl` version should match the `@@intent` version it implements. When intent has a **major version bump**, the impl block is automatically flagged as **stale** — its version is still on the old major. To unblock agents from writing:

1. Developer edits the `@@impl` header to bump its version to match intent
2. Agents can now call `workflow.writeImpl` again

This is a deliberate friction point: the developer must consciously acknowledge the breaking change before agents proceed.

---

## `@@changelog` block

```
@@changelog flow:<FlowName>
  v2.0.0 feat!: migrate to OAuth2 — BREAKING dropped direct credentials
  v1.1.0 feat: add remember-me support
  v1.0.0 feat: initial JWT auth flow
@@end
```

- No version on the `@@changelog` header line (unlike `@@intent` and `@@impl`)
- Entries are listed newest-first by convention
- Each entry is one line: `v<version> <conventional-commit-message>`
- Entries are append-only — do not edit or remove past entries

---

## Multi-flow files

A single `.wfl` file can contain multiple flows. Each flow has its own set of three blocks. Flows are independent unless explicitly declared with `# depends:` comments.

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

---

## Versioning semantics

wfl uses semver with explicit meaning for each bump kind:

| Intent bump | Meaning | `@@impl` effect |
|-------------|---------|-----------------|
| `1.0.0 → 1.0.1` patch | Clarification; no structural change | `@@impl` stays valid |
| `1.0.0 → 1.1.0` minor | New scope added; non-breaking | `@@impl` stays valid; agent should re-read |
| `1.0.0 → 2.0.0` major | Breaking change to the flow structure | `@@impl` marked **stale**; agent blocked until developer bumps `@@impl` version |

The stale check is: `intent.major > impl.major`.

---

## Workflow file lifecycle

```
Developer writes @@intent
        ↓
Team reviews (via wfl render → Mermaid diagram in PR)
        ↓
wfl validate passes in CI / pre-commit hook
        ↓
Agent reads intent via MCP (getIntent, getContext, getSubtasks)
        ↓
Agent implements feature
        ↓
Agent writes back via workflow.writeImpl
        ↓
Developer reviews @@impl notes
        ↓
Feature merged — .wfl file committed alongside code
        ↓
Intent evolves → version bumped → agent re-implements stale flows
```

---

## Complete example

```
@@intent v1.1.0 flow:UserAuth
  # feat: add MFA support to login flow
  # depends: SMSProvider

  client -> gateway: POST /login { email, password, otpCode? }
  gateway -> authService: validateCredentials()
    @context: src/auth/validator.go:42
    @api: internal/auth
    @db: users.credentials
  authService -> mfaService: checkOTP(userId, otpCode)
    @context: src/mfa/checker.go:15
    @api: internal/mfa
  authService -> tokenStore: issueJWT(userId, mfaVerified)
    @context: src/token/issuer.go:18
  gateway -> client: 200 { token, expiresAt }
@@end:sha256:<hash>

@@impl v1.1.0 flow:UserAuth
  # Implemented 2026-10-06
  # src/auth/validator.go:42 — unchanged from v1.0.0
  # src/mfa/checker.go:15   — new; validates TOTP code against users.mfa_secret
  # src/token/issuer.go:18  — added mfa_verified boolean claim to JWT payload
  # Edge case: otpCode optional; if user has no MFA enrolled, mfaService no-ops
@@end

@@changelog flow:UserAuth
  v1.1.0 feat: add MFA support
  v1.0.0 feat: initial JWT auth flow
@@end
```
