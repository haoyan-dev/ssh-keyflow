# Whole-TUI review resolutions

Date: 2026-11-08 JST.
Reviewer: Astra (`auto/gpt-6-astra`, high effort), delegated read-only review.
Base: `41fe1ea2359f0a7c23ab07209fc2df5a4b3b4d68`.
The [initial audit](astra-review.md) informed the [design](design.md).

## Repository standards and safety

Astra reviewed repository standards separately from specification compliance.
The first implementation review found a misleading Task shortcut: a list
description advertised `v`, which is only a supported technical-document route
from an overview. Task lists now advertise Enter details; overview footers
advertise `v`. A public-event first-viewport check follows the advertised route.

The second review found no new blocking standards or safety violation. It
confirmed retained fixed IDs and revalidation, read-only and stale restrictions,
native approval boundaries, independently labelled registration and connection
evidence, presentation-only picker context, read-only ANSI-stripped instructions,
and retained raw documents with width reflow. No backend/schema capability was
added. `git diff --check` passed.

## Specification compliance

The initial implementation review requested changes. Each finding was resolved
before completion; the second review confirmed the earlier fixes and identified
two further visual identity issues.

| Finding | Resolution and verification |
| --- | --- |
| Generic Task summaries hide actual failure | Integrity reason remains verbatim in overview/technical; list omits only the redundant known review prefix. Typed pending file update/unfinished Include evidence gets explicit summaries. First-viewport checks require the actual missing/unreadable reason. |
| Long duplicate names hide discriminator | Reserve reference suffix width before shortening the human name. CJK duplicate key regression at32×12. |
| Long active value hides the rest of the form | Keep one complete form document, including fixed targets, preceding/following fields, guidance, full value and controls. Vertical-access and delayed cursor-response journeys pass. |
| Resized overlay return hides focus | Compare restored viewport dimensions, resize and reveal focus only when necessary. Review/Edit, Help/back and Discard/Keep editing journeys cover80×24→32×12. Caret models remain intact. |
| Custom page capacity disagrees with keyboard paging | One selection layout determines both visible rows and Bubbles delegate/viewport sizing. PgUp/PgDown and exact chosen ID checks at all three sizes. |
| Unregister/continuation picker loses origin | Separate presentation caption from request IDs. Unregister assertion; source review of continuation/result context. |
| Result promises unsupported continuation | Controls and next-step prose use the same supported-resume predicate. Config-only read-only result journey checks both. |
| Narrow result lacks scroll position | Three-row footer includes focused action, Tab action count, direct instructions and scroll position. Result journeys verify bounded size and document restoration. |
| Result entity history has empty scope | Resolve human label by kind/ID with full ID fallback. Result→entity→history journey verifies registration scope. |
| Reopened Add key picker labels previous key | Give draft destination caption precedence on both initial entry and reopening. Public-event journey at all three sizes. |
| Same-name registrations look identical | Detect colliding destination/key names and reserve a registration reference suffix. Long CJK names, selection and exact registration/key/destination verified at all three sizes. |

## Evidence handoff

Full TUI suite after these changes: `.tmp/tui-readability/test-twelfth.txt`,
PASS in15.606s. The [preview](preview.md) contains actual terminal renders;
54 full captures are available in the ignored local
`.tmp/tui-readability/after/` directory.

Final independent review and exact-source full checks are recorded in
[verification.md](verification.md). Local implementation does not authorize
remote push, merge, tag or publication.
