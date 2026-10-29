package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testUTC(y int, m time.Month, d, hour int) time.Time {
	return time.Date(y,m,d,hour,0,0,0,time.UTC)
}

func TestRealCalendarDaysDoNotAdvanceOnClockOut(t *testing.T) {
	start:=testUTC(2026,time.October,29,21)
	g:=newGame()
	if err:=g.actAt("start","UTC Tester","psychic",start);err!=nil {t.Fatal(err)}
	if g.CampaignStart!="2026-10-29" || g.LastResetDate!="2026-10-29" ||
		g.Player.Days!=30 || g.Player.Bumfights!=fightsPerDay {
		t.Fatalf("start not anchored to UTC date: %+v",g)
	}
	g.Player.Bumfights=3
	g.Player.CocaineEncounters=2
	g.Player.PsychicEnergy = new(int)
	money:=g.Player.Money
	if err:=g.actAt("clockout","","",start);err!=nil {t.Fatal(err)}
	if g.Phase!="hq" || !g.ShiftEnded || g.Player.Bumfights!=0 || g.Player.Days!=30 ||
		g.Player.Money!=money+g.Player.shiftIncome() || g.Player.CocaineEncounters!=0 ||
		g.Player.psychicLeft()!=0 {
		t.Fatalf("clockout wrongly advanced UTC calendar or refilled PSI: %+v",g)
	}
	if err:=g.actAt("clockout","","",start.Add(time.Minute));err==nil {t.Fatal("second clockout awarded salary twice")}
	if err:=g.actAt("buy_handcuffs","","",start.Add(time.Minute));err==nil {t.Fatal("purchases allowed after clockout")}
	if err:=g.actAt("patrol","","",start.Add(time.Hour));err==nil {t.Fatal("patrol allowed after clockout")}
	if g.Player.Money!=money+g.Player.shiftIncome() {t.Fatal("duplicate income awarded")}
	if g.syncCalendar(testUTC(2026,time.October,29,23)) || !g.ShiftEnded {
		t.Fatal("same calendar day reset shift twice")
	}
	if !g.syncCalendar(testUTC(2026,time.October,30,0)) ||
		g.ShiftEnded || g.Player.Bumfights!=25 || g.Player.Days!=29 ||
		g.Player.psychicLeft()!=maxPsychicEnergy || g.LastResetDate!="2026-10-30" {
		t.Fatalf("midnight UTC reset not applied once: %+v",g)
	}
	if g.syncCalendar(testUTC(2026,time.October,30,10)) {t.Fatal("repeated UTC reset")}
	if g.syncCalendar(testUTC(2026,time.October,29,22)) ||
		g.Player.Days!=29 || g.Player.Bumfights!=25 {
		t.Fatal("backward clock rewound campaign")
	}
}

func TestRealCalendarSkipsMissedDaysWithoutBankingTurnsOrCash(t *testing.T) {
	start:=testUTC(2026,time.October,29,7)
	g:=newGame()
	if err:=g.actAt("start","Missed Days","psychic",start);err!=nil {t.Fatal(err)}
	g.Player.Bumfights=2
	g.Player.Money=700
	g.Player.Infections=1
	g.Player.CocaineEncounters=2
	g.startEncounter()
	if g.Phase!="combat" {t.Fatal("no combat")}
	zero:=0
	g.Player.PsychicEnergy=&zero
	if !g.syncCalendar(start.AddDate(0,0,4)) {
		t.Fatal("missed days did not advance")
	}
	if g.Phase!="hq" || g.Enemy!=nil || g.Player.Bumfights!=7 ||
		g.Player.Days!=26 || g.Player.Money!=700 || g.Player.Infections!=1 ||
		g.Player.CocaineEncounters!=0 || g.Player.psychicLeft()!=maxPsychicEnergy {
		t.Fatalf("missed days should skip expired fights and pay, reset buffs and restore PSI: %+v",g)
	}
	if !logContains(g,"3 missed UTC day(s)") {t.Fatalf("missing missed-day report: %v",g.Messages)}
}

func TestRealCalendarFinalDayCompletesAfterFullDay(t *testing.T) {
	start:=testUTC(2026,time.January,1,22)
	g:=newGame()
	if err:=g.actAt("start","Final Officer","robot",start);err!=nil {t.Fatal(err)}
	day30:=testUTC(2026,time.January,30,23)
	if !g.syncCalendar(day30) || g.Player.Days!=1 || g.Phase!="hq" {t.Fatalf("day30 must remain playable: %+v",g)}
	if err:=g.actAt("clockout","","",day30);err!=nil {t.Fatal(err)}
	if g.Phase!="hq" || !g.ShiftEnded || g.Player.Days!=1 {t.Fatal("day 30 clockout finished too early")}
	if !g.syncCalendar(testUTC(2026,time.January,31,0)) || g.Phase!="gameover" ||
		g.Player.Days!=0 || !strings.Contains(g.EndReason,"30 real-world days") {
		t.Fatalf("calendar did not finish after day30: %+v",g)
	}
	if g.syncCalendar(testUTC(2026,time.February,3,0)) {t.Fatal("completed campaign restarted")}
}

