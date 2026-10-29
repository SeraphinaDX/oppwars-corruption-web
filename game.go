// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
// A browser-oriented adaptation of the original terminal game's mechanics.
package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const fightsPerDay = 25
const starterHP = 200
const starterMoney = 5000

var levelThresholds = []int{1000, 4000, 9000, 18000, 30000, 60000, 120000, 240000, 480000, 960000, 1920000, 3840000, 7680000, 15360000}

// Player follows the names and starting values in the original Go program.
// ClubShares replaces ownership of people in this initial browser adaptation.
type Player struct {
	Name               string `json:"name"`
	Class              string `json:"class"`
	HP                 int    `json:"hp"`
	Attack             int    `json:"attack"`
	Money              int    `json:"money"`
	Days               int    `json:"days"`
	Experience         int    `json:"experience"`
	Level              int    `json:"level"`
	Condoms            int    `json:"condoms"`
	Handcuffs          int    `json:"handcuffs"`
	Infections         int    `json:"infections"`
	ClubShares         int    `json:"clubShares"`
	Cocaine            int    `json:"cocaine"`
	Fentanyl           int    `json:"fentanyl"`
	LSD                int    `json:"lsd"`
	ParagonRank        int    `json:"paragonRank"`
	RenegadeRank       int    `json:"renegadeRank"`
	Bumfights          int    `json:"fightsLeft"`
	ClassPower         int    `json:"classPower"` // Robot Charged Shot multiplier (1–4)
	PsychicEnergy      *int   `json:"psychicEnergy,omitempty"` // nil identifies pre-meter saves
	CocaineEncounters  int    `json:"cocaineEncounters"`
	FentanylEncounters int    `json:"fentanylEncounters"`
	LSDEncounters      int    `json:"lsdEncounters"`
}

type Encounter struct {
	Name  string `json:"name"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"maxHP"`
	Catchphrase string `json:"catchphrase,omitempty"`
}

type Decision struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Choices     []Choice `json:"choices"`
}
type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Game struct {
	Version   string     `json:"version"` // Release version presented in the browser
	Phase     string     `json:"phase"` // setup, hq, combat, event, redroom_result, gameover
	Player    Player     `json:"player"`
	Enemy     *Encounter `json:"enemy,omitempty"`
	Event     *Decision  `json:"event,omitempty"`
	RedRoom   *RedRoomResult `json:"redRoom,omitempty"`
	Messages  []string   `json:"messages"`
	Turn      int        `json:"turn"`
	EndReason string     `json:"endReason,omitempty"`
	CampaignStart string `json:"campaignStart,omitempty"` // Calendar date at 00:00 UTC
	LastResetDate string `json:"lastResetDate,omitempty"`
	ShiftEnded bool `json:"shiftEnded"`
}

var enemies = []string{
	"Hostile Street Drifter", "Alleyway Troublemaker", "Unhinged Roadside Heckler",
	"Aggressive Tent-Camp Raider", "Rogue District Thug",
}
var attacks = []string{
	"a broken bottle", "a rusty piece of metal", "a newspaper", "a crack pipe", "a suspiciously heavy boot",
}
var events = []Decision{
	{"THE EVIDENCE LOCKER", "An envelope of unclaimed cash appears in the evidence locker. Internal Affairs is conveniently out for lunch.", []Choice{{"renegade", "Pocket $500 and a couple of samples"}, {"paragon", "Log the money as evidence"}}},
	{"A PUBLIC COMPLAINT", "A citizen reports that your patrol car is parked across the sidewalk.", []Choice{{"renegade", "Issue the citizen a warning"}, {"paragon", "Move the car and apologize"}}},
	{"DEPARTMENT POLITICS", "Captain Grimstead invites you to rubber-stamp an absurd incident report.", []Choice{{"renegade", "Sign it and take the bonus"}, {"paragon", "Refuse to sign"}}},
	{"STREET INTELLIGENCE", "Someone offers to tell you where the contraband warehouse is, for a price.", []Choice{{"renegade", "Accept the suspicious payment"}, {"paragon", "File an official report"}}},
	{"THE MISSING BADGE", "A rookie has lost their badge and wants you to keep quiet about it.", []Choice{{"renegade", "Demand a favour"}, {"paragon", "Help them report it"}}},
}

