# CLAUDE.md — Igor

**Version:** 1.1

This file is auto-loaded by Claude Code in every conversation. Read it in full before writing, modifying, or reviewing any code. All work is held to the highest standard of correctness, security, documentation, and testability.

---

## 1. Project Overview

Igor is a node reservation manager for clusters that host ad-hoc multi-user/multi-project work. Igor differs from traditional cluster reservation systems like SLURM in that it provisions cluster nodes with an OS of the reservation owner's choice that is installed onto the reserved nodes' bare-metal via netboot and TFTP. This allows the users complete control over their provisioned nodes' runtime environment.

Users can also create groups that share access to reservation management. Administrators can fine-tune reservation creation and resource access across a wide variety of settings from open and permissive to regulated and tightly controlled.

Official project github URL: https://github.com/sandia-minimega/igor2
Project fork URL where this coding effort lives: https://github.com/afbagwell/igor2
Online documentation: https://www.sandia.gov/igor/documentation/

#### 1a. Online Documentation Site Map

The published documentation is the closest thing this project has to a written
behavioral specification, since `docs/engineering_docs/` is not yet populated. It is
useful for checking documented behavior against actual behavior.

All paths below are relative to `https://www.sandia.gov/igor/documentation/`.

| Guide | Page | Path |
|---|---|---|
| Administration | Index | `administration-guide/` |
| Administration | Operational Requirements | `administration-guide/operational-requirements/` |
| Administration | Setup | `administration-guide/setup/` |
| Administration | Concepts & Configuration Options | `administration-guide/concepts-configuration/` |
| User | Index | `user-guide/` |
| User | Getting Started | `user-guide/getting-started/` |
| User | Reservations | `user-guide/reservations/` |
| User | Reservation Management | `user-guide/reservation-management/` |
| User | Distros | `user-guide/distros/` |
| User | Profiles | `user-guide/profiles/` |
| User | Groups | `user-guide/groups/` |
| Igor-web | User Guide | `igor-web-user-guide/` |

Also `https://www.sandia.gov/igor/` (home) and `https://www.sandia.gov/igor/download/`.

**Read these pages on demand**, when a task actually touches that subject area. Do not
bulk-read the whole set up front.

**Fetched pages come back summarized, not verbatim.** Never quote this documentation as
authoritative from a paraphrase. When an exact flag name, configuration key, default
value, or wording is load-bearing, explicitly request verbatim extraction, and prefer
confirming the behavior against the source code or a running DEVMODE instance.

Be especially careful where a page states a constraint and a default in the same breath;
a summary can blur them. For example `MinReserveTime` has an absolute floor of 10
minutes (configurable minimum, typically only useful for testing) while the default when
unset is 30 minutes — two different values, and only one of them is a limit.

#### 1b. Project Documents

| Document | Purpose |
|---|---|
| `docs/bug_tracking/bug_tracker.md` | Index, progress tracker and quick summary for defects found during code analysis. |
| `docs/bug_tracking/BUG-XXX.md` | Full detail for one bug, one file per assigned number. |
| `docs/engineering_docs/` | Architecture, requirements, and status documents governed by §9. Not yet populated. |

**`docs/bug_tracking/`** records only **concrete, reproducible** defects — never
hypothetical problems that require exotic or assumed circumstances. Before adding an
entry, establish that user or administrator input can actually reach the code path, and
prefer demonstrating the failure with a run or a test over reasoning from source alone.

**Consult and update `bug_tracker.md` first.** It holds the summary table — ID, status,
component, severity and one-line description, each row linking to its detail document —
and it is the file to read for current state. It also defines the status values and the
procedure for adding a bug.

**Each bug gets its own `BUG-XXX.md`** in the same folder, numbered from the next free
ID in the summary table. A detail document carries the location and cause, a
reproduction, the impact, and — once resolved — the fixing commit and the test that
covers it. Add the summary row and the detail document together; neither is complete
alone.

