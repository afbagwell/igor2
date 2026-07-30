# Igor Bug Tracker

**Version:** 1.5

Index and progress tracker for concrete, reproducible defects found during code
analysis. Hypothetical or exotic-circumstance concerns are **not** recorded here.

Each bug has its own detail document, `BUG-XXX.md`, in this folder. This file
carries only the summary table and the rejected findings below it; consult and
update it first, then open the individual document for the full account.

**Status values:** `open` · `triage` (awaiting maintainer decision) · `fixed` (commit ref) · `wontfix`

## Summary

| ID | Status | Component | Severity | Summary |
|---|---|---|---|---|
| [BUG-001](BUG-001.md) | fixed (`0f23ca6`) | cli | medium | A `%` in the cluster MOTD corrupts `igor show` output for every user |
| [BUG-002](BUG-002.md) | fixed (`3ea340e`) | core | low | A `%` in the `duration` query param yields a mangled stats error message |
| [BUG-003](BUG-003.md) | fixed (`594ee54`) | core, cli | medium | 40 non-constant format-string call sites; blocks `go test ./...` |

## Relationships

Bugs are rarely independent. Record every relationship in **both** directions — here and
in each detail document — so that neither view can go stale on its own.

**Vocabulary:**

| Term | Meaning |
|---|---|
| **blocks** / **blocked by** | A hard dependency. The blocked bug cannot be closed until the blocking bug is fixed, because the blocked bug's completion criterion depends on it. |
| **subsumes** / **part of** | One bug is a broader class; the other is a specific instance within it. Fixing the class necessarily fixes the instance, but not the reverse. Omitted from the table below where a `blocked by` row already implies it. |
| **related to** | Shares a root cause or subsystem, but the two are isolated — either can be fixed first, and neither blocks the other. |

**Current relationships**, in ID order:

| Bug | Relationship | Bug | Note |
|---|---|---|---|
| [BUG-001](BUG-001.md) | related to | [BUG-002](BUG-002.md) | Same root cause in different binaries; isolated, and fixed in separate commits. |
| [BUG-003](BUG-003.md) | blocked by | [BUG-001](BUG-001.md), [BUG-002](BUG-002.md) | BUG-003 closes only when `go vet` is clean, which requires every one of its 40 sites — including the two owned by BUG-001 and BUG-002. |

Rows are ordered by the ID in the first column, matching the summary table. A symmetric
relationship is listed once, under the lower ID.

**Omit a `part of` row when a `blocked by` row already implies it.** BUG-001 and BUG-002
are both part of BUG-003, but the `blocked by` row above already establishes that, so
repeating it here would only add clutter. The containment detail — which specific call
sites belong to the class — lives in each bug's own Relationships section.

The blocking direction is worth stating plainly, because it is the opposite of what the
severity ordering suggests: the two small, specific bugs are **prerequisites** for the
large umbrella one, not the other way round. BUG-003's status therefore cites `594ee54`,
the last of its three commits, rather than the first.

## Adding a bug

1. Take the next free `BUG-XXX` number from the table above.
2. Create `BUG-XXX.md` in this folder, following the layout of the existing
   documents: a title, a status/component/severity line, the date found and the
   affected version, then Location, Cause, Reproduction, Impact and — once
   resolved — Resolution naming the fixing commit and the covering test.
3. Add a row to the summary table linking to it.
4. Record any relationship to an existing bug in **both** the Relationships table above
   and a Relationships section in the new detail document, using the vocabulary defined
   there. State blocking dependencies explicitly — which bug must be fixed first, and
   why. Keep the table lean: insert the row in ID order, list a symmetric relationship
   once under the lower ID, and leave out a `part of` row that a `blocked by` row already
   implies. The detail document keeps the full picture either way.
5. Establish that user or administrator input can actually reach the code path
   before adding an entry at all. Prefer demonstrating the failure with a run or a
   test over reasoning from source alone.

## Not tracked as bugs

Noted during analysis, deliberately **not** catalogued — recorded here so they
are not re-investigated.

- **`login_handler.go:68`** builds an error line as
  `actionPrefix + " " + username + " " + password + " ..."` and puts it in both the
  log and the client response body. This looks like a secret-logging violation,
  but it is not reachable: `http.Request.BasicAuth` returns `"", "", false` on
  every failure path, so when `baOK` is false both values are empty strings. The
  concatenation is dead code that would become a real credential leak if the
  stdlib contract ever changed. Worth cleaning up; not a live defect.

## Revision History

| Version | Date | Author | Change |
|---|---|---|---|
| 1.0 | 2026-07-29 | Claude | Initial tracker; BUG-001, BUG-002, BUG-003 |
| 1.1 | 2026-07-29 | Claude | BUG-001, BUG-002, BUG-003 marked fixed with commit refs and resolution notes |
| 1.2 | 2026-07-29 | Claude | Added live before/after verification of BUG-001 and BUG-002 against a DEVMODE server |
| 1.3 | 2026-07-30 | Claude | Split each bug's detail into its own BUG-XXX.md; this file is now index, progress tracker and rejected findings. Removed stray closing tags erroneously written at the end of the file in 5b0c12a |
| 1.4 | 2026-07-30 | Claude | Added Relationships vocabulary and cross-reference table; recorded that BUG-003 is blocked by BUG-001 and BUG-002, and mirrored each relationship into the detail documents |
| 1.5 | 2026-07-30 | Claude | Ordered the Relationships table by ID to match the summary table, and dropped `part of` rows already implied by a `blocked by` row; containment detail remains in the detail documents |
