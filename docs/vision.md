# Vision: Why wfl Exists

## The shift happening in software development

Agent-based development is changing what developers actually do. Code production — writing functions, wiring up routes, marshalling data — is increasingly handled by agents. What developers are left with is a harder, more valuable job: **platform-level thinking**.

That means:
- Designing the technical RFC that a feature needs before any code is written
- Deciding how a system should be structured and why
- Reviewing generated code for correctness, security, and long-term maintainability
- Scaling services and making architecture tradeoffs
- Verifying that what was built matches what was intended

The bottleneck is no longer code production. It is **intent communication**.

## The problem with existing approaches

When you tell an agent to "implement user authentication," you get something that works — but it may not match your architecture, your security model, your DB schema conventions, or the existing patterns in your codebase. The agent didn't have enough context.

The current workarounds are:
- Pasting large amounts of code into the context window (expensive, imprecise)
- Writing long text prompts describing what you want (verbose, not reviewable by teammates)
- Using `CLAUDE.md` files for global context (project-wide, not feature-scoped)
- Hoping the agent reads the right files (unreliable)

None of these give you something a teammate can review, a CI check can enforce, or an agent can query in structured chunks.

## What wfl is

wfl is a **structured intent file** that sits between the developer's thinking and the agent's execution.

It is:
- **Code-like** — developers write it in a familiar flow syntax, not prose
- **Reviewable** — renders as Mermaid sequence diagrams; peers review it before agents run
- **Queryable** — agents call focused MCP tools to read exactly the context they need
- **Versioned** — conventional commit semantics on intent blocks; breaking changes are explicit
- **Self-updating** — agents write back implementation notes post-execution

The `.wfl` file is the **contract** for a feature. The developer authors it; the team reviews it; agents implement from it.

## The role separation

| Role | What they do with wfl |
|------|----------------------|
| Developer | Writes `@@intent` blocks, annotates with `@context` pointers, reviews `@@impl` notes |
| Teammate (reviewer) | Reviews the rendered Mermaid diagram before implementation starts |
| Orchestrating agent | Calls `workflow.listFlows`, `workflow.getSubtasks`, dispatches subagents |
| Implementing subagent | Calls `workflow.getIntent`, `workflow.getContext`, builds the feature, calls `workflow.writeImpl` |
| CI / pre-commit hook | Runs `wfl validate` to ensure no intent was silently tampered |

## The traceability loop

After implementation, a developer can:

1. Open the `.wfl` file — see what was intended (`@@intent`) and what was built (`@@impl`)
2. Read the `@context` annotations — navigate directly to the relevant code locations
3. Check `wfl status` — see if any flows have drifted or have stale implementations
4. Read the `@@changelog` — understand the evolution of the feature over time

This closes the loop between **what you planned** and **what exists in the codebase** — without requiring you to read hundreds of lines of generated code first.

## Why it has to be a file (not a tool or UI)

The `.wfl` file lives in the repository. That means:
- It is versioned with `git` — every change to intent is tracked
- It is reviewed in pull requests — intent changes go through the same process as code changes
- It is diffable — you can `git diff` intent across branches
- It is portable — no account, no service, no lock-in

A UI tool or SaaS product would give you a nicer interface but would separate intent from code. The `.wfl` file keeps them together.

## What wfl is not

- **Not a task tracker** — it does not replace Linear, Jira, or GitHub Issues
- **Not a programming language** — it does not compile or execute
- **Not a documentation tool** — it is not written after the fact; it is written before implementation
- **Not agent-specific** — it works with any MCP-compatible agent, any LLM, any IDE

## Future directions

These are directions the tool may grow. None are committed roadmap items.

### `wfl rehash` command
When a developer intentionally edits an `@@intent` block, they should be able to recompute the hash in one command rather than manually. The recomputed hash is embedded and the changelog is prompted.

### Bidirectional navigation (IDE integration)
An IDE extension that lets a developer click a flow name in a `.wfl` file and jump to the files referenced by `@context` annotations — and conversely, right-click a function and see which workflow flow references it.

### Multi-file flow composition
A `@import: other.wfl:PaymentFlow` annotation that lets flows reference intent from other workflow files. Useful for large features that span multiple teams.

### Agent self-improvement of intent
After implementation, an agent that discovers the intent was ambiguous or incomplete can propose additions to the `@@intent` block as a PR comment, which the developer then approves before merging. The developer merges the improved intent alongside the code.

### `wfl diff <branch>`
Show which flows changed intent between two branches, what kind of bump (patch/minor/major), and which `@@impl` blocks are now stale as a result.

### Parallel subagent dispatch spec
A `@@dispatch` block type that an orchestrating agent writes to describe the parallelization strategy it used — which subagents ran, in what order, what each was responsible for. Makes the multi-agent execution auditable.

### Lock file (v2)
A `.wfl.lock` file (PID + timestamp) for concurrent write protection when multiple agents call `workflow.writeImpl` simultaneously on the same file.
