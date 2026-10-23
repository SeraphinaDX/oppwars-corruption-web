package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLSDActivationAndExpiration(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Tester", "robot"); err != nil {
		t.Fatal(err)
	}
	if err := g.act("buy_lsd", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.act("use_lsd", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.LSD != 4 || g.Player.LSDEncounters != 2 {
		t.Fatalf("LSD should consume exactly one tab and last two encounters: %+v", g.Player)
	}
	if err := g.act("use_lsd", "", ""); err == nil {
		t.Fatal("re-activation stacked an effect")
	}
	if err := g.act("use_cocaine", "", ""); err == nil {
		t.Fatal("different effects should never stack")
	}
	g.startEncounter()
	if err := g.act("retreat", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.LSDEncounters != 1 {
		t.Fatal("retreat should consume one encounter")
	}
	g.startEncounter()
	g.Enemy.HP = 1
	if err := g.act("special", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.LSDEncounters != 0 || g.Phase != "hq" {
		t.Fatalf("LSD did not expire on victory: %+v", g)
	}
	if err := g.act("use_lsd", "", ""); err != nil {
		t.Fatal("should be able to use second tab", err)
	}
	if err := g.act("clockout", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.effectActive() {
		t.Fatal("clockout should end temporary effects")
	}
}

func TestBuffBalance(t *testing.T) {
	p := Player{Attack: 40, Level: 4, HP: 100, Cocaine: 20, Fentanyl: 20}
	if got := p.normalDamage(); got != 160 {
		t.Fatalf("inventory gave passive damage %d", got)
	}
	if base, bonus := p.earnedXP(); base != 800 || bonus != 0 {
		t.Fatalf("inventory gave passive XP %d, %d", base, bonus)
	}
	p.FentanylEncounters = 2
	if got := p.normalDamage(); got != 200 {
		t.Fatalf("expected 25%% damage, got %d", got)
	}
	if got := p.counterDamage(); got != 25 {
		t.Fatalf("missing downside, got %d", got)
	}
	p.FentanylEncounters = 0
	p.CocaineEncounters = 3
	if base, bonus := p.earnedXP(); base != 800 || bonus != 150 {
		t.Fatalf("XP bonus should be capped: %d, %d", base, bonus)
	}
	if hit, bad := distortedDamage(100, 19); hit != 50 || !bad {
		t.Fatal("distortion must halve damage with <20 roll")
	}
	if hit, bad := distortedDamage(100, 20); hit != 100 || bad {
		t.Fatal("distortion must stop at 20")
	}
	if !foresightDodge(24) || foresightDodge(25) {
		t.Fatal("dodge percentage incorrect")
	}
}

func TestBalancedCashAndHealing(t *testing.T) {
	p := Player{Level: 5, HP: 199, ClubShares: 999}
	if p.victoryPay() != 600 {
		t.Fatalf("victory pay must be linear, got %d", p.victoryPay())
	}
	if p.shiftIncome() != 3000 {
		t.Fatalf("club payouts should be capped, got %d", p.shiftIncome())
	}
	p.restoreHP(2000)
	if p.HP != 360 {
		t.Fatalf("HP needs a level based cap, got %d", p.HP)
	}
	p.Level = 100
	p.restoreHP(2000)
	if p.HP != maxOfficerHP {
		t.Fatalf("max HP should cap at %d", maxOfficerHP)
	}
	g := newGame()
	_ = g.act("start", "Tester", "robot")
	g.Player.Money = 50000
	for i := 0; i < maxClubShares; i++ {
		if err := g.act("buy_club", "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.act("buy_club", "", ""); err == nil {
		t.Fatal("allowed unlimited club shares")
	}
}

func TestTemporaryXPAndStockLimits(t *testing.T) {
	g := newGame()
	_ = g.act("start", "Tester", "robot")
	_ = g.act("buy_cocaine", "", "")
	if err := g.act("use_cocaine", "", ""); err != nil {
		t.Fatal(err)
	}
	g.startEncounter()
	g.Enemy.HP = 1
	if err := g.act("special", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.Experience != 250 || g.Player.Money != 4300 || g.Player.CocaineEncounters != 2 {
		t.Fatalf("unexpected reward and effect countdown: %+v", g.Player)
	}
	g.Player.LSD = 20
	if err := g.act("buy_lsd", "", ""); err == nil {
		t.Fatal("allowed buying beyond inventory limit")
	}
	g.Player.RenegadeRank = 10
	g.Player.LSD = 12
	if err := g.act("buy_lsd", "", ""); err == nil {
		t.Fatal("renegade doubled purchase should respect limit")
	}
}

func TestLegacySavesRemainReadable(t *testing.T) {
	const old = `{"phase":"hq","player":{"name":"Veteran","class":"robot","hp":200,"attack":40,"money":5000,"days":30,"level":1,"lsd":5},"messages":[]}`
	var g Game
	if err := json.Unmarshal([]byte(old), &g); err != nil {
		t.Fatal(err)
	}
	if !validGame(&g) || g.Player.LSD != 5 || g.Player.effectActive() {
		t.Fatalf("v0.1 save broken: %+v", g)
	}
	if err := g.act("use_lsd", "", ""); err != nil {
		t.Fatal("old inventory unusable", err)
	}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"lsdEncounters":2`) {
		t.Fatal("effect countdown omitted from save")
	}
}

func TestPsychicPowerCapped(t *testing.T) {
	g := newGame()
	_ = g.act("start", "Psychic", "psychic")
	for i := 0; i < 10; i++ {
		g.startEncounter()
		_ = g.act("fight", "", "")
		if g.Player.ClassPower > maxClassPower {
			t.Fatalf("power leaked above cap: %d", g.Player.ClassPower)
		}
		if g.Phase == "combat" {
			_ = g.act("retreat", "", "")
		}
		if g.Phase == "gameover" {
			break
		}
	}
}