func TestUTCBoundaryRatherThanLocalTimeZone(t *testing.T) {
	location:=time.FixedZone("Toronto Standard",-5*3600)
	start:=time.Date(2026,time.November,1,18,45,0,0,location) // 23:45 UTC
	g:=newGame()
	if err:=g.actAt("start","Timezone Officer","robot",start);err!=nil {t.Fatal(err)}
	if g.CampaignStart!="2026-11-01" {t.Fatal("local start not normalized to UTC")}
	if !g.syncCalendar(start.Add(16*time.Minute)) || g.Player.Days!=29 {t.Fatal("00:01 UTC did not advance day")}
}

func TestOldCampaignIsMigratedWithoutWipingProgress(t *testing.T) {
	g:=newGame()
	if err:=json.Unmarshal([]byte(`{"version":"0.9.0","phase":"hq","player":{"name":"Legacy","class":"robot","days":17,"fightsLeft":8,"money":4567,"level":3,"hp":60,"handcuffs":2}}`),g);err!=nil {t.Fatal(err)}
	now:=testUTC(2026,time.October,29,17)
	if !g.syncCalendar(now) || g.Player.Days!=17 || g.Player.Bumfights!=8 ||
		g.Player.Money!=4567 || g.Player.Handcuffs!=2 || g.Player.HP!=60 ||
		g.Phase!="hq" || g.ShiftEnded || g.LastResetDate!="2026-10-29" {
		t.Fatalf("legacy progress not retained: %+v",g)
	}
	if g.CampaignStart!="2026-10-16" {t.Fatalf("legacy campaign didn't retain elapsed progress: %s",g.CampaignStart)}
	raw,err:=json.Marshal(g);if err!=nil {t.Fatal(err)}
	var reloaded Game
	if err:=json.Unmarshal(raw,&reloaded);err!=nil {t.Fatal(err)}
	if !validGame(&reloaded) || reloaded.syncCalendar(now) || reloaded.Player.Bumfights!=8 {
		t.Fatal("old save migrated twice on same date")
	}
	g.syncCalendar(now.AddDate(0,0,1))
	if g.Player.Days!=16 || g.Player.Bumfights!=25 {
		t.Fatal("old save did not start real-world reset after migration")
	}
}

func TestServerDailyRolloversPersistAcrossRequestAndRestart(t *testing.T) {
	dir:=t.TempDir()
	s,err:=newServer(dir);if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){s.db.Close()})
	now:=testUTC(2026,time.October,29,23)
	s.clock=func()time.Time{return now}
	handler:=s.handler()
	guest:=apiCall(t,handler,"GET","/api/state","",nil);assertHTTP(t,guest,200)
	cookie:=sessionFrom(t,guest)
	start:=apiCall(t,handler,"POST","/api/action",`{"action":"start","name":"Shared Officer","class":"psychic"}`,cookie)
	assertHTTP(t,start,200)
	closed:=apiCall(t,handler,"POST","/api/action",`{"action":"clockout"}`,cookie)
	assertHTTP(t,closed,200)
	var game Game
	if err:=json.Unmarshal(closed.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
	if !game.ShiftEnded || game.Player.Days!=30 {t.Fatal("did not lock current day")}
	before:=game.Player.Money
	assertHTTP(t,apiCall(t,handler,"POST","/api/action",`{"action":"clockout"}`,cookie),400)
	now=testUTC(2026,time.October,30,0)
	fresh:=apiCall(t,handler,"GET","/api/state","",cookie)
	assertHTTP(t,fresh,200)
	if err:=json.Unmarshal(fresh.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
	if game.ShiftEnded || game.Player.Days!=29 || game.Player.Bumfights!=25 || game.Player.Money!=before {
		t.Fatalf("GET failed to persist calendar reset or paid twice: %+v",game)
	}
	second,err:=newServer(dir);if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){second.db.Close()})
	second.clock=func()time.Time{return now}
	loaded:=apiCall(t,second.handler(),"GET","/api/state","",cookie)
	assertHTTP(t,loaded,200)
	if err:=json.Unmarshal(loaded.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
	if game.ShiftEnded || game.Player.Days!=29 || game.Player.Money!=before {
		t.Fatal("server restart lost current UTC day")
	}
}
