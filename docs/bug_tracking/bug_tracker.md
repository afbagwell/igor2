# Igor Bug Tracker

**Version:** 1.17

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
| [BUG-004](BUG-004.md) | fixed (`267035c`) | core | medium | Reservation owned by a user with no email retries an undeliverable warning every minute forever |
| [BUG-005](BUG-005.md) | fixed (`f5aa095`) | core | medium | `helpLink` (a URL) is used as an email recipient, so the account-removal alert never sends |
| [BUG-006](BUG-006.md) | fixed (`c3f2549`) | core | high | Removing a group owner executes a nil template and panics the server |
| [BUG-007](BUG-007.md) | fixed (`c3f2549`) | core | high | With `resNotifyOn: false`, every reservation start and expiry executes a nil template and panics the server |
| [BUG-008](BUG-008.md) | fixed (`934e720`) | core | high | Arista VLAN RPC has no timeout; a silent switch blocks all writes server-wide indefinitely |
| [BUG-009](BUG-009.md) | open | core | high | Power commands pass timeout `0`, running unbounded while holding the global write mutex |
| [BUG-010](BUG-010.md) | fixed (`05e94ed`) | core | high | `handleCreateReservations` unlocks without `defer`, so a panic permanently leaks the global write mutex |
| [BUG-011](BUG-011.md) | open | core | high | `findBestSolution` panics on index out of range with two or more restricted host policies |
| [BUG-012](BUG-012.md) | open | core | medium | Host delete/update blocks all writes on an unbuffered channel send to the probe manager |
| [BUG-013](BUG-013.md) | fixed (`528e6f0`, `6034343`) | core | medium | `Shutdown` has no timeout, so a wedged request makes restart require SIGKILL |
| [BUG-014](BUG-014.md) | open | core | medium | `panicHandler` calls `logger.Panic()` and re-panics, so the 500 response is never written |
| [BUG-015](BUG-015.md) | open | core | medium | `igor sync arista` panics on any switch error response via unchecked type assertions |
| [BUG-016](BUG-016.md) | fixed (`934e720`) | core | medium | An empty `networkPassword` mangles every Arista error message into unreadable output |
| [BUG-017](BUG-017.md) | fixed (`1ad2730`) | core | high | Image registration deadlocks against the initrd worker, wedging all writes permanently |

BUG-008 through BUG-014 were found together while investigating an intermittent production
condition in which all database-writing commands hang while reads continue to work. They
share one root: a single process-wide `dbAccess` mutex (`database.go:21`) guards every
write handler, read handlers take no lock at all, and several code paths hold that mutex
across unbounded external I/O. The investigation has **not** yet established which of them
causes the observed outage; see the Relationships notes and each document's Reproduction
section for what is demonstrated versus traced.

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
| [BUG-004](BUG-004.md) | related to | [BUG-005](BUG-005.md) | Both follow from `igor-admin` being seeded with no email, but fail differently — empty recipient list vs. unparseable address. Isolated; either order. |
| [BUG-006](BUG-006.md) | related to | [BUG-007](BUG-007.md) | Same failure mechanism — a nil template reaching `Execute` — from different causes: never written vs. registered only on some configurations. Isolated; either order. |
| [BUG-008](BUG-008.md) | related to | [BUG-009](BUG-009.md) | Same class: an unbounded external call made while holding `dbAccess`. Different subsystems (switch RPC vs. IPMI) and both fixed by the same structural change. Isolated; either order. |
| [BUG-008](BUG-008.md) | related to | [BUG-013](BUG-013.md) | BUG-013 is why a hang from BUG-008, BUG-009, BUG-010 or BUG-012 can only be cleared by SIGKILL. Recorded once here rather than under each. |
| [BUG-009](BUG-009.md) | related to | [BUG-012](BUG-012.md) | BUG-012's stall duration is set by the probe sweep, governed by the same `DefaultRunner` timeout, retry and concurrency knobs as BUG-009. |
| [BUG-010](BUG-010.md) | related to | [BUG-011](BUG-011.md) | BUG-011 is a demonstrated panic source inside the exact call BUG-010 leaves unguarded; together they turn one request into a permanent write outage. Composition, not containment. |
| [BUG-010](BUG-010.md) | related to | [BUG-014](BUG-014.md) | Both concern the aftermath of a handler panic — BUG-014 suppresses the response and splits the diagnostics, BUG-010 leaks the mutex. Independent; either order. |
| [BUG-014](BUG-014.md) | related to | [BUG-015](BUG-015.md) | BUG-014 turns the BUG-015 panic into a silent dropped connection instead of a 500 carrying the switch's error text. Fixing BUG-014 alone makes BUG-015 far less confusing without fixing it. |
| [BUG-012](BUG-012.md) | related to | [BUG-017](BUG-017.md) | Same shape: a channel send made while holding `dbAccess`. BUG-012 stalls for a bounded probe sweep; BUG-017 never clears, because its receiver waits on the very mutex the sender holds. Isolated; either order. |
| [BUG-013](BUG-013.md) | related to | [BUG-017](BUG-017.md) | BUG-013 is why the BUG-017 wedge cannot be cleared by a normal restart. |
| [BUG-015](BUG-015.md) | related to | [BUG-016](BUG-016.md) | Both are error-handling defects in `network_arista.go` that surface only when the switch returns something other than success. Isolated; either order. |

