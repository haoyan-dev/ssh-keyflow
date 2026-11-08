# Local TUI redesign preview

Generated from the production NewProgram entry through keyboard events on 2026-11-08. These are actual ANSI-stripped terminal renders using synthetic data; trailing padding is removed. Color is supplemental. See [design](design.md), [specification](spec.md), and [review resolutions](review-resolutions.md).

## destinations-80x24

```text
SSH KeyFlow
[Destinations] | Keys | Registrations | Tasks
Managed destinations  1/9

> build-1
  alice@server-1.example:22 · linux

  build-2
  alice@server-2.example:22 · linux

  build-3
  alice@server-3.example:22 · linux

  build-4
  alice@server-4.example:22 · linux

  build-5
  alice@server-5.example:22 · linux



Requires review (g: details)
a add · r register · u unregister
↑/↓ select · Enter details · / search · Tab category · ? help · q quit
```

## picker-32x12

```text
SSH KeyFlow
Register > Choose key
Destination: build-1
Managed keys  1/9
> Laptop [key-1]
  public only · SHA256:AQAAAA…
  Laptop [key-2]
  public only · SHA256:AgAAAA…
  Laptop [key-3]
  public only · SHA256:AwAAAA…
enter choose · / search
? help · esc back
```

## register-80x24

```text
SSH KeyFlow
Prepare · Register public key
Register public key

> Destination
  build-1 · alice@server-1.example:22 [host-1]

Managed key: Laptop [key-1] · public only
Auth: manual/password/key (←/→): manual
SSH configuration: yes/no (←/→): no
  [Continue]
  [Help]
  [Cancel]

Field 1/7







tab/shift+tab move · enter next · ctrl+s review · esc back
PgUp/PgDn scroll · Help control
```

## add-os-60x18

```text
SSH KeyFlow
Prepare · Add destination
Destination · step 1/3

Destination alias: 作業-box
Host / IP: server.example
Login user: alice
Port: 22
> OS: linux/macos/windows
  linux

  [Continue]
  [Help]
  [Cancel]

Field 5/8
tab/shift+tab move · enter next · ctrl+s review · esc back
PgUp/PgDn scroll · Help control
```

## review-32x12

```text
SSH KeyFlow
Review · Register public key
Review register
No changes yet.

Target
build-1 · alice@server-
1.example:22 [host-1]
Laptop [key-1]
[Continue] Edit Cancel Help
Tab choose · Enter select
e/Esc edit · ? Help
```

## destination-32x12

```text
SSH KeyFlow
[Destinations] (tab: switch)
Destination: build-1
alice@server-1.example:22
Saved status
Key: Laptop [key-1]
Registration: registered
Connection: failed at last
check
v technical · F5 refresh
↑/↓ scroll 0% · ? help
g review · esc back · q quit
```

## registrations-32x12

```text
SSH KeyFlow
[Registrations] (tab: switch)
Registrations  1/9
> build-1 / Laptop
  Saved: registered
  Connection: failed at last …


Requires review (g: details)
Enter details · / search
r register · u unregister
Tab category · ? help · q quit
```

## result-32x12

```text
SSH KeyFlow
Operation result
fixture · me@example.test:2222
Handoff fixture

Next steps
Next: Continue saved work or
View task; reconcile before
remote retry.
enter: Continue saved work
o instructions · Tab 1/8
↑/↓ 0% · ? help · esc back
```

## instructions-32x12

```text
SSH KeyFlow
Instructions / diagnostics
Diagnostics / instructions
Session output; not saved
after exit.

Register
SHA256:RXm/ruZ0eTzRXKwi1AQEDyn
B0VgHQ2ac9KPSFdf/YnA at
fixture (me@example.test:2222)
↑/↓ scroll 0% · ? help
esc back · q quit
```

## discard-32x12

```text
SSH KeyFlow
Unsaved draft · esc keep editing
Discard changes?
Create Ed25519 key
Unsaved inputs will be lost.

> Keep editing
  Discard draft

tab/↑/↓ choose · enter select ·
esc keep editing
? help
```

## history-60x18

```text
SSH KeyFlow
History · read-only
History (read-only)
Scope: All recorded registration events
Recorded evidence, not current remote state.

No recorded events in this view.
No actions run from history.









↑/↓ scroll all · ? help · esc back · q quit
```

## help-60x18

```text
SSH KeyFlow
Help · esc back
Help: Destinations

Actions here
c Create key · i Import key · a Add destination
r Register: choose a managed key
u Unregister: choose a current eligible registration
s Continue supported saved registration intent; otherwise
inspect Tasks. Tasks uses saved evidence only.

Move and find
/ Search names, login, fingerprints and IDs
Search: enter accepts; esc keeps query
x Clear search
tab/shift+tab Category (lists/entities)
enter Overview / choose related target
↑/↓ scroll 0% · esc/? back · ctrl+c back
```
