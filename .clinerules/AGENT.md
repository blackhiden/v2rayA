# AGENT.md

> Unified guide for all contributors — human, AI, automation, or bot — participating in the maintenance, extension, or automation of the v2rayA project.

---

## Purpose

`AGENT.md` defines the expectations, workflows, and principles that every agent and contributor must follow when working in this codebase. It combines:

- project conventions and scope discipline,
- documentation and session workflows,
- automated and AI-assisted development patterns,
- validation, testing, security, and release expectations.

Anyone or anything making changes should treat this document as binding guidance, not as a suggestion.

---

## 1. Audience

This document applies to:

- **human contributors** (maintainers, developers, reviewers),
- **AI-assisted workflows** (code generation, review, analysis, search, planning),
- **automation and bots** (CI, scripts, tooling, integration agents, release helpers).

If you are automating part of the project, read this file first and follow it as strictly as a human would.

---

## 2. Project Orientation

v2rayA is a V2Ray client with a web GUI, cross-platform support, and a service-oriented internal structure. Before making changes, understand:

- **Frontend:** Web GUI lives in the `gui/` tree and is built/bundled for embedding.
- **Core/server components:** The service layer and related packages drive local proxy behavior, scheduling, and runtime integration.
- **Kernel integration:** Deep integration points exist for core routing behavior, observatory, and live flow support.
- **Docs:** Most authoritative references live in `docs/`, not in the repository root; the root still contains `README.md` and (transient) `session.md`.
- **Static web assets:** `web/` contains built frontend and branding resources.

Always start from:

- `README.md` — user-facing orientation
- `docs/` — architecture, specification, implementation notes, security/evaluation docs, and roadmap material

Do not assume anything about behavior, APIs, or build invariants without checking them.

---

## 3. Core Principles

All agents and humans must follow the same principles:

- Prefer the smallest simple, maintainable change.
- Prefer composition over inheritance where relevant.
- Follow existing architecture, conventions, and abstractions.
- Reuse existing code and documentation.
- Do not change unrelated behavior.
- Do not introduce unnecessary dependencies, abstractions, refactoring, or speculative features.
- Keep public-facing docs and internal docs synchronized.
- Validate before claiming success.

---

## 4. Mandatory Documentation Workflow

Documentation is part of implementation. Before doing **any** analysis, planning, coding, debugging, or refactoring:

1. **Read all relevant `.md` files** in the project root and in `docs/`, including subdirectories if appropriate.
2. **Read `session.md`** when it exists in the root. It is the working context for iterative changes and may contain current objectives, known issues, assumptions, and validation state.
3. If required project documentation is missing, create it in the project root before continuing.
4. Identify which documentation is authoritative for the current task.
5. Keep authoritative docs synchronized with the actual implementation.
6. When behavior, architecture, schema, API, configuration, security, testing, integration, or development workflow changes, update the affected docs in the same change.
7. Keep `README.md` synchronized with setup, usage, configuration, and capabilities.
8. Record significant decisions, assumptions, or code/documentation discrepancies in `session.md`.
9. Never create duplicate documentation when an existing document can be updated instead.

When in doubt, update the fewer, authoritative documents instead of creating new ones.

---

## 5. Session Continuity

Use `session.md` for short-lived working context, not permanent documentation. Keep it concise and current.

A useful `session.md` entry includes:

- current objective and progress,
- important decisions and assumptions,
- discoveries that affect implementation or documentation,
- known issues and limitations,
- pending work,
- relevant test results and validation notes.

Update `session.md` before finishing work and whenever the working context changes meaningfully.


---

## 6. Requirements and Architecture

Before implementation:

- Understand the current implementation and architecture.
- Read applicable specifications and identify:
  - requirements
  - constraints
  - inputs and outputs
  - edge cases
  - acceptance criteria
- Do not invent behavior when requirements are ambiguous.
- Document significant architectural decisions:
  - components
  - responsibilities
  - dependencies
  - data flow
  - constraints
  - trade-offs
- Ensure architecture documentation matches the actual implementation, not an older mental model.

---

## 7. Database and State Changes

For any state-store or database-related change, including config storage and runtime state evolution:

- Read the current schema and related documentation.
- Check migrations, relationships, constraints, indexes, query patterns, and existing data.
- Consider integrity, performance, compatibility, migration safety, and rollback.
- Add or update appropriate tests.
- Update documentation to reflect the actual schema and behavior.

---

## 8. Testing Expectations

For every feature, bug fix, or behavior change:

- Add or update appropriate tests when practical.
- Run relevant tests.
- Investigate failures; do not ignore them.
- Document meaningful testing limitations.
- Never claim a test was run if it was not.

Automated agents should avoid shortcuts such as assuming tests pass because earlier steps looked green.


---

