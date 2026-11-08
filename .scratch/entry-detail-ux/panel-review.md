# Local color and panel review

Reviewed 2026-11-08 at `762acdc` before implementation.

The overview's hierarchy is currently conveyed only through bold text and
blank lines. Saved status and actions have the same visual weight as related
rows. Thin panels can clarify those two groups without enclosing every row.
Terminal palette colors can reinforce saved outcomes without inventing a
global connection status or implying a live SSH check.

Fixed-width panels introduce a resize dependency: overview documents are
composed when opened, while Help and technical disclosure retain copies of
their origin viewports. Recompose the visible overview when its viewport width
changes, including restored origins. Keep technical documents as plain wrapped
text and keep all existing action eligibility and confirmation boundaries.

Use existing Lip Gloss and viewport dependencies, with no background fills.
Collapse borders below 60 columns to preserve reading space at 32x12. Keep
explicit status wording so monochrome and terminal palette variations retain
the same information.