func newGame() *Game {
	return &Game{Version: gameVersion, Phase: "setup", Messages: []string{"OPP HQ terminal ready. Report for duty."}}
}

func (g *Game) log(msg string) {
	g.Messages = append([]string{msg}, g.Messages...)
	if len(g.Messages) > 30 {
		g.Messages = g.Messages[:30]
	}
	g.Turn++
}

func (g *Game) act(action, name, class string) error {
	return g.actAt(action, name, class, time.Now())
}

// actAt is the authoritative action entry point with injectable server time.
// The calendar is synchronized first, so stale sessions cannot act yesterday.
func (g *Game) actAt(action, name, class string, now time.Time) error {
	g.syncCalendar(now)
	if action == "reset" {
		if g.Phase != "gameover" && g.Phase != "setup" {
			return errors.New("your 30 real-world duty days must finish before starting a new campaign")
		}
		*g = *newGame()
		return nil
	}
	if g.Phase == "setup" {
		if action != "start" {
			return errors.New("create your officer first")
		}
		name = strings.TrimSpace(name)
		if name == "" || utf8.RuneCountInString(name) > 24 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
			return errors.New("officer name must contain 1–24 printable characters")
		}
		if class != "robot" && class != "psychic" && class != "psychopath" {
			return errors.New("choose a class")
		}
		g.Player = Player{Name: name, Class: class, HP: starterHP, Attack: 40, Money: starterMoney, Days: campaignDays, Level: 1, Bumfights: fightsPerDay, ClassPower: 1}
		g.CampaignStart = utcDate(now)
		g.LastResetDate = utcDate(now)
		g.ShiftEnded = false
		if class == "psychic" {
			g.Player.refillPsychicEnergy()
		}
		g.Phase = "hq"
		g.log(fmt.Sprintf("Officer %s joined the force as a %s. Captain Grimstead looks worried.", name, class))
		return nil
	}
	if g.Phase == "gameover" {
		return errors.New("campaign ended — begin a new game")
	}
	if g.ShiftEnded {
		return errors.New("shift ended — return after midnight UTC for the next day's patrol turns")
	}
	p := &g.Player
	switch g.Phase {
	case "hq":
		switch action {
		case "patrol":
			if p.Bumfights <= 0 {
				return errors.New("no patrol encounters left today — clock out")
			}
			if rand.IntN(6) == 1 { // Original: 1 in 6 patrols opens a random event.
				evt := events[rand.IntN(len(events))]
				g.Event = &evt
				g.Phase = "event"
				g.log("Dispatch has a questionable new assignment.")
				return nil
			}
			g.startEncounter()
			return nil
		case "buy_cocaine":
			return g.buy(cocainePrice, "cocaine", 1)
		case "buy_fentanyl":
			return g.buy(fentanylPrice, "fentanyl", 1)
		case "buy_lsd":
			return g.buy(lsdPrice, "lsd", lsdPurchaseUnits)
		case "buy_handcuffs":
			return g.buy(handcuffsPrice, "handcuffs", handcuffsPerPurchase)
		case "use_cocaine", "use_fentanyl", "use_lsd":
			return g.useEffect(action)
		case "buy_condoms":
			if p.Money < condomPackPrice {
				return errors.New("not enough cash")
			}
			p.Money -= condomPackPrice
			p.Condoms += condomsPerPack
			g.log(fmt.Sprintf("%s buys %d condoms from the Red Room for $%d.", p.Name, condomsPerPack, condomPackPrice))
			return nil
		case "buy_club":
			if p.ClubShares >= maxClubShares {
				return errors.New("venue investment limit reached (5 shares)")
			}
			if p.Money < 2000 {
				return errors.New("not enough cash")
			}
			p.Money -= 2000
			p.ClubShares++
			g.log("Invested $2,000 in the Red Room. Each share pays $500 per shift.")
			return nil
		case "nightlife":
			const price = 100
			if p.Bumfights <= 0 {
				return errors.New("no patrol turns left today — clock out for the next shift")
			}
			if p.Money < price {
				return errors.New("not enough cash")
			}
			// Resolve all consequences exactly once, before entering the result
			// phase. A refresh only rereads the saved receipt.
			oldHP := p.HP
			protected := p.Condoms > 0
			p.Money -= price
			p.Bumfights-- // A visit occupies one of the day's 25 patrol turns.
			p.restoreHP(200)
			restoredPSI := p.restorePsychicEnergy(psychicRedRoomRestore)
			if restoredPSI > 0 {
				g.log(fmt.Sprintf("%s feels rejuvenated: +%d psychic energy (%d blasts).", p.Name, restoredPSI, restoredPSI/psychicBlastCost))
			}
			if protected {
				p.Condoms--
			}
			infected := !protected && rand.IntN(100) < 50
			if infected {
				p.Infections++
				g.log(fmt.Sprintf("%s leaves the Red Room with an unwelcome souvenir. The patrol sergeant notices the limp.", p.Name))
			} else if protected {
				g.log(fmt.Sprintf("%s leaves the Red Room in one piece. The doctor can take the night off.", p.Name))
			} else {
				g.log(fmt.Sprintf("%s leaves the Red Room feeling better, and with no new complaints.", p.Name))
			}
			g.RedRoom = &RedRoomResult{
				Scene: newRedRoomScene(p.Name),
				Protected: protected,
				InfectionGained: infected,
				TotalInfections: p.Infections,
				HealthRecovered: p.HP - oldHP,
				PsychicEnergyRestored: restoredPSI,
				TurnsSpent: 1,
				Cost: price,
			}
			g.Phase = "redroom_result"
			return nil
		case "clinic":
			if p.Infections == 0 {
				return errors.New("the doctor says you're fine")
			}
			cost := p.treatmentCost()
			if p.Money < cost {
				return fmt.Errorf("treatment costs $%d for %d infection(s)", cost, p.Infections)
			}
			cured := p.Infections
			p.Money -= cost
			p.Infections = 0
			g.log(fmt.Sprintf("The doctor clears %s for duty: %d infection(s) treated for $%d.", p.Name, cured, cost))
			return nil
		case "clockout":
			g.endShift(false)
			return nil
		default:
			return errors.New("action unavailable at HQ")
		}
	case "redroom_result":
		if action != "continue" {
			return errors.New("finish your Red Room encounter before returning to duty")
		}
		g.RedRoom = nil
		g.Phase = "hq"
		g.log(fmt.Sprintf("%s returns from the Red Room to headquarters.", p.Name))
		return nil
	case "event":
		if action != "choose_renegade" && action != "choose_paragon" {
			return errors.New("choose your response")
		}
		if action == "choose_renegade" {
			p.RenegadeRank++
			p.ParagonRank--
			// Money and drug effects echo the original branch-specific events.
			switch g.Event.Title {
			case "THE EVIDENCE LOCKER":
				p.Money += 500
				p.Cocaine = min(maxInventoryEach, p.Cocaine+2)
			case "DEPARTMENT POLITICS":
				p.Money += 1000
			case "STREET INTELLIGENCE":
				p.Fentanyl = min(maxInventoryEach, p.Fentanyl+2)
			}
			g.log("Renegade decision recorded. Captain Grimstead approves.")
		} else {
			p.ParagonRank++
			p.RenegadeRank--
			g.log("Paragon decision recorded. Internal Affairs is suspiciously impressed.")
		}
		g.Event = nil
		g.startEncounter()
		return nil
	case "combat":
		if g.Enemy == nil {
			return errors.New("missing encounter")
		}
		switch action {
		case "retreat":
			p.Bumfights--
			for _, msg := range p.finishEncounter() {
				g.log(msg + ".")
			}
			g.Enemy = nil
			g.Phase = "hq"
			g.log(fmt.Sprintf("%s retreats to headquarters. One patrol encounter used.", p.Name))
			return nil
		case "arrest":
			if p.Handcuffs <= 0 {
				return errors.New("no hand cuffs in the property room inventory")
			}
			if !canArrestEnemy(g.Enemy) {
				return errors.New("weaken the suspect to 50% HP or less before arresting")
			}
			p.Handcuffs--
			g.finishCombat(true)
			return nil
		case "fight":
			if rand.IntN(100) > 40 {
				damage := p.normalDamage()
				if p.LSDEncounters > 0 {
					var distorted bool
					damage, distorted = distortedDamage(damage, rand.IntN(100))
					if distorted {
						g.log(fmt.Sprintf("%s swings at a mirage. The hit loses half its force.", p.Name))
					}
				}
				g.Enemy.HP -= damage
				g.log(fmt.Sprintf("%s hits %s for %d HP. The paperwork practically fills itself out.", p.Name, g.Enemy.Name, damage))
			} else if p.LSDEncounters > 0 && foresightDodge(rand.IntN(100)) {
				g.log(fmt.Sprintf("%s sees the counterattack coming and dodges!", p.Name))
			} else {
				hit := p.counterDamage()
				p.HP -= hit
				g.log(fmt.Sprintf("%s misses %s! %s strikes %s with %s for %d HP.", p.Name, g.Enemy.Name, g.Enemy.Name, p.Name, attacks[rand.IntN(len(attacks))], hit))
			}
		case "special":
			// Charged Shot charges from 1x to 4x over combat turns;
			// Psychic Blast is always 2x damage and costs 25 of 300 PSI.
			damage := p.Attack * p.Level
			label := "Chow Down"
			switch p.Class {
			case "robot":
				power := max(1, min(maxClassPower, p.ClassPower))
				damage *= power
				label = fmt.Sprintf("Charged Shot (%dx)", power)
			case "psychic":
				if !p.spendPsychicEnergy() {
					return errors.New("not enough psychic energy — clock out to recharge")
				}
				damage *= 2
				label = "Psychic Blast"
			case "psychopath":
				label = "Chow Down"
			}
			// All three specials share the same infection penalty as normal attacks.
			damage = p.infectionAdjustedDamage(damage)
			if p.LSDEncounters > 0 {
				var distorted bool
				damage, distorted = distortedDamage(damage, rand.IntN(100))
				if distorted {
					g.log(fmt.Sprintf("%s targets a hallucination. Special damage is halved.", p.Name))
				}
			}
			if p.Class == "psychopath" {
				p.restoreHP(damage / 2)
			}
			g.Enemy.HP -= damage
			g.log(fmt.Sprintf("%s uses %s against %s for %d damage.", p.Name, label, g.Enemy.Name, damage))
			if p.Class == "robot" {
				p.ClassPower = 1 // A fired Charged Shot must build power again.
			}
		default:
			return errors.New("invalid combat action")
		}
		if p.HP <= 0 {
			g.losePatrol(g.Enemy.Name)
			return nil
		}
		if g.Enemy.HP <= 0 {
			g.finishCombat(false)
			return nil
		}
		if rand.IntN(3) == 0 {
			g.Enemy.Catchphrase = nextCatchphrase(g.Enemy.Name, g.Enemy.Catchphrase)
			g.log(fmt.Sprintf("%s shouts: %q", g.Enemy.Name, g.Enemy.Catchphrase))
		}
		if p.Class == "robot" && action == "fight" {
			// Charge grows only after normal combat turns. Fired specials reset to 1.
			p.ClassPower = min(maxClassPower, p.ClassPower+1)
		}
		return nil
	default:
		return errors.New("invalid game phase")
	}
}

