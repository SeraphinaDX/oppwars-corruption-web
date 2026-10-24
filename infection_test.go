package main

import (
	"strings"
	"testing"
)

func TestCondomPricesAndProtection(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Officer Delta", "psychopath"); err != nil { t.Fatal(err) }
	g.Player.Money = condomPackPrice - 1
	if err := g.act("buy_condoms", "", ""); err == nil { t.Fatal("allowed unaffordable condoms") }
	g.Player.Money = condomPackPrice + 100
	if err := g.act("buy_condoms", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Money != 100 || g.Player.Condoms != condomsPerPack {
		t.Fatalf("wrong purchase cost or quantity: %+v", g.Player)
	}
	g.Player.HP = 100
	if err := g.act("nightlife", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Condoms != condomsPerPack-1 || g.Player.Infections != 0 || g.Player.HP != 200 || g.Player.Money != 0 {
		t.Fatalf("protected Red Room encounter changed: %+v", g.Player)
	}
}

func TestInfectionsCauseCombatAndShiftPenalties(t *testing.T) {
	for _, tc := range []struct {
		infections int
		normal     int
		patrols    int
	}{
		{0, 40, 25},
		{1, 34, 7},
		{2, 28, 5},
		{3, 22, 5},
		{99, 22, 5}, // Bounded combat damage and guaranteed future patrols.
	} {
		p := Player{Attack: 40, Level: 1, Infections: tc.infections}
		if got := p.normalDamage(); got != tc.normal {
			t.Errorf("%d infections: damage=%d want %d", tc.infections, got, tc.normal)
		}
		if got := p.shiftPatrols(); got != tc.patrols {
			t.Errorf("%d infections: patrols=%d want %d", tc.infections, got, tc.patrols)
		}
	}
	p := Player{Level: 1, Infections: 1}
	if got := p.shiftIncome(); got != 350 { t.Fatalf("one infection shift income=%d", got) }
	p.Infections = 2
	if got := p.shiftIncome(); got != 200 { t.Fatalf("two infections shift income=%d", got) }
	p.Infections = 99
	if got := p.shiftIncome(); got != minShiftIncome { t.Fatalf("income floor broken: %d",got) }

	p = Player{Attack: 40, Level: 1, Infections: 1, FentanylEncounters: 2}
	if got := p.normalDamage(); got != 42 { t.Fatalf("fentanyl and infection modifier got %d want 42",got) }
}

func TestInfectionsWeakenAllClassSpecials(t *testing.T) {
	for _, tc := range []struct{
		class string
		damage int
	}{
		{"robot",68}, // Robot starts an encounter with 2x charge.
		{"psychic",68}, // Fixed double damage.
		{"psychopath",34},
	}{
		t.Run(tc.class,func(t *testing.T){
			g:=newGame()
			if err:=g.act("start","Injured Officer",tc.class);err!=nil {t.Fatal(err)}
			g.Player.Infections=1
			g.Player.HP=100
			g.startEncounter()
			g.Enemy.HP=1000
			if err:=g.act("special","","");err!=nil {t.Fatal(err)}
			if got:=1000-g.Enemy.HP;got!=tc.damage {
				t.Fatalf("special dealt %d, expected %d",got,tc.damage)
			}
			if tc.class=="psychopath" && g.Player.HP!=117 {t.Fatalf("healing was %d, expected 117",g.Player.HP)}
		})
	}
}

func TestClinicCostsScaleAndCurePenalties(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Clinic","robot");err!=nil {t.Fatal(err)}
	g.Player.Infections=2
	if g.Player.treatmentCost()!=1000 {t.Fatalf("unexpected cost %d",g.Player.treatmentCost())}
	g.Player.Money=999
	if err:=g.act("clinic","","");err==nil || !strings.Contains(err.Error(),"1000") {
		t.Fatalf("incorrect affordability guard: %v",err)
	}
	if g.Player.Infections!=2 || g.Player.Money!=999 {t.Fatal("rejected treatment changed officer")}
	g.Player.Money=1000
	if err:=g.act("clinic","","");err!=nil {t.Fatal(err)}
	if g.Player.Money!=0 || g.Player.Infections!=0 || g.Player.normalDamage()!=40 {
		t.Fatalf("treatment didn't cure penalties: %+v",g.Player)
	}
}

func TestUpdatedMedicalPricesMatchBrowser(t *testing.T) {
	page,err:=embeddedAssets.ReadFile("static/app.js")
	if err!=nil {t.Fatal(err)}
	js:=string(page)
	if !strings.Contains(js,"+ 5 CONDOMS • $250") || !strings.Contains(js,"p.money<250") {
		t.Fatal("Red Room condom label or affordability doesn't match Go")
	}
	if !strings.Contains(js,"Math.max(0,p.infections)*500") {
		t.Fatal("Medical Wing treatment cost isn't proportional to infections")
	}
	if !strings.Contains(js,"Math.min(45") {
		t.Fatal("Medical Wing is missing the infection damage cap")
	}
}
