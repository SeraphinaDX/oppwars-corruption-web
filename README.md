# OPP Wars: Corruption — Web Edition v0.1.0

A playable, dependency-free Go browser adaptation of the [original OPP Wars](https://git.cerberusgames.ca/Starstreak/oppwars) terminal game. GPL-3.0-only, consistent with the original repository.

## Run

Requires Go **1.23 or newer**. In your existing OPP Wars checkout, place this `web/` directory alongside `main.go`:

```sh
cd web
go run .
```

Open **http://127.0.0.1:8080** in your browser. To build a standalone executable:

```sh
go build -o oppwars-web .
./oppwars-web -listen=127.0.0.1:8080
```

Windows: `go build -o oppwars-web.exe .` and then `oppwars-web.exe`.

Custom listen address and save location:

```sh
./oppwars-web -listen=127.0.0.1:8090 -save-dir=/path/to/saves
```

**Security note:** Default binding is localhost. This is a local single-player game, **not** a ready-to-publish public multiuser service. Don't expose the server to the public internet without TLS, authentication, rate limiting and hardening. Browser sessions are identified by unguessable HttpOnly cookies, and save data is stored by the server. Clearing cookies loses access to an existing save; back up the saves folder if needed.

## What's in v0.1.0

- Responsive neon BBS-style dashboard, without dependencies or external assets.
- Three original classes and abilities: Robot (Charged Shot), Psychic (Psychic Blast), Psychopath (Chow Down).
- Source-aligned starting stats: $5,000, 200 HP, 40 attack, 30 days, 25 patrols/day.
- Original-like combat, rewards, XP thresholds, 1-in-6 random events, and Paragon/Renegade choices.
- Property room with passive drug buffs and purchases; medical treatment; night-life recovery; daily economy.
- Per-browser-session JSON saved automatically to OS user configuration directory (`oppwars-web/saves`).
- Server-side authority for game actions (the browser doesn't award itself money/XP).

## Scope and compatibility

This is **a reconstructed adaptation**, not yet shared game-logic code. The current upstream original is a single terminal-oriented `main.go` with menus and game mechanics interleaved; we avoided touching it so the classic game keeps building normally. Once this is merged, the next step should extract a shared `game/` package so both front ends use exactly the same rules.

Differences in this first edition:

- Adult material is presented as non-graphic nightlife. Instead of buying people, the Red Room has venue investments with the same $2,000 cost and $500/day revenue model.
- The original game's exploitative reproductive/child-trading scenario is not included.
- Combat returns to HQ after victory rather than immediately opening another encounter; use Hit the Streets to continue.
- Original encounter descriptions and random event text are paraphrased, not copied.
- No accounts, multiplayer, online leaderboards, public deployment, or terminal save import yet.

## Tests

```sh
cd web
go test ./...
go test -race ./...
```

The source contains no third-party packages. Tests use the standard Go tooling.

## License

GPL-3.0-only, the same license as OPP Wars. When installing this as `web/` in the original repository, its root `LICENSE` applies. No source code from the original is replaced.
