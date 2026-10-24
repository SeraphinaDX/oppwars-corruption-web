package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCombatDefeatForcesNextShiftInsteadOfGameOver(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Officer Dusk", "robot"); err != nil {
		t.Fatal(err)
	}
	beforeCash := g.Player.Money
	g.startEncounter()
	enemyName := g.Enemy.Name
	g.Enemy.HP = 99999 // Special cannot win the encounter.
	g.Player.HP = 0 // Force the defeat branch without depending on random hits.
	g.Player.Bumfights = 12
	g.Player.CocaineEncounters = 2
	g.Player.ClassPower = 4
	if err := g.act("special", "", ""); err != nil {
		t.Fatal(err)
	}
	p := g.Player
	if g.Phase != "hq" || g.Enemy != nil || g.EndReason != "" {
		t.Fatalf("defeat ended the campaign or retained enemy: %+v", g)
	}
	if p.Days != 29 || p.Bumfights != fightsPerDay {
		t.Fatalf("defeat didn't end the shift: days=%d patrols=%d", p.Days, p.Bumfights)
	}
	if p.HP != p.maxHP()/2 || p.ClassPower != 1 {
		t.Fatalf("hospital recovery/charge reset wrong: HP=%d power=%d", p.HP, p.ClassPower)
	}
	if p.Money != beforeCash || p.Experience != 0 || p.CocaineEncounters != 0 {
		t.Fatalf("defeat improperly granted rewards or kept buffs: %+v", p)
	}
	if !strings.Contains(strings.Join(g.Messages, "\n"), "defeated by "+enemyName) ||
		!strings.Contains(strings.Join(g.Messages, "\n"), "Forced clock-out") {
		t.Fatalf("missing named defeat and forced clock-out messages: %v", g.Messages)
	}
	if err := g.act("patrol", "", ""); err != nil && g.Phase != "event" {
		t.Fatalf("cannot continue the campaign after defeat: %v", err)
	}
}

func TestCombatDefeatRefillsPsychicEnergyButKeepsInfections(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Officer Psi", "psychic"); err != nil { t.Fatal(err) }
	g.Player.Infections = 2
	g.Player.HP = 0
	noEnergy := 0
	g.Player.PsychicEnergy = &noEnergy
	g.startEncounter()
	g.Enemy.HP = 99999
	before := g.Player.Money
	// Even a successful normal attack will not prevent defeat with 0 HP.
	if err := g.act("fight", "", ""); err != nil { t.Fatal(err) }
	if g.Phase != "hq" || g.Player.Days != 29 { t.Fatalf("wrong phase/day %+v", g) }
	if g.Player.psychicLeft() != maxPsychicEnergy { t.Fatal("psychic meter not reset on new shift") }
	if g.Player.Infections != 2 || g.Player.Bumfights != minInfectedPatrols {
		t.Fatalf("defeat removed infection or bypassed patrol penalty: %+v", g.Player)
	}
	if g.Player.Money != before || g.Player.HP != g.Player.maxHP()/2 { t.Fatalf("wrong recovery: %+v", g.Player) }
	if err := g.act("clockout", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Days != 28 || g.Player.Money != before+g.Player.shiftIncome() {
		t.Fatalf("normal shift pay broken after defeat: %+v",g.Player)
	}
}

func TestLastDayDefeatOnlyEndsCampaignNormally(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Last","psychopath");err!=nil {t.Fatal(err)}
	g.Player.Days=1
	g.Player.HP=0
	g.startEncounter()
	g.Enemy.HP=99999
	if err:=g.act("fight","","");err!=nil {t.Fatal(err)}
	if g.Phase!="gameover" || g.Player.Days!=0 {
		t.Fatalf("final campaign day didn't complete normally: %+v",g)
	}
	if !strings.Contains(g.EndReason,"30-day tour") {
		t.Fatalf("wrong end reason: %q",g.EndReason)
	}
	if err:=g.act("patrol","","");err==nil {t.Fatal("patrol allowed after campaign was completed")}
}

func TestDepartmentAmbushDefeatAlsoEndsShift(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Ambush","robot");err!=nil {t.Fatal(err)}
	g.Player.Days=13
	g.Player.HP=0
	g.startEncounter()
	g.losePatrol("Captain Grimstead's ambush")
	if g.Phase!="hq" || g.Enemy!=nil || g.Player.Days!=12 || g.Player.HP!=100 {
		t.Fatalf("ambush defeat didn't advance to next shift: %+v",g)
	}
}

func TestDefeatStateSurvivesSaveReload(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Persist","robot");err!=nil {t.Fatal(err)}
	g.startEncounter()
	g.Enemy.HP=99999
	g.Player.HP=0
	if err:=g.act("special","","");err!=nil {t.Fatal(err)}
	raw,err:=json.Marshal(g);if err!=nil {t.Fatal(err)}
	var reload Game
	if err:=json.Unmarshal(raw,&reload);err!=nil {t.Fatal(err)}
	if !validGame(&reload) || reload.Phase!="hq" || reload.Player.Days!=29 || reload.Player.HP!=100 {
		t.Fatalf("loss doesn't persist playable state: %+v",reload)
	}
}
