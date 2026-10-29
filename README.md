# OPP Wars: Corruption — Web Edition v0.10.0

A playable, dependency-free Go browser adaptation of the [original OPP Wars](https://git.cerberusgames.ca/Starstreak/oppwars) terminal game. GPL-3.0-only, consistent with the original repository.

![Screenshot](screenshot.avif)

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

**Multiplayer security:** This is a *first-stage multiplayer beta*, not yet a fully hardened public service. Account registration and login use salted Argon2id password hashes; sessions are random tokens stored as hashes in SQLite. Guest saves still work without accounts. Host the backend on loopback behind HTTPS Caddy, use `-secure-cookies`, back up the `-save-dir` including `precinct.sqlite` and account JSON saves, and monitor public access. Basic per-account login throttling is not a substitute for edge rate limits, account recovery, abuse controls, moderation, or an operations audit. Never expose port 8080 directly to the internet.

### Caddy deployment

Point a domain's DNS record to the server. Sample Caddyfile (replace the domain):

```caddyfile
oppwars.example.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:8080
}
```

Run the game server **on the same host as Caddy**, keeping the Go port private:

```sh
go build -o oppwars-web .
./oppwars-web -listen=127.0.0.1:8080 -secure-cookies
```

Caddy handles HTTPS and forwards the original Host header, which lets same-origin API protection continue to work. The `-secure-cookies` flag is important because TLS terminates at Caddy, so the Go HTTP request itself is not marked HTTPS. Do not configure permissive CORS or forward untrusted client-provided authentication headers. When Caddy runs on a different host, use a private link and firewall the backend instead of exposing it.

### What you can do online in v0.9.0

Create a new account from the **Account** menu, or register after starting as a guest to attach your existing character. Use the username/password from another browser to resume the same campaign. Registered officers appear in **Online Precinct**, with an activity-based online indicator (last five minutes), public class/level/XP/karma profiles, and an XP leaderboard. Chat, mugger contracts, PvP, and co-op patrols are planned later. Real-world daily resets are implemented in v0.10.0.

Only one Go process should write to the `-save-dir` at a time. SQLite stores credentials, account sessions and public profile indexes; per-account JSON save files still store full game state. Sessions expire after 30 days, sign-out invalidates the current browser, and no email/password recovery is included yet. If an officer is currently signed out, their online indicator turns off after five minutes. Account usernames are unique and case-insensitive.

## What's in v0.10.0

- Responsive neon BBS-style dashboard, without dependencies or external assets.
- Three original classes and abilities: Robot (Charged Shot), Psychic (Psychic Blast), Psychopath (Chow Down).
- Source-aligned starting stats: $5,000, 200 HP, 40 attack, 30 days, 25 patrols/day.
- Original-like combat, balanced rewards, XP thresholds, 1-in-6 random events, and Paragon/Renegade choices.
- Property room with activated temporary effects, no passive inventory buffs; medical treatment; night-life recovery; daily economy.
- Per-browser-session JSON saved automatically to OS user configuration directory (`oppwars-web/saves`).
- Server-side authority for game actions (the browser doesn't award itself money/XP).

## v0.10.0 — 30 real-life UTC campaign days

- Each new officer's campaign lasts **30 UTC calendar dates**, starting on the UTC date the officer is created. The 30th day remains playable until the next midnight; the campaign completes at the beginning of day 31.
- **25 patrol turns** (or the existing infection-reduced amount) become available at **00:00 UTC** every real-world day. Officers cannot manually advance time or bank unspent turns.
- **Clock Out** ends today's shift and awards its net wages exactly once; it cannot be used to jump to the next day. The officer is locked out of further game actions until midnight UTC.
- Being defeated still forcibly ends the shift and restores HP to half the maximum, but **does not grant wages**. The campaign continues on the next real UTC date; even a day-30 defeat finishes only when that date expires.
- Missing days causes calendar time to pass regardless of player activity. Unspent turns disappear and missed wages are not automatically awarded. Temporary buffs and unresolved encounters expire at the new UTC date; Psychic energy refills only then.
- A Red Room visit still consumes one of the day's patrol turns. All other prices, experience, items, Paragon/Renegade and combat rules are unchanged.
- The sidebar displays a **live countdown** to the next midnight UTC. Once the date changes, the browser refreshes the game state from the server. Server time is authoritative.
- Prior v0.9 and earlier active saves adopt the real-world calendar on their first request, preserving the remaining days, current turns, cash, items, and progression. For example, a character with 17 days remaining keeps 17 real calendar dates.
- Online accounts and guest saves both follow the same rules. The account and online precinct screens remain available after clocking out.
- To avoid seasonal abuse, new campaigns restart from day 1 and replace the current officer as before. Permanent career records and seasons are a later milestone.

## v0.9.0 — shared online precinct and player accounts

- New SQLite-backed registration and login system; user passwords are hashed using Argon2id, account session tokens are kept hashed in the database, and account cookies are HttpOnly and SameSite=Strict.
- Player characters are associated with their accounts, allowing existing guest officers to be adopted on registration and returning players to load their game from another browser.
- A shared precinct shows the last-five-minutes roster and a top-25 experience leaderboard. Each registered officer has a public class, level, XP, Paragon and Renegade profile. Money, possessions, combat state, and game logs stay private.
- Solo/guest play continues to work; existing saved browser sessions remain compatible.
- `-secure-cookies` supports HTTPS termination at a Caddy reverse proxy; server still binds to loopback by default.
- Public multiplayer gameplay and real-calendar 30-day campaigns remain future work.

## v0.8.0 — longer-term karma perks and Psychic recharge

- **Karma thresholds:** Renegade's two-for-one Property Room purchases and the department's high-Paragon ambush trigger now require **rank 100**, up from 10. With up to 25 patrol encounters each day, rank 10 was too easy to reach through arrests and eliminations. The effects stay the same after unlocking.
- **Psychic Blast:** Psychic energy increases from **100 PSI (four blasts)** to **300 PSI (twelve blasts)** per shift. Each double-damage blast still spends 25 PSI and the meter fully refreshes at normal or forced clock-out.
- **Red Room PSI recovery:** a paid Red Room encounter restores **150 PSI (up to six blasts)** to a Psychic, never beyond 300. The normal $100 price, HP recovery, condom/infection chances, and story-result screen are retained.
- **Time cost:** a Red Room encounter now consumes **one remaining patrol turn** from the current shift, for every class. At zero turns, the action is unavailable even if the officer has enough money. The result screen shows the spent turn and recovered PSI.
- **Save compatibility:** when a Psychic's existing v0.7.x (or older) save is loaded, the remaining fraction of the old 100-PSI meter is converted to a 300-point meter. A depleted meter stays at zero; reopening the game cannot grant a free refill. All other progress remains intact.
- The recruitment screen, combat button, progress meter, Headquarters, Property Room, and Red Room show the updated limits.
- **Version:** bumped to v0.8.0 in `version.go`, with automated release consistency checks.

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
