package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKarmaPerksRequireOneHundredRank(t *testing.T) {
	for _, n := range []int{0, 10, 25, 75, 99} {
		p := Player{ParagonRank: n, RenegadeRank: n}
		if p.paragonAmbushEligible() || p.renegadeShopPerk() {
			t.Fatalf("rank %d unlocked expensive karma perks prematurely", n)
		}
	}
	p := Player{ParagonRank: 100, RenegadeRank: 100}
	if !p.paragonAmbushEligible() || !p.renegadeShopPerk() {
		t.Fatal("rank 100 must unlock both eligibility checks")
	}
	g := newGame()
	if err := g.act("start", "Karma Tester", "robot"); err != nil { t.Fatal(err) }
	g.Player.RenegadeRank = 99
	if err := g.act("buy_handcuffs", "", ""); err != nil {t.Fatal(err)}
	if g.Player.Handcuffs != 1 {t.Fatal("rank 99 was granted the 2-for-1 perk")}
	g.Player.RenegadeRank = 100
	if err := g.act("buy_handcuffs", "", ""); err != nil {t.Fatal(err)}
	if g.Player.Handcuffs != 3 {t.Fatal("rank 100 must receive 2 handcuffs per purchase")}
}

func TestPsychicMeterHasTwelveBlastsAndPartialRefill(t *testing.T) {
	p := Player{Class: "psychic"}
	p.refillPsychicEnergy()
	if p.psychicLeft() != maxPsychicEnergy || maxPsychicEnergy/psychicBlastCost != 12 {
		t.Fatal("new Psychic meter should store 12 double-damage blasts")
	}
	for i:=0;i<12;i++ {
		if !p.spendPsychicEnergy() {t.Fatalf("blast %d was unavailable", i+1)}
	}
	if p.psychicLeft()!=0 || p.spendPsychicEnergy() {t.Fatal("meter did not stop at 12")}
	if restored:=p.restorePsychicEnergy(psychicRedRoomRestore); restored!=150 || p.psychicLeft()!=150 {
		t.Fatalf("first visit should restore six blasts, got %d PSI",restored)
	}
	if restored:=p.restorePsychicEnergy(psychicRedRoomRestore); restored!=150 || p.psychicLeft()!=300 {
		t.Fatalf("second visit should reach 12 stored blasts, got %d PSI",restored)
	}
	if restored:=p.restorePsychicEnergy(psychicRedRoomRestore); restored!=0 || p.psychicLeft()!=300 {
		t.Fatal("visits must not charge PSI above the meter cap")
	}
}

func TestRedRoomPsychicRefillConsumesOneTurn(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Psychic Officer","psychic");err!=nil {t.Fatal(err)}
	zero:=0
	g.Player.PsychicEnergy=&zero
	g.Player.Condoms=2
	beforeCash:=g.Player.Money
	if err:=g.act("nightlife","","");err!=nil {t.Fatal(err)}
	if g.Phase!="redroom_result" || g.RedRoom==nil {t.Fatal("missing Red Room result")}
	if g.Player.Bumfights!=24 || g.Player.Money!=beforeCash-100 || g.Player.Condoms!=1 {
		t.Fatalf("visit didn't spend one patrol turn and $100: %+v",g.Player)
	}
	if g.Player.psychicLeft()!=150 || g.RedRoom.PsychicEnergyRestored!=150 || g.RedRoom.TurnsSpent!=1 {
		t.Fatalf("visit did not restore six blasts: %+v",g.RedRoom)
	}
	receipt:=*g.RedRoom
	raw,err:=json.Marshal(g);if err!=nil {t.Fatal(err)}
	var restored Game
	if err:=json.Unmarshal(raw,&restored);err!=nil {t.Fatal(err)}
	if restored.RedRoom==nil || *restored.RedRoom!=receipt || restored.Player.Bumfights!=24 || restored.Player.psychicLeft()!=150 {
		t.Fatal("save/reload charged more turns or rerolled PSI")
	}
	if err:=g.act("nightlife","","");err==nil {t.Fatal("repeated visit allowed before Continue")}
	if g.Player.Bumfights!=24 || g.Player.psychicLeft()!=150 {t.Fatal("failed visit changed resources")}
	if err:=g.act("continue","","");err!=nil {t.Fatal(err)}
	if err:=g.act("nightlife","","");err!=nil {t.Fatal(err)}
	if g.Player.Bumfights!=23 || g.Player.psychicLeft()!=300 || g.RedRoom.PsychicEnergyRestored!=150 {
		t.Fatalf("second paid visit did not cap at 12 blasts: %+v",g.Player)
	}
	if err:=g.act("continue","","");err!=nil {t.Fatal(err)}
	if err:=g.act("nightlife","","");err!=nil {t.Fatal(err)}
	if g.Player.Bumfights!=22 || g.RedRoom.PsychicEnergyRestored!=0 {
		t.Fatal("third visit must still cost one turn but not overfill PSI")
	}
}

func TestRedRoomRejectsVisitWhenNoTurnsRemain(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Psychic Officer","psychic");err!=nil {t.Fatal(err)}
	g.Player.Bumfights=0
	g.Player.Condoms=1
	g.Player.HP=50
	v:=0
	g.Player.PsychicEnergy=&v
	initialMoney:=g.Player.Money
	err:=g.act("nightlife","","")
	if err==nil || !strings.Contains(err.Error(),"no patrol turns") {t.Fatalf("should reject visit without turns: %v",err)}
	if g.Phase!="hq" || g.RedRoom!=nil || g.Player.HP!=50 || g.Player.Money!=initialMoney ||
		g.Player.Condoms!=1 || g.Player.psychicLeft()!=0 || g.Player.Bumfights!=0 {
		t.Fatalf("rejected visit charged resources: %+v",g)
	}
}

func TestLegacyPsychicSavesPreserveFractionAndEmptyMeter(t *testing.T) {
	cases:=[]struct{version string; saved, want int}{
		{"0.7.0",100,300},
		{"0.7.0",75,225},
		{"0.6.2",25,75},
		{"0.7.0",0,0},
		{"",50,150},
		{"0.8.0",75,75},
	}
	for _,tc:=range cases {
		old:=tc.saved
		p:=Player{Class:"psychic",PsychicEnergy:&old}
		p.migratePsychicEnergy(tc.version)
		p.normalizeLegacyClassResources()
		if got:=p.psychicLeft();got!=tc.want {t.Errorf("saved version %q / %d PSI migrated to %d; expected %d",tc.version,tc.saved,got,tc.want)}
	}
}
