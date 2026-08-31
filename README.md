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

Elysia can react to whatever's currently playing on Spotify. There are two
ways this works:

**1. Local detection (default, no setup at all).** If the Spotify desktop
app is installed and running on the same machine, Elygochi reads what's
playing directly from it — no login, no developer account, and critically
**no Spotify Premium required** (Spotify's Web API player-state endpoints
are Premium-only; reading the local app directly isn't). This just works
out of the box the moment you open Spotify.

- macOS: via AppleScript talking to the local `Spotify.app`.
- Windows: by reading the track/artist straight out of the Spotify window's
  title bar (a long-standing, stable behavior of the Windows client).

**2. Web API (optional, only useful if you're on Premium and want Spotify
running on some *other* device — phone, a different computer — to count
too).** This needs a one-time setup, since Spotify requires every app to
register its own "Client ID":

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard),
   log in with your Spotify account, and click **Create app**.
2. Give it any name/description. For **Redirect URI**, enter exactly:
   `http://127.0.0.1:8888/callback`
3. Check the **Web API** checkbox, save.
4. Open the app you just created, copy its **Client ID**.
5. Paste it into the "Spotify Client ID" field in the in-app Settings panel
   (or save it as plain text to
   `~/Library/Application Support/elygochi/spotify_client_id.txt`).
6. The next time the app starts, it'll open your browser straight to
   Spotify's login/approval page — log in, approve, and you can close the
   tab. A refresh token is saved locally so you won't need to do this again.

There isn't a single shared Client ID built into the app — Spotify's
developer terms require each app to register its own, so this step can't be
skipped or done on your behalf. In practice, local detection covers the
common case (Spotify open on the same computer) with zero setup, so most
people never need the Web API path at all.

## Assets

`internal/assets/portraits` and `internal/assets/stickers` hold the source
art (embedded into the binary via `go:embed`). Since the art wasn't drawn
per-mood, `internal/assets/assets.go` maps the closest-fitting images/GIFs to
each mood and tints the ones that don't have a dedicated illustration — edit
the `portraitMoods`/`stickerMoods` tables there to change the mapping.
