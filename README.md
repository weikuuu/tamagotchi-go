# Elygochi

A desktop tamagotchi built with Go + [Ebitengine](https://ebitengine.org).

- **Main window** — feed, play with, and put your pet to sleep. Its mood
  (`happy`, `content`, `bored`, `sad`, `hungry`, `tired`, `sick`) is derived
  from Hunger/Energy/Happiness/Cleanliness stats that decay over time, and
  drives which Elysia art is shown. There's also an Affection stat that only
  ever goes up, and a "catch the hearts" mini-game.
- **Desktop overlay** — a small transparent, always-on-top window with a
  chibi sticker that drifts around the screen on its own. Click it to pet it
  (small happiness boost), drag it around, or (if flee mode is on in
  Settings) chase it with your cursor.

Both windows launch together from one binary and stay in sync through a
shared save file (`~/Library/Application Support/elygochi/state.json` on
macOS, or the OS equivalent), since Ebitengine only supports one window per
process — the main process spawns a second copy of itself as the overlay.

## Run

```sh
go run ./cmd/elygochi
```

## Optional personal touches

All of these are configurable from the in-app **Settings** panel (click
"Настр." in the top-left of the main window) — no need to edit files by
hand. They're stored as plain text under
`~/Library/Application Support/elygochi/` (created automatically on first
run). None are required — everything works fine without them.

- **Name** — Elysia will occasionally address you by it.
- **Birthday** — as `ДД-ММ` (day-month, e.g. `14-03`). Elysia has special
  lines for that day (and for New Year's, automatically).
- **Spotify** — see below.

### Spotify ("what are we listening to?")

Elysia reacts to whatever's currently playing in the local Spotify desktop
app — **no login, no developer account, no Client ID, no setup at all.**
This just works the moment Spotify is open, on any account tier (Free or
Premium).

- macOS: via AppleScript talking to the local `Spotify.app`. The first time,
  macOS may show a one-time **"Elygochi wants to control Spotify"** prompt
  (System Settings → Privacy & Security → Automation) — this has to be
  allowed for it to work. If it doesn't seem to be picking anything up and
  you don't remember seeing that prompt, check that section of System
  Settings directly; if Elygochi is listed there but *unchecked* for
  Spotify, turn it on. If it's not listed at all yet, quit and reopen the
  app to trigger the prompt again.
- Windows: by reading the track/artist straight out of the Spotify window's
  title bar.

There's no fallback beyond this — if Spotify isn't running locally (e.g.
you're listening on a phone instead), the "now playing" pill just doesn't
show up.

## Assets

`internal/assets/portraits` and `internal/assets/stickers` hold the source
art (embedded into the binary via `go:embed`). Since the art wasn't drawn
per-mood, `internal/assets/assets.go` maps the closest-fitting images/GIFs to
each mood and tints the ones that don't have a dedicated illustration — edit
the `portraitMoods`/`stickerMoods` tables there to change the mapping.
