package main

import (
 "encoding/json"
 "testing"
)

func TestPsychicMeterDailyLimitAndRecharge(t *testing.T) {
 g:=newGame()
 if err:=g.act("start","Psi","psychic");err!=nil {t.Fatal(err)}
 if g.Player.psychicLeft()!=300 {t.Fatal("missing initial meter")}
 for i:=0;i<12;i++ {
   g.startEncounter()
   g.Enemy.HP=100000
   if err:=g.act("special","","");err!=nil {t.Fatal(err)}
   if want:=300-(i+1)*25;g.Player.psychicLeft()!=want {t.Fatalf("meter=%d want=%d",g.Player.psychicLeft(),want)}
   if err:=g.act("retreat","","");err!=nil {t.Fatal(err)}
 }
 g.startEncounter()
 g.Enemy.HP=100000
 before:=g.Enemy.HP
 if err:=g.act("special","","");err==nil {t.Fatal("blast available without energy")}
 if g.Enemy.HP!=before {t.Fatal("rejected blast damaged enemy")}
 if err:=g.act("retreat","","");err!=nil {t.Fatal(err)}
 if err:=g.act("clockout","","");err!=nil {t.Fatal(err)}
 if g.Player.psychicLeft()!=0 || !g.ShiftEnded {t.Fatal("clockout must not recharge PSI before UTC reset")}
 start,_:=parseUTCDate(g.CampaignStart)
 g.syncCalendar(start.AddDate(0,0,1))
 if g.Player.psychicLeft()!=300 || g.ShiftEnded {t.Fatal("new UTC day didn't recharge PSI")}
}

func TestRobotChargeResetsAfterShotAndCaps(t *testing.T) {
 g:=newGame();_ = g.act("start","Bot","robot")
 g.startEncounter()
 if g.Player.ClassPower!=2 {t.Fatalf("initial charge %d",g.Player.ClassPower)}
 g.Enemy.HP=100000
 if err:=g.act("special","","");err!=nil {t.Fatal(err)}
 if g.Player.ClassPower!=1 || g.Enemy.HP!=99920 {t.Fatalf("shot %d charge %d",g.Enemy.HP,g.Player.ClassPower)}
 for i:=0;i<8;i++ {
   if err:=g.act("fight","","");err!=nil {t.Fatal(err)}
   if g.Player.ClassPower>4 {t.Fatalf("charge exceeded cap %d",g.Player.ClassPower)}
 }
 if g.Player.ClassPower!=4 {t.Fatalf("should cap at 4: %d",g.Player.ClassPower)}
}

func TestPsychicEnergySaveZeroAndLegacy(t *testing.T) {
 var old Game
 if err:=json.Unmarshal([]byte(`{"phase":"hq","player":{"name":"Legacy","class":"psychic","hp":200,"level":1}}`),&old);err!=nil {t.Fatal(err)}
 old.Player.normalizeLegacyClassResources()
 if old.Player.psychicLeft()!=300 {t.Fatal("legacy character was not initialized")}
 zero:=0;old.Player.PsychicEnergy=&zero
 data,err:=json.Marshal(old);if err!=nil {t.Fatal(err)}
 var restored Game
 if err:=json.Unmarshal(data,&restored);err!=nil {t.Fatal(err)}
 restored.Player.normalizeLegacyClassResources()
 if restored.Player.psychicLeft()!=0 {t.Fatal("reloading replenished an empty meter")}
}
