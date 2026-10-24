package main

import (
	"strings"
	"testing"
)

// Guard the combat HUD ordering so new taunts don't push the HP bar below
// the enemy's dialogue again.
func TestEnemyHealthBarAboveCatchphrase(t *testing.T) {
	page, err := embeddedAssets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(page)
	start := strings.Index(js, "function patrolPage(g)")
	end := start + strings.Index(js[start:], "if(g.phase==='event')")
	if start < 0 || end <= start {
		t.Fatal("cannot locate combat UI rendering code")
	}
	combat := js[start:end]
	hpLabel := strings.Index(combat, "Enemy health ${fmt(")
	bar := strings.Index(combat, "<div class=\"bar\" role=\"progressbar\" aria-label=\"Enemy health\"")
	taunt := strings.Index(combat, "${e.catchphrase?")
	if hpLabel < 0 || bar < 0 || taunt < 0 {
		t.Fatalf("combat HUD missing HP readout, accessible bar or catchphrase: %d %d %d", hpLabel, bar, taunt)
	}
	if !(hpLabel < bar && bar < taunt) {
		t.Fatalf("incorrect combat HUD order: HP %d, bar %d, catchphrase %d", hpLabel, bar, taunt)
	}
	if !strings.Contains(combat, "aria-valuenow=") {
		t.Fatal("enemy health bar does not expose current HP")
	}
}
