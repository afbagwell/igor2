# ISSUE-001 — Igor cannot talk to a VLAN switch over TLS

**Status:** open · **Type:** design limitation (not a defect) · **Component:** core
**Raised:** 2026-07-31 · **Version:** v2.3.2 · **Affects:** any deployment with `vlan.network` set

---

## Why this is an ISSUE and not a BUG

`docs/bug_tracking/` records concrete, reproducible **defects** — code that does something
other than what it was written to do. This is not that. `aristaJSONRPC` transmits over
plain HTTP because it was written to transmit over plain HTTP, and it does so correctly.
Nothing malfunctions.

What is wrong is the *capability*: igor offers deployers no way to encrypt switch
management traffic, and no way to opt out of that. Filing it as a bug would misrepresent
it and dilute the tracker. Leaving it unrecorded would be worse, because the consequences
fall on downstream sites rather than on the project's own instance.

It is recorded in `bug_tracker.md` under "Not tracked as bugs" with a pointer here.

## The limitation

`internal/app/igor-server/network_arista.go:77`:

```go
path := fmt.Sprintf("http://%s:%s@%s", user, password, URL)
```

The scheme is a string literal. `vlan.networkURL` is documented as a host-port-path
fragment (`cmd/igor-server/igor-server.yaml:348-351`):

```
# networkURL (string) - Network service URL.
# Ex: arista.mysite.com:80/command-api
```

There is no configuration key that selects a scheme, and no code path that produces
`https://`. A deployer who wants TLS has no supported way to get it.

## Ramifications

### 1. Igor mandates cleartext for every deployment

This is not a default that a careful administrator can override — it is the only behaviour
available. Any site running igor's VLAN feature is transmitting switch configuration
commands in the clear on its management network, whether or not that is acceptable under
local policy.

For a site with an encryption-in-transit requirement covering network device management,
the practical options are to violate the policy, or to not use VLAN segmentation at all.
Neither should be the position a deployer is put in by a project that otherwise supports
TLS throughout: `igor-server` requires a certificate and key for its own API
(`server.go:77-86`), and `server.CbUseTLS` makes even the node callback listener
configurable. The switch client is the one place where TLS is unreachable.

### 2. Credentials cross the network in cleartext, embedded in the URL

`vlan.networkPassword` is a real configuration key that many sites will populate — it
authenticates an account with authority to reconfigure switch ports. On every reservation
install and teardown, once per host, igor sends it as HTTP Basic in the clear.

A 50-node reservation transmits that credential 50 times on install and 50 more on
teardown. A busy cluster does so thousands of times a day. Anyone positioned on the
management network can capture it and then reconfigure switch ports at will.

The credential is also embedded in the URL rather than set as a header, which is why the
code has to scrub it out of error strings at all — see
[BUG-016](bug_tracking/BUG-016.md), where that scrubbing is itself defective.

### 3. Sites that leave the password blank are exposed differently, not less

Where the switch account has no password — as on the instance this was investigated on —
there is no secret to intercept. The exposure is the reverse: eAPI accepts an unauthenticated
`igor` identity over cleartext HTTP, and that identity can reconfigure VLANs. Anyone who
can reach port 80 on the switch can do everything igor does, without needing to capture
anything at all.

That is a defensible posture on a well-segmented management network, and it is the
deployer's decision to make. The point is that igor gives them no way to make a different
one.

### 4. Reservation topology is observable

The request and response bodies carry interface names, VLAN ids and port assignments. On a
multi-tenant cluster that reveals which nodes belong to which reservation and when
reservations start and end. For sites where the work itself is sensitive, that metadata may
matter more than the credential.

### 5. A naive fix makes things actively worse

This is the ramification most likely to cause harm, and the reason this document exists
rather than a one-line note.

The transport is already configured as though TLS were expected
(`network_arista.go:62-70`):

```go
t := &http.Transport{
    TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
    TLSHandshakeTimeout: time.Second * 5,
    ...
}
```

Both settings are dead code today, because no handshake ever occurs. Change line 77 to
emit `https://` and they come alive immediately — and `InsecureSkipVerify: true`
unconditionally disables certificate verification.

The result would be TLS that encrypts but authenticates nothing: trivially
machine-in-the-middled by anyone who can intercept the traffic, while presenting every
outward sign of being secure. Both the deployer and any auditor would reasonably read
`https://` in the configuration as the problem being solved. It would be a worse outcome
than the honest cleartext of today, because cleartext at least advertises what it is.

Anyone implementing this must treat `InsecureSkipVerify` as part of the change, not as
pre-existing code to leave alone.

## What a fix needs to cover

Sketched rather than specified; this needs the §9 treatment before implementation.

1. **Scheme selection with backward compatibility.** Accept a scheme prefix on
   `vlan.networkURL` and honour it. A value with no scheme must continue to mean `http://`,
   so no existing deployment changes behaviour on upgrade. Note that a deployer who writes
   `https://...` today gets `http://user:pass@https://host/path`, which fails to parse and
   returns an error — loudly, at least, but with no indication that the scheme was the
   problem.
2. **Certificate verification as a deliberate choice.** `InsecureSkipVerify` should become
   a documented configuration key defaulting to verification *on*. Self-signed switch
   certificates are common, so an explicit opt-out is needed — but it must be opt-out, and
   it should log a warning when engaged.
3. **A trust anchor option.** Sites with an internal CA need to supply a bundle rather than
   choosing between full verification against the system store and none at all.
4. **Move credentials out of the URL.** `req.SetBasicAuth(user, password)` against a
   credential-free URL. This is worth doing regardless of scheme, and it removes the need
   for the error-string scrubbing that [BUG-016](bug_tracking/BUG-016.md) documents.
5. **Documentation.** The `networkURL` example in `cmd/igor-server/igor-server.yaml` should
   show both forms, and the security implications of the cleartext form should be stated
   where a deployer will encounter them.

Items 1 and 4 are small. Item 2 is where the care is required.

## Relationships

| Document | Relationship |
|---|---|
| [BUG-008](bug_tracking/BUG-008.md) | `TLSHandshakeTimeout` is dead code for the reason described here, which is why the only bound present in that transport protects nothing against the unbounded `client.Do`. Fixing this issue does not fix BUG-008 — a TLS connection stalls just as indefinitely — and would add per-connection handshake cost, making the absent connection reuse slightly worse. |
| [BUG-016](bug_tracking/BUG-016.md) | The credential-in-URL construction described in item 4 is why that scrubbing code exists at all. Fixing item 4 removes the defective code rather than repairing it. |

## Revision History

| Version | Date | Author | Change |
|---|---|---|---|
| 1.0 | 2026-07-31 | Claude | Initial record of the TLS limitation and its downstream consequences |