**Cross-reference related bugs in both directions.** Bugs are rarely independent, and a
relationship recorded in only one place goes stale silently. Every relationship belongs
in the tracker's Relationships table *and* in a Relationships section in each detail
document involved. The tracker defines the vocabulary: **blocks / blocked by** for a hard
dependency where one bug cannot be closed until another is fixed, **subsumes / part of**
where one bug is a specific instance of a broader class, and **related to** where two
bugs share a root cause but are isolated and can be fixed in either order. Say which bug
must be fixed first and why — the blocking direction is not always the obvious one.

Keep the tracker's table lean, since it is the at-a-glance view: order rows by the ID in
the first column to match the summary table, list a symmetric relationship once under the
lower ID, and omit a `part of` row that a `blocked by` row already implies. Detail that
does not survive this pruning belongs in the detail documents, which carry the full
relationship picture without restriction.

When a finding looks alarming but proves unreachable, record it in `bug_tracker.md`'s
**"Not tracked as bugs"** section together with the reason it was rejected, so that it is
not investigated a second time. Rejected findings stay in the index and do not get a
`BUG-XXX.md`.

Update these documents in the same commit as the fix they describe, and bump the
tracker's version and Revision History per the §9.4 authoring rules.

---

## 2. Architecture Reference

This is a brownfield open-source coding project. Architecture documents as described elsewhere in this file do not yet exist in the source code repository. In the absence of such documents or specific details within them, the source code serves as the architectural reference.

**Key architectural constraints:**

- Igor server is designed to run in a Linux environment with command line support for invoking other Linux command-line tools. Windows is not supported.
- Igor's server-client architecture communicates over HTTP API calls.
- Igor's CLI is completely stateless. Whenever a command is run it performs a complete operation and exits. It has no database.
- The majority of persistent state lives in the igor-server database. At this time only SQLite is supported. Any in-memory-only state (e.g., the admin 'elevate' map) operates like a cache that does not need to survive restarts or power loss.
- Any architecture documents are authoritative. Do not violate component boundaries, skip abstraction interfaces, or introduce new infrastructure dependencies without explicit design discussion.

#### 2a. Minimum software versions required to build Igor

- Go 1.24.x
- NodeJS 22.x
- NPM 10.x
- Vue.js (version 2)

---

## 3. Code Style & Formatting

### Go
- **Formatter:** goimports
- **Linting:** golangci-lint

### Vue.js / JavaScript
- **Linting:** ESLint strict configuration + eslint-plugin-vue
- **Formatting:** Prettier

### Commits
- **Format:** Conventional Commits with scope.
- **Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `security`
- **Scopes:** `auth`, `core`, `shared`, `db`, `deploy`, `cli`, `web-frontend`
- Example: `feat(core): implement SelfLickingIceCreamCone service`

---

## 4. Security Standards

- **No hardcoded secrets.** No API keys, passwords, tokens, or private keys in source code. The only exception is defaults used in initial setup.
- **Secret handling:** Never log secrets (tokens, API keys, passwords, etc.). Never return secrets in API responses.
- **Error responses:** Never expose raw exception messages to clients in production.

---

## 5. Git Standards

- **Frequent commits.** Each commit is one logical, self-contained change that leaves the codebase passing.
- **Conventional Commits format.** See section 3 for types and scopes.
- **Never skip hooks.** Pre-commit hooks, if they exist, should always be run.
- **Never commit or force-push to main.**
- **Never commit:** config files with values that assume anything other than defaults on a newly deployed instance, secrets/keys, commented-out code, print statements directly to the console, build artifacts, or failing tests. Do not override .gitignore to commit development artifacts in the project folder.
- **All tests must pass** before committing.

---

## 6. Database Standards

- **SQLite 3** is the official supported database for this project.
- **GORM ORM** for all table definitions and queries except where certain queries for SQLite cannot be expressed with GORM.

---

## 7. What Claude Should Always Do

