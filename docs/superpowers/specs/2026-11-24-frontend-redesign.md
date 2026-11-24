# passenger-go: frontend redesign, on collage plugins, skeuomorphic

Date: 2026-11-24 · Branch: `collage-rewrite`

## Why

The user found the UI bad and asked to rewrite the frontend from scratch:
built on collage's own plugins instead of hand-written equivalents, laid out
by collage's conventions (`AGENTS.md`, written by `collage ai init`), and with
a cleaner, skeuomorphic look.

## Scope

In: everything a browser sees and every page, fragment, layout, action,
guard, middleware, template, stylesheet and script. Out (unchanged):
`internal/vault`, `internal/store`, `internal/legacy`, `internal/strength`,
`internal/generator`, `internal/transfer`, `internal/config`, the session
manager's DEK storage, the security model in
`docs/superpowers/specs/2026-11-24-collage-rewrite-design.md`.

Behaviour that must survive (the existing tests pin most of it; a test may be
rewritten for new markup, never weakened):
- every route guarded as today; own-URL actions carry their own guard;
- CSRF on every form and fetch; no inline script or style (CSP stays strict);
- the per-IP brute-force protection on every secret-checking form, and the
  Argon2 concurrency cap;
- the wrap mutex on passphrase/recovery changes; DEK copies wiped;
- the session-expiry warning, extend and redirect with server-relative time;
- a missing/stale session goes to plain `/login` (`next` only off the home
  page); no `expired` URL parameter;
- copy buttons never change size; keyboard navigation works page-wide;
- the account passphrase is echoed on a refused account form; master
  passphrase and recovery key never are.

## Plugins (decided)

