# Igor Bug Tracker

Concrete, reproducible defects found during code analysis. Hypothetical or
exotic-circumstance concerns are **not** recorded here.

**Status values:** `open` · `triage` (awaiting maintainer decision) · `fixed` (commit ref) · `wontfix`

| ID | Status | Component | Severity | Summary |
|---|---|---|---|---|
| BUG-001 | open | cli | medium | A `%` in the cluster MOTD corrupts `igor show` output for every user |
| BUG-002 | open | core | low | A `%` in the `duration` query param yields a mangled stats error message |
| BUG-003 | open | core, cli | medium | 40 non-constant format-string call sites; blocks `go test ./...` |

---

## BUG-001 — A `%` in the cluster MOTD corrupts `igor show` output

**Status:** open · **Component:** cli · **Severity:** medium
**Found:** 2026-07-29 · **Version:** v2.3.2

### Location

`internal/app/igor-cli/show.go:874,876` in `printMotd`:

```go
if clusterData.MotdUrgent {
    cMotdUrgent.Printf(finalMotd)
} else {
    cMotdNotUrgent.Printf(finalMotd)
}
```

`finalMotd` is runtime data passed as the *format string*.

### Cause

The MOTD is arbitrary admin-supplied free text. `validateMotdParams`
(`internal/app/igor-server/cluster_handlers.go:200-205`) only checks that the value
is a string — there is no restriction on content. When that text reaches
`Printf` as a format string, any `%` in it is interpreted as a verb.

### Reproduction

Set a MOTD containing a percent sign — an entirely natural thing for an admin to write:

```
igor cluster mod -motd "Downtime Saturday, 80% of nodes affected"
```

Then run `igor show`. Verified output:

```
MOTD: Downtime Saturday, 80%!o(MISSING)f nodes affected
```

The `%` and the following character are consumed and replaced with an error verb.

### Impact

Every user running `igor show` sees a corrupted MOTD. Worse for messages where
the mangling eats meaningful text (`"50% capacity"` → `"50%!c(MISSING)apacity"`).
No crash, no data loss — cosmetic but user-visible on a cluster-wide notice, and
silently misinforms users about the very thing the MOTD exists to communicate.

### Fix direction

Use `Print`, not `Printf`, when the string carries no format arguments:
`cMotdUrgent.Print(finalMotd)`. Same for the sibling call at line 876.

---

## BUG-002 — A `%` in the `duration` query param yields a mangled error message

**Status:** open · **Component:** core · **Severity:** low
**Found:** 2026-07-29 · **Version:** v2.3.2

### Location

`internal/app/igor-server/stats.go:81-84`:

```go
msg := fmt.Sprintf("error converting string %v to int", v[0])
logger.Debug().Msgf(msg)
status = http.StatusBadRequest
return stats, status, fmt.Errorf(msg)
```

`msg` already contains interpolated user data, then is used again as a format string.

### Reproduction

Request stats with a non-integer `duration` containing a percent sign, e.g.
`GET /stats?duration=50%25`. `v[0]` is `50%`, so `msg` becomes
`error converting string 50% to int`, and re-formatting it produces:

```
error converting string 50%!t(MISSING)o int
```

That string is both logged and returned to the client as the error.

### Impact

Malformed error message in logs and in the API response. Client-triggerable but
harmless beyond the confusing text. The neighbouring `d < 0` branch
(`stats.go:82-84` sibling at line 67-70) has the same `Msgf`/`Errorf` misuse but
is *not* reachable with a `%`, since `d` is already an `int` — it is covered by
BUG-003 as hygiene, not as a live defect.

### Fix direction

`errors.New(msg)` instead of `fmt.Errorf(msg)`, and `Msg(msg)` instead of
`Msgf(msg)`. Alternatively build the error once with `fmt.Errorf("... %v ...", v[0])`
and log that.

---

## BUG-003 — 40 non-constant format-string call sites block `go test ./...`

**Status:** open · **Component:** core, cli · **Severity:** medium
**Found:** 2026-07-29 · **Version:** v2.3.2

### Symptom

`go test ./...` fails to build `igor-server` and `igor-web` — not from test
failures, but because `go test` runs a default subset of `go vet` that now
includes the printf analyzer's non-constant-format-string diagnostic:

```
FAIL	igor2/internal/app/igor-server [build failed]
FAIL	igor2/internal/app/igor-web [build failed]
```

`go test -vet=off ./...` passes. This means the project rule "all tests must pass
before committing" cannot currently be satisfied with the plain command.

### Cause

The diagnostic is **not** from zerolog. It is Go's own `printf` vet analyzer
flagging Igor call sites that pass a runtime-built string where a format string
is expected. It began firing because the check is gated on the module's declared
language version, and `go.mod` declares `go 1.24.0`.

Affected call targets, all of which the analyzer knows are printf-like:

| Target | Sites |
|---|---|
| `zerolog.Event.Msgf` | most of the server hits |
| `fmt.Errorf` | `authn_ldap.go:31`, `authn_local.go:30`, `stats.go:70,82`, `cluster_create.go:234,250`, `ldap.go:103` |
| `fmt.Printf` | 9 CLI sites (`cluster.go`, `distro.go`, `group.go`, `host.go`, `host_policy.go`, `image.go`, `kickstart.go`, `profile.go`, `reservation.go`, `user.go`) |
| `gookit/color` `Sprintf` / `Style256.Printf` | `logger.go` (server + web), `show.go`, `host.go` |

Two of these sites are live defects in their own right: BUG-001 and BUG-002. The
remainder are latent — they only misbehave if a `%` reaches them.

### Impact

- The test suite cannot be run with the standard command, so CI and the
  pre-commit rule are effectively bypassed or silenced.
- Each site is a latent output-corruption bug of the BUG-001 kind.

### Fix direction

Mechanical, per call site — no rule weakening, no `//nolint`:

- `Msgf(s)` → `Msg(s)`
- `fmt.Errorf(s)` → `errors.New(s)`
- `fmt.Printf(s)` → `fmt.Print(s)`
- `Sprintf(s)` → `Sprint(s)`, `Printf(s)` → `Print(s)`

Where the intent really was formatting, pass the arguments properly rather than
pre-building the string. Verify with `go vet ./...` clean and `go test ./...`
green without `-vet=off`.

---

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

---

## Revision History

| Version | Date | Author | Change |
|---|---|---|---|
| 1.0 | 2026-07-29 | Claude | Initial tracker; BUG-001, BUG-002, BUG-003 |
</content>
</invoke>