## 13. Required Task Sequence

For every task, follow this order:

1. Read relevant documentation, including `session.md`.
2. Check for missing required documentation and create it if necessary.
3. Understand the existing implementation and architecture.
4. Identify applicable specification, database, security, testing, and integration requirements.
5. Check relevant TODOs, tasks, roadmap items, and deferred work.
6. Implement the smallest correct change.
7. Add or update tests.
8. Run relevant verification.
9. Update affected documentation.
10. Update `session.md`.
11. Confirm code, documentation, tests, and project state are consistent.

The final project state must be consistent, documented, tested, secure, and maintainable.

---

## 14. Scope Discipline

Do not:

- modify unrelated code or documentation,
- perform broad refactoring unless required,
- introduce speculative features,
- duplicate existing documentation,
- silently change established behavior.

Prefer incremental, reviewable changes that preserve the project's existing purpose and architecture.


---

## 9. Security Expectations

For security-sensitive changes and for any automation touching sensitive paths:

- Validate untrusted input.
- Verify authentication and authorization.
- Protect secrets and sensitive data.
- Use secure defaults.
- Consider injection, privilege escalation, data exposure, authentication bypass, and related attack paths.
- Add security tests when appropriate.
- Update security documentation when the security model changes.
- **Never** hardcode passwords, tokens, API keys, private keys, or other secrets.

---

## 10. External Integrations

For integration changes:

- Read existing integration documentation first.
- Verify authentication, authorization, request/response contracts, and compatibility.
- Handle failures appropriately.
- Consider timeouts, retries, idempotency, and rate limits where applicable.
- Add integration tests or suitable mocks.
- Update integration documentation when behavior changes.

---

## 11. Project Planning

Keep planning documents accurate when applicable:

- Track bugs, incomplete work, technical debt, and follow-up tasks.
- Keep TODOs specific and remove completed items.
- Keep roadmap items aligned with actual project direction.
- Keep intentionally deferred work separate from active tasks.
- Do not implement future-work items unless the current task requires them.
- Avoid speculative planning.

---

## 12. Contribution Workflow

When applicable, keep contribution-oriented documentation accurate for:

- development setup,
- required tools,
- coding conventions,
- branch and workflow conventions,
- testing,
- pull requests,
- commits,
- documentation requirements,
- common local commands.

If contribution guidance is spread across several files, keep the most relevant entry points consistent with one another.


---

## 15. Agent and Automation-Specific Expectations

Agents and automation must apply the same standard of care as human contributors. In practice, this means:

### 15.1 Before proposing or generating code

- Read the relevant docs, especially in `docs/`, plus `session.md` if present.
- Check whether the requested change already exists, is intentionally deferred, or is out of scope.
- Identify the smallest plausible change before generating large diffs.
- Prefer existing patterns and existing helper code over new abstractions.

### 15.2 While implementing

- Keep edits localized.
- Preserve existing naming conventions, imports, logging style, and error handling patterns as far as possible.
- Do not "improve" surrounding code unless that improvement is part of the task.
- When a requirement is ambiguous, stop and document the ambiguity instead of guessing silently.

### 15.3 Verification

- Build and run tests relevant to the changed area.
- If live feedback is available, use it.
- Include explicit verification steps when reporting results.
- If verification is blocked or incomplete, say so.

### 15.4 Documentation and session notes

- Update documentation alongside code.
- Update `session.md` when the working context changes.
- Prefer clarity and correctness over quantity.
- Avoid duplicating documentation; update existing authoritative documents instead.

### 15.5 Communication

- When generating or reviewing changes, explain:
  - what the change does,
  - why it is needed,
  - what is intentionally not changed,
  - what validation was performed,
  - what limitations remain
- If a decision could have security, migration, or compatibility impact, state that explicitly.


---

## 16. Common Pitfalls

- Editing files without reading the relevant documentation first.
- Assuming a missing file means the topic is absent; the project may centralize docs in `docs/`.
- Claiming tests passed without running them.

---

## 17. Reference Documents

Routine starting points:

- `README.md`
- `CHANGELOG.md`
- `docs/ARCHITECTURE.md`
- `docs/SPECIFICATION.md`
- `docs/ROADMAP.md`
- `docs/CONTRIBUTING.md`
- `session.md`

Read these first when working on architecture, production behavior, onboarding, or project direction. If a referenced document is missing from the root but present in `docs/`, treat the `docs/` copy as authoritative until the project structure changes.

---

## 18. If You Still Are Unsure

When requirements are unclear or a change could affect security, migration, protocol behavior, or release expectations:

1. Stop.
2. Record the uncertainty in `session.md`.
3. Narrow the change to the smallest safe step.
4. Request review or clarification before expanding the change.

It is better to leave a small, well-documented uncertainty than to silently invent behavior.