func (g *Game) buy(price int, item string, units int) error {
	p := &g.Player
	if p.Money < price {
		return errors.New("not enough cash")
	}
	if p.renegadeShopPerk() {
		units *= 2
	}
	var owned int
	switch item {
	case "cocaine":
		owned = p.Cocaine
	case "fentanyl":
		owned = p.Fentanyl
	case "lsd":
		owned = p.LSD
	case "handcuffs":
		owned = p.Handcuffs
	default:
		return errors.New("unknown item")
	}
	if owned+units > maxInventoryEach {
		return errors.New("property room inventory limit reached (20 per item)")
	}
	p.Money -= price
	switch item {
	case "cocaine":
		p.Cocaine += units
	case "fentanyl":
		p.Fentanyl += units
	case "lsd":
		p.LSD += units
	case "handcuffs":
		p.Handcuffs += units
	}
	g.log(fmt.Sprintf("Property room: bought %d %s for $%d.", units, item, price))
	return nil
}
// finishCombat resolves either a nonlethal arrest or an enemy elimination.
// Both consume one patrol and end temporary encounter buffs exactly once.
// Arrests deliberately grant lower immediate rewards in exchange for Paragon.
func (g *Game) finishCombat(arrested bool) {
	p := &g.Player
	opponent := g.Enemy.Name
	g.Enemy = nil
	g.Phase = "hq"
	p.Bumfights--

	pay := p.victoryPay()
	baseXP, bonusXP := p.earnedXP()
	if arrested {
		pay /= arrestRewardDivisor
		baseXP /= arrestRewardDivisor
		bonusXP /= arrestRewardDivisor
		p.ParagonRank += combatKarmaGain
	} else {
		p.RenegadeRank += combatKarmaGain
	}
	gained := baseXP + bonusXP
	p.Money += pay
	p.Experience += gained
	oldLevel := p.Level
	for i, threshold := range levelThresholds {
		if p.Experience >= threshold {
			p.Level = i + 2
		}
	}
	if arrested {
		g.log(fmt.Sprintf("%s arrests %s and brings them in! +%d Paragon, +%d XP (%d bonus), +$%d.", p.Name, opponent, combatKarmaGain, gained, bonusXP, pay))
	} else {
		g.log(fmt.Sprintf("%s defeats %s! +%d Renegade, +%d XP (%d bonus), +$%d.", p.Name, opponent, combatKarmaGain, gained, bonusXP, pay))
	}
	for _, msg := range p.finishEncounter() {
		g.log(msg + ".")
	}
	if p.Level > oldLevel {
		g.log(fmt.Sprintf("PROMOTION: %s reaches level %d!", p.Name, p.Level))
	}
}


