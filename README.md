# Elysia Tamagotchi

A desktop tamagotchi built with Go + [Ebitengine](https://ebitengine.org).

- **Main window** — feed, play with, and put your pet to sleep. Its mood
  (`happy`, `content`, `bored`, `sad`, `hungry`, `tired`, `sick`) is derived
  from Hunger/Energy/Happiness stats that decay over time, and drives which
  Elysia art is shown.
- **Desktop overlay** — a small transparent, always-on-top window with a
  chibi sticker that drifts around the screen on its own. Click it to pet it
  (small happiness boost), or drag it around.

Both windows launch together from one binary and stay in sync through a
shared save file (`~/Library/Application Support/tamagotchi/state.json` on
macOS, or the OS equivalent), since Ebitengine only supports one window per
process — the main process spawns a second copy of itself as the overlay.

## Run

```sh
go run ./cmd/tamagotchi
```

## Optional personal touches

These are all plain text files you create by hand in
`~/Library/Application Support/tamagotchi/` (created automatically on first
run). None are required — everything works fine without them.

- **`username.txt`** — your name, one line. Elysia will occasionally
  address you by it.
- **`birthday.txt`** — your birthday as `MM-DD` (e.g. `03-14`). Elysia has
  special lines for that day (and for New Year's, automatically).
- **`spotify_client_id.txt`** — see below.

### Spotify ("what are we listening to?")

Elysia can react to whatever's currently playing on your Spotify account —
desktop app, web player, phone, doesn't matter — via the official Spotify
Web API. One-time setup:

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard),
   log in, and click **Create app**.
2. Give it any name/description. For **Redirect URI**, enter exactly:
   `http://127.0.0.1:8888/callback`
3. Check the **Web API** checkbox, save.
4. Open the app, copy its **Client ID**.
5. Save that Client ID as plain text to
   `~/Library/Application Support/tamagotchi/spotify_client_id.txt`.
6. Run the app. The first time, it'll open your browser to ask you to
   approve access — approve it, and you can close the tab. A refresh token
   gets saved locally (`spotify_refresh_token.txt`) so you won't need to do
   this again.

Without a `spotify_client_id.txt`, this feature just stays off.

## Assets

`internal/assets/portraits` and `internal/assets/stickers` hold the source
art (embedded into the binary via `go:embed`). Since the art wasn't drawn
per-mood, `internal/assets/assets.go` maps the closest-fitting images/GIFs to
each mood and tints the ones that don't have a dedicated illustration — edit
the `portraitMoods`/`stickerMoods` tables there to change the mapping.
