# Entry detail UX review

Reviewed: 2026-11-08, base `76d392826f757e8aa4cfdb72d2d7b5216660f0fc`.
Scope: the TUI pages opened with Enter from Destinations, Keys,
Registrations and Tasks, including entry links from results.

## Findings

- F01: `navigation.go:entityDetails` presents names, fixed IDs, fingerprints,
  file paths, provenance and operational evidence as equally weighted lines.
  The initial viewport does not explain what the entry is or what to do next.
- F02: Destination and key pages embed complete registration detail documents.
  Repeated identity, configuration and history text makes relationships hard
  to scan, especially with several registrations.
- F03: Raw values such as `unverified`, operation stages and missing timestamps
  require interpretation. Public-only keys and destination-specific connection
  evidence need explicit explanations.
- F04: Every entity appends shared configuration digests and Include edits.
  These global records are useful for diagnosis but distract from the entry
  and can look like evidence that its alias is active.
- F05: The generic detail footer advertises local deletion on destinations and
  registrations, hides saved continuation, and loses action hints in compact
  windows. Task details expose source IDs before explaining the task.

## Direction and review against the request

Use the existing terminal theme, bold titles/section headings, blank-line
separation and left alignment. Terminal font and colors belong to the user's
terminal; no web typography, new palette or decorative panels are needed.
The distinctive information structure is the destination/key relationship:
identity first, saved registration and connection facts together, configuration
separate, supported next actions explicit.

```text
Destination: build
alice@server.example:22 · Linux

Saved status
  concise counts / what was last checked

Next actions
  contextual shortcuts and blockers

Registered keys
  key name + fingerprint hint
  registration / connection evidence

v Technical details     h History
```

`v` opens a read-only technical document and Esc restores the exact overview
viewport. Full values are never truncated there. This avoids replacing the
flat dump with decorative boxes or repeating the same diagnostic document
under each relationship. Existing scroll, help, category and action semantics
remain authoritative.

## Dependencies

Review -> specification -> ticket -> implementation -> verification.
Implementation touches shared detail composition and navigation, so keep it
in one branch rather than parallel edits to the same TUI files. Independent
Go test/vet checks can run concurrently once the final code is stable.