// losePatrol handles both ordinary enemy defeats and lethal department ambushes.
// The campaign only ends when the 30-day tour reaches its natural last shift.
func (g *Game) losePatrol(opponent string) {
	g.Player.HP = 0
	g.log(fmt.Sprintf("%s was defeated by %s and taken to the medical wing.", g.Player.Name, opponent))
	g.endShift(true)
}

func (g *Game) startEncounter() {
	p := &g.Player
	maxHP := 100 * p.Level
	g.Enemy = &Encounter{Name: enemies[rand.IntN(len(enemies))], HP: maxHP, MaxHP: maxHP}
	g.Enemy.Catchphrase = nextCatchphrase(g.Enemy.Name, "")
	if p.Class == "robot" {
		p.ClassPower = min(maxClassPower, p.ClassPower+1)
	}
	g.Phase = "combat"
	g.log(fmt.Sprintf("%s runs into %s on patrol.", p.Name, g.Enemy.Name))
	g.log(fmt.Sprintf("%s yells: %q", g.Enemy.Name, g.Enemy.Catchphrase))
	if p.paragonAmbushEligible() && rand.IntN(4) == 1 {
		hit := min(50*p.Level, p.maxHP()/3)
		p.HP -= hit
		g.log(fmt.Sprintf("Captain Grimstead arranges a drive-by against %s, costing %d HP.", p.Name, hit))
		if p.HP <= 0 {
			g.losePatrol("Captain Grimstead's ambush")
		}
	}
}

// useEffect spends one inventory unit and starts a short, non-stacking effect.
// These fictional game mechanics do not describe real-world drug effects.
func (g *Game) useEffect(action string) error {
	p := &g.Player
	if p.effectActive() {
		return errors.New("an effect is already active — finish it before using another")
	}
	switch action {
	case "use_cocaine":
		if p.Cocaine == 0 {
			return errors.New("no cocaine in inventory")
		}
		p.Cocaine--
		p.CocaineEncounters = cocaineEncounters
		g.log("XP Rush active: +25% patrol XP (up to +150) for 3 encounters.")
	case "use_fentanyl":
		if p.Fentanyl == 0 {
			return errors.New("no fentanyl in inventory")
		}
		p.Fentanyl--
		p.FentanylEncounters = fentanylEncounters
		g.log("Reckless Strength active for 2 encounters: +25% normal attack damage, +5 retaliation damage.")
	case "use_lsd":
		if p.LSD == 0 {
			return errors.New("no LSD in inventory")
		}
		p.LSD--
		p.LSDEncounters = lsdEncounters
		g.log("REALITY DISTORTION: the streetlights are giving a performance review. Two encounters remaining.")
	default:
		return errors.New("unknown effect")
	}
	return nil
}
