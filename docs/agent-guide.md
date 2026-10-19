# Agent Guide: Using wfl to Implement Features

This document is written for agents — it describes exactly how to use the wfl MCP tools to implement a feature from a `.wfl` workflow file.

---

## Critical: the implementation loop is not complete until you call `workflow.writeImpl`

**Calling `workflow.writeImpl` after implementation is not optional.** It is the step that closes the traceability loop — without it, the developer has no record of what was built, where it lives in the codebase, or what deviations from intent occurred.

Do not report a feature as done until `workflow.writeImpl` has been called for every flow you implemented and `workflow.getStatus` returns `Drift: "in-sync"`.

---

## What you are working with

A `.wfl` file is a developer's specification of a feature. It contains:

- `@@intent` blocks — what the developer wants built (hash-locked, read-only to you)
- `@@impl` blocks — where you record what you built (writable by you)
- `@@changelog` blocks — version history (read-only, append-only)

Your job: read the intent, implement the feature, **write back what you did**.

---

## Before you start

Check that the workflow file is healthy:

```
workflow.listFlows({ workflow_file_path: "/abs/path/feature.wfl" })
```

Look at the `Drift` field for each flow:

| Drift | What to do |
|-------|-----------|
| `in-sync` | Already implemented. Review `@@impl` to understand what exists. |
| `impl-missing` | Needs implementation. Proceed. |
| `stale` | Intent has breaking changes (major version bump). **Stop — do not implement.** Tell the developer: the `@@impl` version needs to be bumped to acknowledge the breaking change before you can proceed. |
| `hash-mismatch` | Intent block has been tampered. **Stop.** Do not proceed. Tell the developer: `wfl validate` will show which flow is affected. |

Only proceed on flows with `Drift == "impl-missing"` or when explicitly asked to re-implement an `in-sync` flow.

---

## Step 1: Read the intent

```
workflow.getIntent({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl"
})
```

This returns the `@@intent` block content. Read it carefully:

- **Comment lines** (`# ...`) tell you the purpose, conventional commit type, and declared dependencies
- **Flow arrows** (`client -> gateway: POST /login`) are the interactions you need to implement
- **Annotations** are pointers to relevant code — use `workflow.getContext` to resolve them

---

## Step 2: Get real code context

```
workflow.getContext({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl"
})
```

This resolves every `@context: path/to/file.go:42` annotation to an actual ±20-line excerpt from that file. You get:

- `contexts[]` — array of resolved file excerpts
- `metadata.api[]` — API names (context for what external services are involved)
- `metadata.db[]` — DB table/collection names (context for what data is accessed)

Read the excerpts. They show existing patterns, function signatures, and data structures that your implementation should follow or extend.

---

## Step 3: Check dependencies

```
workflow.getDependencies({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl"
})
```

If a flow declares `# depends: OtherFlow`, implement that flow first (or confirm it is already `in-sync`). A flow that depends on an unimplemented flow will likely need to call code that doesn't exist yet.

---

## Step 4: Decompose if needed

For complex flows, get the subtask list:

```
workflow.getSubtasks({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl"
})
```

Each subtask maps to one arrow in the intent. Subtasks with the same `Order` value can be implemented in parallel. If you are an orchestrating agent, dispatch subagents per subtask group and aggregate their results.

---

## Step 5: Implement

Implement the feature using the code files referenced in `@context` annotations as your primary navigation points. Follow the patterns you observed in the excerpts.

Implement:
- The code changes described by the flow arrows
- Unit tests covering the happy path and key edge cases
- Any error handling implied by the flow

Do not:
- Modify the `@@intent` block — ever
- Modify the `@@changelog` block
- Change the `@@impl` version header — only the developer does this

---

## Step 6: Write back your notes

```
workflow.writeImpl({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl",
  content: "  # Implemented 2026-10-06\n  # src/auth/validator.go:42 — ValidateCredentials uses bcrypt comparison\n  # src/token/issuer.go:18  — HS256 JWT, 24h expiry, userId in sub claim\n  # Tests: auth_test.go happy path + wrong password + expired token\n  # No deviations from intent"
})
```