Rows are ordered by the ID in the first column, matching the summary table. A symmetric
relationship is listed once, under the lower ID.

Two directions in the new cluster are easy to get backwards. **BUG-010 before BUG-011:**
BUG-011 is one known panic source, while BUG-010 turns *every* panic source in the create
path — known and unknown — into a permanent outage, so it is the higher-value fix if only
one is done. **BUG-008 before BUG-009:** both are unbounded external calls under the same
mutex, but IPMI is UDP and `ipmitool` self-limits, so BUG-009 stalls for minutes and then
clears. Only BUG-008 can hang until the process is killed. A wedge that survives until
SIGKILL is BUG-008's signature, not BUG-009's.

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

- **`resNotifyChan` lock inversion.** Producers send on `resNotifyChan` while holding
  `dbAccess` (`scheduling.go:325` and `:589`), and the consumer `processResNotifyEvent`
  acquires `dbAccess` itself (`notify.go:641`). That is a genuine lock-ordering inversion:
  if the channel buffer ever fills, a producer blocks on the send while holding the mutex
  the consumer needs, and neither can proceed — a permanent, silent deadlock.

  Not tracked, because filling it is not a realistic scenario. The buffer is 100
  (`server.go:28`), and the consumer's slowest step is bounded: `gomail.NewDialer` defaults
  `Timeout` to 10 seconds (`gopkg.in/mail.v2@v2.3.1/smtp.go:44-61`), so SMTP cannot stall
  it indefinitely. Draining 100 queued events therefore needs a burst that outpaces a
  consumer which is never blocked for more than ~10s at a time. Worth revisiting if the
  buffer is ever reduced, if the SMTP timeout is ever disabled (`Timeout: 0` is legal), or
  if a future notify type does unbounded work — any of which would make it reachable.

  Note that [BUG-004](BUG-004.md), now fixed, previously re-queued an undeliverable warning
  every minute for every affected reservation, which is exactly the kind of burst that
  could have filled the buffer. That is one reason this is recorded rather than dismissed.

  **Update 2026-08-03:** the identical inversion on a *different* channel is now tracked as
  [BUG-017](BUG-017.md), and it is reachable. The distinction is only how the buffer fills:
  `resNotifyChan` needs an implausible burst, whereas the initrd queue is filled **by
  design** at startup by `EnqueuePendingJobs`, which pushes the entire unprocessed-image
  backlog with no bound. This entry stays rejected on its own merits, but the two share a
  root and should be fixed together — the rule wanted here is "never send on a channel while
  holding `dbAccess`", not two separate patches.

  **Update 2026-08-12:** BUG-017 is fixed in `1ad2730`; this one is not, and the rule above
  was not adopted wholesale. `scheduling.go:325` and `:589` still send on `resNotifyChan`
  with `dbAccess` held. The rejection reasoning is unchanged and still holds — the consumer
  remains bounded by the 10s SMTP timeout — so this stays out of the tracker. But note that
  the argument is now doing more work than it was: it is the *only* thing standing between
  this code and the failure BUG-017 turned out to produce, and BUG-017's fix demonstrates
  that moving a hand-off past the commit is a contained change. Worth doing on the next
  occasion that file is open, rather than waiting for the buffer to be reduced.