| Need | Plugin | Replaces |
|---|---|---|
| security headers + CSP | `elagoht/secure` (CSP = today's string) | `internal/web/headers.go` |
| one-time messages after a redirect | `elagoht/flash` | `?saved=1` and similar query flags |
| form validation, values and errors back in the form | `elagoht/validate`, checks in `validations/<area>.go` | if-chains in actions |
| cookie, guards for signed-in / guest | `elagoht/session` (hybrid, below) | `internal/session` cookie code, `RequireSession`/`RequireGuest` |
| request limits on secret forms | `elagoht/ratelimit` rules on `/login`, `/recover`, `/setup`, `/export`, `/settings` | the request-limiting half of `internal/ratelimit` |
| bans for brute force and probes | `elagoht/fail2ban` | — |
| production basics | `elagoht/health`, `elagoht/compress`, `elagoht/accesslog` | — |
| redirect target check | `collage.SafeRedirect` | `web.SafeNext` |

Keep our own code only where no plugin fits, and say why in a comment:
- **The DEK store.** `elagoht/session` keeps its data in a cookie; the DEK must
  never leave server memory. Hybrid: the plugin owns the cookie, its
  lifetime (`IdleTimeout` = `SESSION_IDLE`, `MaxAge` = `SESSION_MAX`),
  `Regenerate` on sign-in and the guards; the session holds only an opaque
  random handle under `session.UserKey`; our manager maps handle → DEK with
  the same idle/absolute limits and wipes on expiry, sign-out, rotation.
  `Encrypt` on, so the handle is not readable in the cookie.
- **Failure-counted backoff and the Argon2 semaphore**, if `ratelimit` +
  `fail2ban` cannot express "count failures, not requests"; check
  `fail2ban`'s jails first (a strike API would cover it).
- **The session-expiry banner** script (no plugin does it).

If a plugin's behaviour conflicts with a "must survive" item, keep the item
and document the gap; do not drop the plugin silently either — say so in the
report.

## Structure (collage conventions, `AGENTS.md`)

- Pages in `pages/<area>/`, content fragments in `fragments/pages/<area>/`
  with **inline templates** (`collage.NewInlineFragment` + `collage.InlineHTML`
  const); layouts in `fragments/layouts/` + `templates/layouts/`.
- Actions one per file in `actions/<area>/`, handler under its builder.
- Form checks in `validations/<area>.go`, one function per form.
- Guards in `guards/`, middleware in `middleware/`.
- Files are created with `collage add …` where the CLI supports it
  (`collage` binary: build it from the collage source in the scratchpad
  if it is not on PATH).
- `collage check` and `go test ./...` pass at the end.

## Visual design: a travel-document wallet

The name is Passenger; the vault is a leather travel wallet and each account
is a boarding pass kept in it. Skeuomorphic means real materials, light and
depth — done in CSS only (gradients, layered shadows, an SVG noise texture as
a `data:` URI, which the CSP allows for images), no raster images besides the
existing logo.

Materials and palette:
- **Leather** (page background): deep navy-blue leather `#1d2a44` with a fine
  grain (feTurbulence noise, low opacity) and a soft vignette; the content
  area is bordered by a **stitched seam** (dashed thread line `#c9a86a`, a
  shadowed groove beside it).
- **Paper** (boarding passes, forms): warm-white card stock `#f6f3ea` with a
  faint fibre texture, a 1px darker edge, and a lifted shadow; text in
  ink `#1f2a3a`.
- **Brass / gold foil** (wordmark, primary buttons, keys): `#c9a86a` →
  `#a8843f` gradient with a top highlight; foil text uses a subtle
  embossed text-shadow.
- **Ink stamp** (feedback): stamp red `#b33a3a` and stamp blue `#2f4f8f`.
- Status: strength weak `#b33a3a`, fair `#c98a1b`, strong `#2f7d55`.

Type (local fonts only — CSP has no external fonts):
- Wordmark and pass headings: a condensed signage face
  (`"DIN Condensed", "DIN Alternate", "Bahnschrift", system-ui`).
- Body: `system-ui`.
- Pass data (usernames, codes, keys): `ui-monospace` — boarding passes print
  their data in mono.
- Labels on passes are small sentence-case words in ink-2, not all caps.

Components:
- **Boarding pass (account row).** Paper card, left part: a coloured carrier
  band on top (hue derived from the platform name, computed server-side, used
  as a CSS custom property set via a class from a fixed palette of ~8 hues —
  no inline style), the initials in a round "carrier logo" badge, platform as
  the flight name, username in mono, strength as a three-segment "class"
  meter. Right part: the **stub**, separated by a perforation (dotted line
  with two half-circle notches cut out of the paper, showing leather through),
  holding *Copy password* (brass button), *Copy username*, *Open site*.
  Selected pass (keyboard): lifted further with a brass outline.
- **Copy feedback: a rubber stamp.** Copying stamps "Copied" in stamp ink,
  rotated ~-8°, onto the pass (absolutely positioned over the pass, so no
  layout shift), fading after ~1.2 s. This is the one orchestrated motion;
  respect `prefers-reduced-motion` (show it without the press animation).
  The toast for screen readers stays (visually hidden `aria-live`).
- **Buttons.** Tactile: raised gradient, 1px top highlight, bottom shadow;
  `:active` presses in (inset shadow, 1px down). Brass for primary, paper for
  secondary, stamp-red text for destructive.
- **Search.** A recessed brass-rimmed slot (inset shadow) at the top of the
  wallet, wide, with the `/` hint engraved.
- **Account form.** A luggage tag: paper tag with a reinforced eyelet and a
  string, fields as handwritten-looking lines (underlines, not boxes) — keep
  them clearly inputs (focus ring). The password sits in a darker "secure
  pocket" panel (leather inset) with Show / Copy / Generate.
- **Auth pages (login, setup, recover).** A closed passport: navy leather
  cover, gold-foil embossed "Passenger" and the key emblem, the form on the
  inner paper page with a faint guilloche pattern (repeating radial
  gradients). The recovery key is printed on a perforated ticket with a copy
  button.
- **Session banner.** A paper notice slid out from the bottom edge of the
  wallet, with a brass *Stay signed in* button.
- **Flash messages.** Small paper slips pinned at the top of the content.
- **Empty vault.** An open, empty wallet pocket with one line and the two
  actions.

Quality floor: responsive to 360 px (passes stack, stub below a horizontal
perforation), visible focus (brass ring), `prefers-reduced-motion`, contrast
AA for text on paper and on leather, no layout shift on copy.

## Verification

- `go test ./...`, `go vet ./...`, `collage check`.
- Screenshots via headless Chrome of every page at 1280 px and 390 px (inside
  an iframe of that width — headless Chrome has a minimum window width).
- The CDP keyboard script (`scratchpad/cdp/kb.mjs`) still passes: focus on
  load, ↓↓↑, click-then-↓, Enter copies the right password, `/` + type +
  Enter, Shift+Enter copies the username, widths stable.