Good impl notes include:
- Date of implementation
- The actual file paths and line numbers where the implementation lives
- Key implementation decisions (algorithm used, library chosen, etc.)
- Any edge cases you handled that weren't in the intent
- Any deviations from the intent, and why

This is the audit trail. A developer reading `@@impl` should be able to understand what was built and navigate directly to the code.

---

## Step 7: Verify

```
workflow.getStatus({
  flow_name: "UserAuth",
  workflow_file_path: "/abs/path/feature.wfl"
})
```

Confirm `Drift == "in-sync"` before reporting the task as done.

---

## What to do in each situation

### The intent references a file that doesn't exist

`workflow.getContext` returns `exists: false` for that annotation. This is non-fatal — the file may be one you are supposed to create. Read the intent to determine whether the file should be created or whether the annotation is pointing to a file that will be provided by another flow.

### The flow is stale

You will see `Drift: "stale"` from `workflow.listFlows` or `workflow.getStatus`. Do not attempt to implement. Report to the developer:

> Flow `UserAuth` is stale: intent is at v2.0.0 but impl is at v1.0.0. This is a breaking change. Please bump the `@@impl` version header to v2.0.0 to acknowledge the breaking change, then I can proceed.

### You cannot implement a part of the intent

Write what you were able to do in `workflow.writeImpl` and note the gap explicitly:

```
  # Partial implementation 2026-10-06
  # Implemented: credential validation and JWT issuance
  # NOT IMPLEMENTED: MFA OTP check at mfaService step
  # Reason: src/mfa/checker.go does not exist; SMSProvider flow (dependency) is not yet implemented
  # Next: implement TokenRefresh flow first, then return here
```

### You discover the intent is ambiguous

Implement your best interpretation. In `@@impl`, note what you assumed:

```
  # Assumption: "validateCredentials()" implies bcrypt comparison against users table
  # Assumption: JWT expiry is 24h (not specified in intent)
  # Developer should review and update intent if assumptions are wrong
```

---

## Full example session

```
# 1. Discover flows
workflow.listFlows({ workflow_file_path: "/project/auth.wfl" })
→ UserAuth: impl-missing
→ TokenRefresh: impl-missing

# 2. Check dependencies
workflow.getDependencies({ flow_name: "TokenRefresh", ... })
→ dependencies: ["UserAuth"]

# 3. Implement UserAuth first (no dependencies)
workflow.getIntent({ flow_name: "UserAuth", ... })
workflow.getContext({ flow_name: "UserAuth", ... })
# ... read excerpts, implement code ...

# 4. Write back
workflow.writeImpl({
  flow_name: "UserAuth",
  content: "  # Implemented\n  # src/auth/validator.go:42 ..."
})

# 5. Verify
workflow.getStatus({ flow_name: "UserAuth", ... })
→ Drift: "in-sync" ✓

# 6. Now implement TokenRefresh (dependency satisfied)
workflow.getIntent({ flow_name: "TokenRefresh", ... })
# ... implement ...
workflow.writeImpl({ flow_name: "TokenRefresh", content: "..." })
workflow.getStatus({ flow_name: "TokenRefresh", ... })
→ Drift: "in-sync" ✓
```

---

## Rules summary

| Rule | Reason |
|------|--------|
| Never modify `@@intent` blocks | Hash-locked; any change is detectable by `wfl validate` and considered tampering |
| Never modify `@@changelog` blocks | Append-only audit trail |
| Never bump `@@impl` version | Only the developer acknowledges breaking intent changes |
| Check `Drift` before implementing | Do not proceed on stale or hash-mismatch flows |
| Always call `writeImpl` after implementing | Closes the loop; makes the implementation auditable |
| Use `getContext` before writing code | Follow existing patterns; avoid re-implementing what already exists |
