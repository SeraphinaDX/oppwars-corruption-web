package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCombatDefeatEndsCurrentUTCShiftNotCampaign(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Dusk","robot");err!=nil {t.Fatal(err)}
	start,_:=parseUTCDate(g.CampaignStart)
	beforeCash:=g.Player.Money
	g.startEncounter()
	enemy:=g.Enemy.Name
	g.Enemy.HP=99999
	g.Player.HP=0
	g.Player.Bumfights=12
	g.Player.CocaineEncounters=2
	g.Player.ClassPower=4
	if err:=g.act("special","","");err!=nil {t.Fatal(err)}
	if g.Phase!="hq" || !g.ShiftEnded || g.Player.Days!=30 || g.Player.Bumfights!=0 ||
		g.Enemy!=nil || g.EndReason!="" {
		t.Fatalf("defeat must end current calendar shift, not campaign: %+v",g)
	}
	if g.Player.HP!=g.Player.maxHP()/2 || g.Player.ClassPower!=1 ||
		g.Player.Money!=beforeCash || g.Player.Experience!=0 || g.Player.CocaineEncounters!=0 {
		t.Fatalf("defeat recovery or forfeiture incorrect: %+v",g.Player)
	}
	if !logContains(g,"defeated by "+enemy) || !logContains(g,"Forced clock-out") {t.Fatal("missing defeat report")}
	for _,action:=range []string{"patrol","clockout","buy_cocaine","nightlife","clinic"} {
		if err:=g.actAt(action,"","",start.Add(12*time.Hour));err==nil {
			t.Fatalf("allowed %s after losing shift",action)
		}
	}
	if !g.syncCalendar(start.AddDate(0,0,1)) || g.ShiftEnded || g.Phase!="hq" ||
		g.Player.Days!=29 || g.Player.Bumfights!=fightsPerDay {
		t.Fatalf("next UTC day didn't restore playable patrols: %+v",g)
	}
	if g.Player.Money!=beforeCash {t.Fatal("defeat granted next day's salary")}
}

func TestCombatDefeatKeepsInfectionsAndRechargesPsychicOnNextDay(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Psi","psychic");err!=nil {t.Fatal(err)}
	start,_:=parseUTCDate(g.CampaignStart)
	g.Player.Infections=2
	g.Player.HP=0
	zero:=0
	g.Player.PsychicEnergy=&zero
	g.startEncounter()
	g.Enemy.HP=99999
	before:=g.Player.Money
	if err:=g.act("fight","","");err!=nil {t.Fatal(err)}
	if g.Phase!="hq" || !g.ShiftEnded || g.Player.Days!=30 ||
		g.Player.psychicLeft()!=0 || g.Player.Infections!=2 || g.Player.Bumfights!=0 {
		t.Fatalf("defeat advanced psychic resources too soon: %+v",g)
	}
	if g.Player.Money!=before || g.Player.HP!=g.Player.maxHP()/2 {t.Fatal("defeat awarded cash or wrong HP")}
	if err:=g.act("clockout","","");err==nil {t.Fatal("defeat let officer collect wages")}
	g.syncCalendar(start.AddDate(0,0,1))
	if g.Player.psychicLeft()!=maxPsychicEnergy || g.Player.Bumfights!=minInfectedPatrols ||
		g.Player.Infections!=2 || g.Player.Days!=29 {
		t.Fatalf("new day didn't recharge Psychic or preserve infection: %+v",g.Player)
	}
	if err:=g.actAt("clockout","","",start.AddDate(0,0,1).Add(time.Hour));err!=nil {t.Fatal(err)}
	if g.Player.Money!=before+g.Player.shiftIncome() || !g.ShiftEnded {t.Fatal("normal next-day clockout failed")}
}

func TestLastDayDefeatCompletesOnlyAfterMidnight(t *testing.T) {
	g:=newGame()
	today:=utcDay(time.Now())
	if err:=g.actAt("start","Officer Last","psychopath",today.Add(time.Hour));err!=nil {t.Fatal(err)}
	g.CampaignStart=utcDate(today.AddDate(0,0,-29))
	g.LastResetDate=utcDate(today)
	g.Player.Days=1
	g.Player.HP=0
	g.startEncounter()
	g.Enemy.HP=99999
	if err:=g.actAt("fight","","",today.Add(2*time.Hour));err!=nil {t.Fatal(err)}
	if g.Phase!="hq" || !g.ShiftEnded || g.Player.Days!=1 {
		t.Fatalf("last-day defeat should only close shift: %+v",g)
	}
	if g.syncCalendar(today.AddDate(0,0,1))==false || g.Phase!="gameover" || g.Player.Days!=0 ||
		!strings.Contains(g.EndReason,"30 real-world days") {
		t.Fatalf("campaign did not finish at end of day 30: %+v",g)
	}
	if err:=g.actAt("patrol","","",today.AddDate(0,0,1));err==nil {t.Fatal("gameover accepted patrol")}
}

func TestAmbushDefeatForfeitsOnlyTodaysShift(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Ambush","robot");err!=nil {t.Fatal(err)}
	g.Player.HP=0
	g.startEncounter()
	g.losePatrol("Captain Grimstead's ambush")
	if g.Phase!="hq" || !g.ShiftEnded || g.Enemy!=nil || g.Player.Days!=30 ||
		g.Player.HP!=100 || g.Player.Bumfights!=0 {
		t.Fatalf("ambush defeat should lock today's shift: %+v",g)
	}
}

func TestDefeatShiftLockPersistsAcrossSaveReload(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Officer Persist","robot");err!=nil {t.Fatal(err)}
	g.startEncounter()
	g.Enemy.HP=99999
	g.Player.HP=0
	if err:=g.act("special","","");err!=nil {t.Fatal(err)}
	raw,err:=json.Marshal(g);if err!=nil {t.Fatal(err)}
	var saved Game
	if err:=json.Unmarshal(raw,&saved);err!=nil {t.Fatal(err)}
	if !validGame(&saved) || !saved.ShiftEnded || saved.Phase!="hq" || saved.Player.Days!=30 ||
		saved.Player.HP!=100 || saved.Player.Bumfights!=0 {
		t.Fatalf("defeat lock didn't persist: %+v",saved)
	}
}
