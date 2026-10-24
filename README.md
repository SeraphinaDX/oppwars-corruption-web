# OPP Wars: Corruption — Web Edition v0.7.0

A playable, dependency-free Go browser adaptation of the [original OPP Wars](https://git.cerberusgames.ca/Starstreak/oppwars) terminal game. GPL-3.0-only, consistent with the original repository.

## Run

Requires Go **1.23 or newer**. In this standalone fork, run from the repository root:

```sh
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

## What's in v0.7.0

- Responsive neon BBS-style dashboard, without dependencies or external assets.
- Three original classes and abilities: Robot (Charged Shot), Psychic (Psychic Blast), Psychopath (Chow Down).
- Source-aligned starting stats: $5,000, 200 HP, 40 attack, 30 days, 25 patrols/day.
- Original-like combat, balanced rewards, XP thresholds, 1-in-6 random events, and Paragon/Renegade choices.
- Property room with activated temporary effects, no passive inventory buffs; medical treatment; night-life recovery; daily economy.
- Per-browser-session JSON saved automatically to OS user configuration directory (`oppwars-web/saves`).
- Server-side authority for game actions (the browser doesn't award itself money/XP).

## v0.7.0 — arrests, hand cuffs, and combat karma

- The Property Room now sells **hand cuffs** at **$100 per pair**. Up to 20 may be stored and they persist across shifts and in older saved games (which begin with zero cuffs). Renegade rank 10+ keeps the existing two-for-one Property Room quantity perk.
- Combat now has **Fight**, **Class Special**, **Arrest**, and **Return to HQ** options.
- **Arrest** becomes available with at least one pair of hand cuffs when the enemy is **alive and at or below half HP**. A successful arrest consumes one pair, ends that encounter, awards **+1 Paragon**, and gives **half the normal victory cash and XP** (including XP Rush bonus).
- Eliminating an opponent through Fight or a class special still grants the normal full rewards, but also awards **+1 Renegade**. Neither combat outcome reduces the opposing karma rank.
- Failed arrest attempts due to health threshold or missing cuffs consume **no items** and give **no rewards**. Retreat and defeat award neither type of combat karma.
- Arrests complete a patrol slot and decrement temporary effect durations exactly like normal victories. XP promotions and Paragon/Renegade event choices are unchanged.
- Added save-compatible inventory, combat UI hints, and regression tests for purchases, rewards, karma, levels, boundaries, and persistence.
- **Version:** v0.7.0 is read dynamically from the server; older save files continue to work.

## v0.6.2 — defeat ends your shift

- Losing a fight now forces an early clock-out instead of instantly ending the entire campaign. Department ambushes follow the same rule.
- The defeated officer gives up remaining patrol assignments and receives **no shift income** from the forfeited day. There is **no cash or XP reward** for the lost fight.
- After hospital recovery, officers return on the following day with **half their maximum HP** (minimum 1 HP). Unused drug effects are cleared, Robot charge is reset, and the Psychic meter refills for the new shift.
- Untreated infections and previous inventory remain intact. The next day's patrol limit still respects infection penalties.
- Losing a fight on the campaign's final day completes the 30-day tour normally instead of starting a nonexistent next shift.
- Combat log reports the defeated officer and opponent plus the forced clock-out penalty; no changes to the saved-game format.

## v0.6.1 — enemy health bar layout fix

- Enemy name, HP readout, and HP bar now appear together, **above** the enemy's catchphrase in combat.
- Expose enemy HP as an accessible progress bar while retaining the existing colours and dialogue.
- No changes to combat calculations or save-file format.

## v0.6.0 — Red Room encounter results

- Choosing **Bang a Prostitute** now opens a dedicated non-graphic story screen rather than returning immediately to Headquarters.
- Each visit draws from several in-character, randomized Red Room scenes.
- An encounter receipt explicitly reports whether the officer **acquired a new infection**, whether a condom was used, current infection count, recovered health, and cash spent. The game tracks infection counts rather than named diseases.
- **Continue** acknowledges the outcome and returns to the main patrol screen. While this receipt is open, no other HQ actions may be performed.
- The server saves the result phase and report in the player's session: reloading or restarting the Go server will not skip the screen, charge again, or reroll the infection.
- Existing Red Room costs, infection probabilities, protection and healing are unchanged. Previous saved games remain compatible.
- Release version incremented in `version.go`; the browser receives it from the server.

## v0.5.0 — Red Room protection and infection consequences

- **Condoms:** five now cost $250 ($50 each), instead of $10. A condom still protects the paid Red Room encounter against infections; encounters cost $100, and the 200 HP recovery is unchanged.
- **Combat illness:** each untreated infection reduces damage from normal attacks and all three class specials by 15%, up to a 45% maximum. These are fictional RPG debuffs, not medical claims.
- **Income:** every infection deducts $150 each new shift (previously $25). Net shift income is never less than $100 so a struggling character can recover.
- **Patrol slots:** an infected officer gets 25 − 15 − 3 × infections assignments next shift, with a minimum of 5. Untreated infections remain until cured.
- **Clinic:** treating all infections costs $500 per infection (previously $100 total), with a clear breakdown in the Medical Wing.
- UI exposes the current penalties and exact cure price; Red Room descriptions stay in character rather than discussing probability rolls. Old save games remain compatible.
- **Version:** incremented server and browser release to v0.5.0.

## v0.4.0 — in-character combat and Red Room

- The Red Room's paid encounter is now labelled **Bang a prostitute**. The action keeps its existing $100 cost, recovery, condom consumption and infection-risk mechanics.
- Red Room flavour text is written as in-world dialogue instead of explaining probability rolls or differences from the terminal edition. Club investments continue to work the same way.
- Combat messages name the officer and their opponent, including hits, misses, dodges, specials, retreats, promotions and victory rewards.
- Each opponent has individual random catchphrases, shown in the combat panel and dispatch log. Enemies occasionally change what they shout during longer fights.
- Catchphrases are included in saves. Older saves without this optional field continue to load.
- The displayed browser version comes from the server's `version.go` constant, now **v0.4.0**.

## v0.3.0 — class specials, pricing, and visible version numbers

- **Robot — Charged Shot:** uses the former charge mechanic. Charge builds to a maximum 4× multiplier, and resets after firing. A normal attack builds charge; firing the special no longer accidentally recharges immediately.
- **Psychic — Psychic Blast:** always does 2× normal attack damage. A 100-PSI meter pays 25 PSI per blast (four uses per shift); the meter refills at clock-out and depleted specials are disabled.
- **Psychopath — Chow Down:** retains damage and healing equal to half the damage dealt.
- **Property Room prices:** cocaine $400 per unit, fentanyl $150 per unit, and LSD $75 per tab. Each purchase provides one unit; Renegade rank 10+ doubles that quantity at the same price, subject to the 20-unit limit.
- Effect strengths and encounter durations remain unchanged. Reduced item costs align them with the early-game $300 victory reward, rather than forcing players to purchase upgrades that barely repay their cost.
- **Version display:** the server exposes the canonical version from `version.go` and the browser reads it dynamically. New and existing sessions show v0.3.0.
- **For every new release:** bump `gameVersion` in `version.go`, update the README heading and changelog, and run the tests before tagging.

## v0.2.0 balance and LSD

- **LSD (Reality Distortion):** activate one tab at HQ for 2 completed patrol encounters; 25% chance to dodge a counterattack on a missed attack, 20% chance that a hit (including class specials) does half damage. During activation the skyline changes colour; animation honours reduced-motion settings.
- **Cocaine (XP Rush):** one unit grants +25% XP on victory (capped at +150) for 3 encounters.
- **Fentanyl (Reckless Strength):** one unit grants +25% normal attack damage, but enemy counterattacks inflict +5 HP, for 2 encounters.
- Active effects do not stack, cannot be refreshed early, and end after their encounter count is used or when clocking out. Retreat counts as an encounter. Inventory max is 20 of each.
- Money awarded per victory is **$300 + $75 per level above 1**, without multipliers on existing cash. Club shares max at 5, healing is capped by level (200 + 40 HP per level above 1; hard max 720), and Psychic charge caps at 4.
- Old v0.1.0 and v0.2.0 JSON saves load normally. Quantities from legacy saves above shop caps remain in inventory until spent; you cannot buy beyond the cap.

These are fictional game mechanics and aren't intended as depictions of real drug effects.

## Scope and compatibility

This is **a reconstructed adaptation**, not yet shared game-logic code. The current upstream original is a single terminal-oriented `main.go` with menus and game mechanics interleaved; we avoided touching it so the classic game keeps building normally. Once this is merged, the next step should extract a shared `game/` package so both front ends use exactly the same rules.

Differences in this first edition:

- Adult material is presented through non-graphic paid adult encounters. Instead of buying people, the Red Room has venue investments with the same $2,000 cost and $500/day revenue model.
- The original game's exploitative reproductive/child-trading scenario is not included.
- Combat returns to HQ after victory rather than immediately opening another encounter; use Hit the Streets to continue.
- Original encounter descriptions and random event text are paraphrased, not copied.
- No accounts, multiplayer, online leaderboards, public deployment, or terminal save import yet.

## Tests

```sh
go test ./...
go test -race ./...
```

The source contains no third-party packages. Tests use the standard Go tooling.

## License

GPL-3.0-only, the same license as OPP Wars. When installing this as `web/` in the original repository, its root `LICENSE` applies. No source code from the original is replaced.
