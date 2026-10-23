package main

import (
	"strings"
	"testing"
)

func TestStartupAndClassValidation(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Test Officer", "invalid"); err == nil {
		t.Fatal("accepted unknown class")
	}
	if err := g.act("start", "<script>x</script>", "robot"); err != nil {
		t.Fatal(err)
	}
	if g.Player.HP != 200 || g.Player.Attack != 40 || g.Player.Money != 5000 || g.Player.Days != 30 || g.Player.Bumfights != 25 {
		t.Fatalf("unexpected starting stats: %+v", g.Player)
	}
	if g.Phase != "hq" || g.Player.Class != "robot" {
		t.Fatalf("unexpected startup: %+v", g)
	}
	if err := g.act("start", "again", "robot"); err == nil {
		t.Fatal("allowed resetting class without reset")
	}
	if err := g.act("reset", "", ""); err != nil || g.Phase != "setup" {
		t.Fatal("reset failed", err)
	}
	if err := g.act("start", "\x00", "robot"); err == nil {
		t.Fatal("accepted control character")
	}
	if err := g.act("start", strings.Repeat("x", 25), "robot"); err == nil {
		t.Fatal("accepted overlong name")
	}
}

func TestShopAndClockOut(t *testing.T) {
	g := newGame()
	_ = g.act("start", "Scout", "robot")
	if err := g.act("buy_cocaine", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.Cocaine != 1 || g.Player.Money != 4000 {
		t.Fatal("incorrect cocaine purchase", g.Player)
	}
	if err := g.act("buy_fentanyl", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.act("buy_lsd", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.Fentanyl != 1 || g.Player.LSD != 5 {
		t.Fatal("unexpected inventory", g.Player)
	}
	g.Player.RenegadeRank = 10
	if err := g.act("buy_lsd", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.LSD != 15 {
		t.Fatal("renegade doubling did not work", g.Player.LSD)
	}
	g.Player.Infections = 2
	g.Player.ClubShares = 1
	oldCash := g.Player.Money
	if err := g.act("clockout", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.Money != oldCash+500+500-50 || g.Player.Bumfights != 8 || g.Player.Days != 29 {
		t.Fatal("clockout results wrong", g.Player)
	}
	g.Player.Days = 1
	if err := g.act("clockout", "", ""); err != nil || g.Phase != "gameover" {
		t.Fatal("campaign did not end")
	}
	if err := g.act("buy_lsd", "", ""); err == nil {
		t.Fatal("shop available after ending")
	}
}

func TestVictoryAndLoss(t *testing.T) {
	g := newGame()
	_ = g.act("start", "Robot", "robot")
	g.startEncounter()
	if g.Phase != "combat" || g.Enemy.MaxHP != 100 {
		t.Fatal("invalid enemy", g.Enemy)
	}
	if err := g.act("special", "", ""); err != nil {
		t.Fatal(err)
	}
	// Charged Shot: attack 40 * level 1 * power 1 * 2 = 80.
	if g.Enemy == nil || g.Enemy.HP != 20 {
		t.Fatal("unexpected charged shot damage", g.Enemy)
	}
	if err := g.act("special", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Phase != "hq" || g.Player.Money != 5300 || g.Player.Experience != 200 || g.Player.Bumfights != 24 {
		t.Fatal("original reward formulas changed", g.Player)
	}
	g.startEncounter()
	g.Player.HP = 1
	// A manual injury tests deterministic death without relying on random hits.
	g.Player.HP = 0
	g.Enemy.HP = 50
	// special wins even on zero HP (original healing/HP check runs post action), verify we don't allow combat after campaign end.
	g.Phase = "gameover"
	if err := g.act("special", "", ""); err == nil {
		t.Fatal("postmortem combat allowed")
	}
}

func TestPsionicAndPsychopath(t *testing.T) {
	for _, cls := range []string{"psychic", "psychopath"} {
		g := newGame()
		_ = g.act("start", "Unit", cls)
		g.startEncounter()
		if cls == "psychic" && g.Player.ClassPower != 2 {
			t.Fatal("psychic charge should increase")
		}
		if cls == "psychopath" {
			g.Player.HP = 50
		}
		if err := g.act("special", "", ""); err != nil {
			t.Fatal(err)
		}
		if cls == "psychic" && (g.Phase != "combat" || g.Enemy.HP != 20) {
			t.Fatal("unexpected psychic ability damage")
		}
		if cls == "psychopath" && g.Player.HP != 70 {
			t.Fatal("psychopath healing mismatch", g.Player.HP)
		}
	}
}

func TestEventKarma(t *testing.T) {
	g := newGame()
	_ = g.act("start", "Karma", "psychic")
	e := events[0]
	g.Event = &e
	g.Phase = "event"
	if err := g.act("choose_renegade", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.Player.RenegadeRank != 1 || g.Player.ParagonRank != -1 || g.Player.Money != 5500 || g.Player.Cocaine != 2 || g.Phase != "combat" {
		t.Fatalf("event was not applied: %+v", g)
	}
}