- Read any relevant section of `docs/engineering_docs/architecture_design.md` before implementing any new feature or modifying an existing one.
- Write tests alongside implementation -- never after. Tests and implementation in the same commit.
- Commit frequently in small, logical units. Prefer five small commits over one large one.
- Raise concerns explicitly before proceeding if a requested change conflicts with the architecture, a security requirement, or a documented constraint.
- Keep the GORM model definitions as the ground truth for the data model. If a code change implies a schema change, write the `db-migrate` migration in the same branch.
- Use structured logging as defined in the logger.go file. Printing to the console is only permitted for the CLI client.
- Follow the typed error hierarchy per package. Never return a bare `errors.New` or `fmt.Errorf` from business logic where a typed error exists; wrap with `%w` so callers can use `errors.As`.

---

## 8. What Claude Should Never Do

- Never skip tests or stub them with `// TODO: add tests later`.
- Never disable or weaken a linting, type-checking, or formatting rule to make code pass CI.
- Never store state in a service process (in-memory caches, module-level mutable variables).
- Never add a dependency without checking its license, maintenance status, and known CVEs. Document rationale in the PR.
- Never expose unauthenticated endpoints except those intended for public access.
- Never write a migration that drops a column/table without removing all code references in the same PR.
- Never log or return secrets in API responses.
- Never commit secrets, commented-out code, or failing tests.

---

## 9. Decision Capture Protocol

System-level decisions made during a coding session are captured in
`docs/engineering_docs/` **in the same commit** as the code change, so documentation and
code never drift.

### 9.1 What triggers the protocol

| Change type | Documents to update |
|---|---|
| New or modified requirement (FR, NFR, SEC) | `requirements_specification.md` + `traceability_matrix.md` + `implementation_progress.md` |
| New architectural decision | New `ADR-nnn` in `architecture_design.md` §2, plus any affected sections |
| Schema / data-model change | `architecture_design.md` §4 + migration in the same branch |
| New or changed public API/interface | `architecture_design.md` §5 |
| New extension implementation | `architecture_design.md` §5 + `technology_selection.md` if a new dependency is introduced |
| New technology / library / infrastructure dependency | `technology_selection.md` (+ architecture if topology changes) |
| Implementation status change | `implementation_progress.md` + `traceability_matrix.md` |
| New or changed security control | `security.md` + `requirements_specification.md` §5 |

### 9.2 What does NOT trigger the protocol

- Bug fixes that don't change documented behavior.
- Internal refactors with no public-API or schema change.
- Test additions or improvements.
- Routine implementation of an already-specified requirement (only the status trackers move).
- Comment, docstring, or formatting cleanups.

### 9.3 Process

1. **Identify** which documents need updating (§9.1).
2. **Propose** a single **batched** set of edits before writing code:
   ```
   File: docs/engineering_docs/<file>.md
   Section: <heading>
   Current:
       <existing text>
   Proposed:
       <new text>
   ```
   Group all edits for the change into one proposal — never trickle them out file-by-file.
3. **Apply** the approved edits and the code change together in a single commit.

### 9.4 Authoring rules

- **Requirement IDs:** auto-pick the next free ID in the right category and confirm before edits land.
- **ADRs:** append the next sequential `ADR-nnn`. Use the standard fields: Decision, Context / Rationale, Requirements Satisfied, Consequences. Never renumber.
- **Versioning:** bump the minor version of any updated doc on every accepted change; major only for breaking restructures.
- **Revision History:** add a row to the doc's `## Revision History` for every accepted change: `| Version | Date | Author | Change |`.
- **Commits:** bundle doc edits and code in one commit; reference new requirement/ADR IDs in the body.

### 9.5 Drift check

On request, sweep the codebase against `implementation_progress.md` and
`traceability_matrix.md` and report drift — features marked Implemented but missing in
code, or code not reflected in the docs. Run ad hoc, not on a schedule.

---

## Revision History

| Version | Date | Author | Change |
|---|---|---|---|
| 1.0 | 2026-07-29 | Allen Bagwell | Initial baseline from current checkout of version v2.3.2 |
| 1.1 | 2026-07-30 | Claude | Added §1a online documentation site map and on-demand reading instruction; added §1b project documents describing the bug tracking layout; corrected typos and replaced Python-template leftovers in §5 and §7 with their Go/GORM equivalents |
