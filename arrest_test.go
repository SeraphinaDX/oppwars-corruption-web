package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func makeArrestGame(t *testing.T, class string) *Game {
	t.Helper()
	g := newGame()
	if err := g.act("start", "Officer Ada", class); err != nil { t.Fatal(err) }
	if err := g.act("buy_handcuffs", "", ""); err != nil { t.Fatal(err) }
	g.startEncounter()
	return g
}

func TestHandcuffsPurchaseAndPersistence(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Officer Ada", "robot"); err != nil { t.Fatal(err) }
	g.Player.Money = handcuffsPrice - 1
	if err := g.act("buy_handcuffs", "", ""); err == nil { t.Fatal("allowed purchase without enough money") }
	if g.Player.Handcuffs != 0 || g.Player.Money != handcuffsPrice-1 { t.Fatal("failed purchase changed inventory") }
	g.Player.Money = 200
	if err := g.act("buy_handcuffs", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Handcuffs != 1 || g.Player.Money != 100 {
		t.Fatalf("wrong cuff purchase balance %+v",g.Player)
	}
	g.Player.RenegadeRank = renegadeShopRank
	if err := g.act("buy_handcuffs", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Handcuffs != 3 || g.Player.Money != 0 { t.Fatalf("two-for-one perk failed %+v",g.Player) }
	g.Player.Handcuffs = maxInventoryEach - 1
	g.Player.Money = handcuffsPrice
	if err := g.act("buy_handcuffs", "", ""); err == nil {t.Fatal("permitted purchase past stock limit")}
	if g.Player.Handcuffs != maxInventoryEach-1 || g.Player.Money != handcuffsPrice {
		t.Fatal("rejected bulk purchase modified money/stock")
	}
	g.Player.RenegadeRank = 0
	if err := g.act("buy_handcuffs", "", ""); err != nil { t.Fatal(err) }
	if g.Player.Handcuffs != maxInventoryEach || g.Player.Money != 0 {t.Fatal("single cuff didn't fit inventory cap")}
	if err := g.act("buy_handcuffs", "", ""); err == nil { t.Fatal("bought past cap") }
	raw,err := json.Marshal(g);if err!=nil {t.Fatal(err)}
	var restored Game
	if err:=json.Unmarshal(raw,&restored);err!=nil {t.Fatal(err)}
	if restored.Player.Handcuffs != maxInventoryEach || !validGame(&restored) { t.Fatal("handcuffs lost during save/reload") }
	var legacy Game
	if err:=json.Unmarshal([]byte(`{"phase":"hq","player":{"name":"Old","class":"robot","hp":200,"level":1}}`),&legacy);err!=nil {t.Fatal(err)}
	if !validGame(&legacy) || legacy.Player.Handcuffs!=0 {t.Fatal("old saves need to start with zero cuffs")}
}

func TestArrestRequiresCuffsAndWeakenedEnemy(t *testing.T) {
	g := makeArrestGame(t, "robot")
	enemyName:=g.Enemy.Name
	beforeCash := g.Player.Money
	beforeTurns := g.Turn
	if err:=g.act("arrest","","");err==nil || !strings.Contains(err.Error(),"50%") {t.Fatalf("healthy target arrested: %v",err)}
	if g.Player.Handcuffs!=1 || g.Player.Money!=beforeCash || g.Enemy.Name!=enemyName || g.Turn!=beforeTurns {
		t.Fatal("rejected early arrest consumed resources or altered combat")
	}
	g.Enemy.HP=g.Enemy.MaxHP/2+1
	if err:=g.act("arrest","","");err==nil {t.Fatal("arrest allowed with target above 50%")}
	g.Player.Handcuffs=0
	g.Enemy.HP=g.Enemy.MaxHP/2
	if err:=g.act("arrest","","");err==nil {t.Fatal("arrest allowed without cuffs")}
	if g.Enemy==nil || g.Phase!="combat" || g.Player.ParagonRank!=0 {t.Fatal("invalid arrest finished combat")}
	g.Player.Handcuffs=1
	g.Enemy.HP=0
	if err:=g.act("arrest","","");err==nil {t.Fatal("arrest allowed on defeated enemy")}
	if g.Player.Handcuffs!=1 {t.Fatal("invalid arrest consumed cuffs")}
	g.Enemy.HP=g.Enemy.MaxHP/2
	g.Player.CocaineEncounters=2
	if err:=g.act("arrest","","");err!=nil {t.Fatal(err)}
	if g.Enemy!=nil || g.Phase!="hq" || g.Player.Handcuffs!=0 {
		t.Fatalf("arrest did not end combat: %+v",g)
	}
	if g.Player.ParagonRank!=1 || g.Player.RenegadeRank!=0 {
		t.Fatalf("wrong karma after arrest %+v",g.Player)
	}
	if g.Player.Money!=beforeCash+g.Player.victoryPay()/2 ||
		g.Player.Experience!=125 || g.Player.Bumfights!=fightsPerDay-1 ||
		g.Player.CocaineEncounters!=1 {
		t.Fatalf("incorrect arrest rewards or effects: %+v",g.Player)
	}
	if !logContains(g,"Officer Ada arrests "+enemyName) {t.Fatal("arrest message missing officer and enemy names")}
	if err:=g.act("arrest","","");err==nil {t.Fatal("arrest accepted outside combat")}
}

func TestArrestHalfXPAndPromotion(t *testing.T) {
	g:=makeArrestGame(t,"psychic")
	g.Player.Experience=950
	g.Player.CocaineEncounters=1
	g.Enemy.HP=g.Enemy.MaxHP/2
	// level 1 grants 200 base + 50 XP Rush: arrest pays 100 + 25.
	if err:=g.act("arrest","","");err!=nil {t.Fatal(err)}
	if g.Player.Experience!=1075 || g.Player.Level!=2 || g.Player.CocaineEncounters!=0 {
		t.Fatalf("arrest bonus XP or promotion failed: %+v",g.Player)
	}
	if !logContains(g,"+125 XP (25 bonus)") {t.Fatal("arrest XP/bonus not reported correctly")}
	if g.Player.psychicLeft()!=maxPsychicEnergy {t.Fatal("arrest spent psychic energy")}
}

func TestEliminationAwardsOnlyRenegade(t *testing.T) {
	for _,class:=range []string{"psychic","robot","psychopath"} {
		t.Run(class,func(t *testing.T){
			g:=newGame()
			if err:=g.act("start","Officer Kaya",class);err!=nil {t.Fatal(err)}
			g.Player.ParagonRank=2
			g.startEncounter()
			g.Enemy.HP=1
			if err:=g.act("special","","");err!=nil {t.Fatal(err)}
			if g.Phase!="hq" || g.Player.Handcuffs!=0 || g.Player.Money!=starterMoney+300 || g.Player.Experience!=200 {
				t.Fatalf("lethal victory rewards changed: %+v",g)
			}
			if g.Player.ParagonRank!=2 || g.Player.RenegadeRank!=1 {
				t.Fatalf("lethal victory karma wrong %+v",g.Player)
			}
			if !logContains(g,"+1 Renegade") {t.Fatal("Renegade message missing")}
			if err:=g.act("special","","");err==nil {t.Fatal("allowed killing twice")}
		})
	}
}

func TestRetreatAndDefeatDoNotAwardCombatKarma(t *testing.T) {
	g:=makeArrestGame(t,"robot")
	if err:=g.act("retreat","","");err!=nil {t.Fatal(err)}
	if g.Player.ParagonRank!=0 || g.Player.RenegadeRank!=0 || g.Player.Handcuffs!=1 {
		t.Fatal("retreat should not affect karma/cuffs")
	}
	g.startEncounter()
	g.Enemy.HP=100000
	g.Player.HP=0
	if err:=g.act("special","","");err!=nil {t.Fatal(err)}
	if g.Player.ParagonRank!=0 || g.Player.RenegadeRank!=0 || g.Player.Handcuffs!=1 {
		t.Fatal("defeat should not affect combat karma/cuffs")
	}
}

func TestArrestButtonAndPropertyRoomStockPresent(t *testing.T) {
	page,err:=embeddedAssets.ReadFile("static/app.js");if err!=nil {t.Fatal(err)}
	s:=string(page)
	for _,check:=range []string{
		"actionButton('arrest'", "e.hp>e.maxHP*0.5", "actionButton('buy_handcuffs'",
		"Hand cuffs", "row('Hand cuffs',p.handcuffs||0)", "actionButton('retreat'",
	} {
		if !strings.Contains(s,check) {t.Errorf("missing arrest UI fragment %q",check)}
	}
}