- **`allowImageUpload` does not gate `POST /images/register`.** The setting is checked in
  exactly one place, `distro_create.go:93`, on the path where a user attaches a KI pair to a
  distro-create request. The dedicated image-registration endpoint never consults it, and its
  route chain (`routes.go:369-373`) adds only `hcDefaultChain`, `hcAuthChain` and parameter
  validation — no admin middleware. That looks like a way to register an image with the
  setting off, and then reference it by `imageRef` on the `distro_create.go:76-84` branch,
  which has no gate either.

  **Investigated 2026-08-12 and rejected: the endpoint is admin-only, just not by the route
  chain.** `authzHandler` derives the required permission from the URL and method rather than
  from the chain, so an absent admin middleware proves nothing. For `POST /images/register`
  it builds `images:*:create` — wildcard in the middle because the route has no `:imageName`
  segment, so `paramsHandler` puts no params in the context (`middleware.go:45`) and authz
  takes its `ps == nil` branch. The permission every user gets through the `all` group is
  `groups,reservations,distros,profiles:*:create` (`database.go:155-163`). `images` is
  deliberately absent from that list, so a non-admin is refused; only the bare `*` held by an
  elevated admin implies it. Confirmed by driving the real `Permission.Implies` against that
  exact default: it implies `distros:*:create` and does not imply `images:*:create`.

  So there is no bypass. A regular user cannot complete step one, and an elevated admin
  registering an image is the documented intent, not an escalation: `igor-server.yaml:97-101`
  describes `allowImageUpload` as governing whether *a user* may upload a KI pair, and says
  that otherwise this is "something only a sysadmin could do". Code and documentation agree.

  Recorded because the route registration genuinely reads as unguarded, and because that
  misreading was made once already — it produced an incorrect reachability claim in
  [BUG-017](BUG-017.md), since corrected. Anyone auditing route chains in this codebase
  should check `authz.go` before concluding an endpoint is open.

- **Igor cannot talk to a VLAN switch over TLS.** The scheme is hardcoded to `http://` at
  `network_arista.go:77` and no configuration key selects it, so every deployment with
  `vlan.network` set transmits switch configuration commands — and the
  `vlan.networkPassword` credential — in cleartext, with no supported way to opt out.

  Not tracked as a bug because it is not a defect: the code does what it was written to do
  and nothing malfunctions. It is a missing capability, and one whose consequences fall on
  downstream deployments rather than on this project's own instance. Recording it here
  would understate it, so it has its own document: **[ISSUE-001](../ISSUE-001.md)**, which
  covers the exposure for sites that set a password and for sites that do not, and — most
  importantly — why a naive fix is worse than the status quo. The transport already carries
  `InsecureSkipVerify: true` as dead code, so merely emitting `https://` would ship TLS that
  encrypts without authenticating while looking secure to both deployer and auditor.

  Any work here needs the §9 treatment, not a bug fix.

## Revision History

| Version | Date | Author | Change |
|---|---|---|---|
| 1.0 | 2026-07-29 | Claude | Initial tracker; BUG-001, BUG-002, BUG-003 |
| 1.1 | 2026-07-29 | Claude | BUG-001, BUG-002, BUG-003 marked fixed with commit refs and resolution notes |
| 1.2 | 2026-07-29 | Claude | Added live before/after verification of BUG-001 and BUG-002 against a DEVMODE server |
| 1.3 | 2026-07-30 | Claude | Split each bug's detail into its own BUG-XXX.md; this file is now index, progress tracker and rejected findings. Removed stray closing tags erroneously written at the end of the file in 5b0c12a |
| 1.4 | 2026-07-30 | Claude | Added Relationships vocabulary and cross-reference table; recorded that BUG-003 is blocked by BUG-001 and BUG-002, and mirrored each relationship into the detail documents |
| 1.5 | 2026-07-30 | Claude | Ordered the Relationships table by ID to match the summary table, and dropped `part of` rows already implied by a `blocked by` row; containment detail remains in the detail documents |
| 1.6 | 2026-07-30 | Claude | Added BUG-004 (fixed), and BUG-005 and BUG-006 (open) found while investigating it; recorded the BUG-004/BUG-005 relationship |
| 1.7 | 2026-07-30 | Claude | BUG-005 marked fixed; corrected the claim in BUG-004 and BUG-005 that gomail rejects a bad address before connecting — `DialAndSend` dials first, then parses recipients |
| 1.8 | 2026-07-30 | Claude | BUG-006 marked fixed; added BUG-007, a second nil-template server panic found while fixing it, and recorded the BUG-006/BUG-007 relationship |
| 1.9 | 2026-07-30 | Allen Bagwell, Claude | Recorded live production verification of BUG-004 (cause and retry loop both confirmed; delivered mail was `EmailResWarn`) and of BUG-005's preconditions, whose failure remains unobserved |
| 1.10 | 2026-07-30 | Claude | Added BUG-008 through BUG-014 from the investigation into intermittent production write hangs, with their relationships; recorded the `resNotifyChan` lock inversion as not tracked, with the reachability analysis that rules it out for now |
| 1.11 | 2026-07-31 | Allen Bagwell, Claude | BUG-008 updated with production measurements: one connection per Arista RPC with no reuse (+10 for 10 calls, +1 for a one-node install), ~60-minute switch-side reclamation, and the resulting rolling-window exposure model. Corrected the earlier reclamation reasoning, which assumed a ~75s keepalive and understated accumulation by roughly fifty-fold |
| 1.12 | 2026-07-31 | Allen Bagwell, Claude | Added BUG-015 (`aristaVlan` unchecked type assertions) and BUG-016 (empty `networkPassword` mangles error text), both demonstrated; recorded the switch TLS limitation under "Not tracked as bugs" with its full account in the new [ISSUE-001](../ISSUE-001.md) |
| 1.13 | 2026-07-31 | Allen Bagwell, Claude | BUG-008 and BUG-016 marked fixed in `934e720`, BUG-010 in `05e94ed`, BUG-013 in `528e6f0`; resolutions and covering tests recorded in each detail document |
| 1.14 | 2026-07-31 | Allen Bagwell, Claude | Recorded the BUG-013 regression found on the testbed and its fix in `6034343`: the bounded wait also returned on a healthy server, exiting the process into a systemd restart loop |
| 1.15 | 2026-08-03 | Allen Bagwell, Claude | Added BUG-017, a permanent write deadlock between image registration and the initrd worker, found while routing every `dbAccess` acquisition through `lockedDbWrite`; recorded its relationships to BUG-012 and BUG-013, and annotated the rejected `resNotifyChan` inversion, which is the same defect on a channel that cannot realistically fill |
| 1.16 | 2026-08-12 | Allen Bagwell, Claude | BUG-017 marked fixed in `1ad2730`, with the resolution and its three covering tests recorded. Noted that the suggested fix as originally written was insufficient — ending the locked region early leaves the enqueue inside the transaction, trading the mutex deadlock for `SQLITE_BUSY`. Updated the `resNotifyChan` rejection to reflect that it remains unfixed and now rests solely on its bounded consumer |
| 1.17 | 2026-08-12 | Allen Bagwell, Claude | Investigated and rejected the apparent `allowImageUpload` gap on `POST /images/register`: the endpoint is admin-only via `authzHandler`'s derived permission, not via its route chain, and code matches documented intent. Corrected BUG-017's reachability claim, which had called the trigger an ordinary user action on the strength of the route chain alone; the register path needs an elevated admin and the user path needs `allowImageUpload: true` |
