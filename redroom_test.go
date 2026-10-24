package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedRoomResultRequiresContinueAndPreservesConsequences(t *testing.T) {
	g := newGame()
	if err := g.act("start", "Officer Avery", "psychopath"); err != nil { t.Fatal(err) }
	g.Player.HP = 50
	g.Player.Condoms = 1
	before := g.Player.Money
	if err := g.act("nightlife", "", ""); err != nil { t.Fatal(err) }
	if g.Phase != "redroom_result" || g.RedRoom == nil || !validGame(g) {
		t.Fatalf("nightlife didn't show a valid result phase: %+v",g)
	}
	r := *g.RedRoom
	if r.InfectionGained || !r.Protected || r.TotalInfections != 0 {
		t.Fatalf("protected encounter produced unexpected infection: %+v",r)
	}
	if !strings.Contains(r.Scene, "Officer Avery") || strings.Contains(r.Scene, "%!") {
		t.Fatalf("result story was not formatted correctly: %q",r.Scene)
	}
	if g.Player.HP != 200 || r.HealthRecovered != 150 || g.Player.Condoms != 0 || r.Cost != 100 || g.Player.Money != before-100 {
		t.Fatalf("Red Room did not apply original cost/healing/protection: %+v %+v",g.Player,r)
	}
	for _,action := range []string{"nightlife","clockout","patrol","clinic","buy_condoms"} {
		if err := g.act(action,"","");err == nil {
			t.Fatalf("allowed %s while encounter report was showing",action)
		}
	}
	if g.Player.Money != before-100 || g.Player.HP != 200 || g.Player.Infections != 0 || *g.RedRoom != r {
		t.Fatal("denied actions changed the outcome")
	}
	if err := g.act("continue","","");err != nil {t.Fatal(err)}
	if g.Phase != "hq" || g.RedRoom != nil {
		t.Fatalf("continue didn't return to HQ: %+v",g)
	}
	if err := g.act("continue","","");err == nil {t.Fatal("continue accepted twice")}
}

func TestRedRoomUnprotectedInfectionReceipt(t *testing.T) {
	for i:=0;i<20;i++ {
		g:=newGame()
		if err:=g.act("start","Night Officer","psychic");err != nil {t.Fatal(err)}
		g.Player.Infections=2 // Existing infections must not be mistaken for new ones.
		before:=g.Player.Money
		if err:=g.act("nightlife","","");err!=nil {t.Fatal(err)}
		if g.RedRoom==nil || g.RedRoom.Protected {t.Fatal("expected unprotected encounter")}
		after:=g.Player.Infections
		if after!=2 && after!=3 {t.Fatalf("bad infection count %d",after)}
		if g.RedRoom.InfectionGained!=(after==3) || g.RedRoom.TotalInfections!=after {
			t.Fatalf("receipt is not based on actual disease outcome: %+v",g.RedRoom)
		}
		if g.Player.Money!=before-100 {t.Fatal("encounter charged wrong amount")}
	}
}

func TestRedRoomRandomStoryFormat(t *testing.T) {
	if len(redRoomScenes)<3 {t.Fatal("not enough story variation")}
	for i:=0;i<50;i++ {
		story:=newRedRoomScene("An <Officer>")
		if !strings.Contains(story,"An <Officer>") || strings.Contains(story,"%!") || len(story)<60 {
			t.Fatalf("broken scene: %q",story)
		}
	}
}

func TestRedRoomResultSurvivesServerRestart(t *testing.T) {
	dir:=t.TempDir()
	s,err:=newServer(dir);if err!=nil {t.Fatal(err)}
	h:=s.handler()
	ask:=func(handler http.Handler, method, path, payload string, cookie *http.Cookie)(int,Game) {
		t.Helper()
		req:=httptest.NewRequest(method,"http://localhost"+path,bytes.NewBufferString(payload))
		if method=="POST" {req.Header.Set("Content-Type","application/json")}
		if cookie!=nil {req.AddCookie(cookie)}
		res:=httptest.NewRecorder()
		handler.ServeHTTP(res,req)
		var game Game
		if res.Code==200 {
			if err:=json.Unmarshal(res.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
		}
		return res.Code,game
	}
	req:=httptest.NewRequest("GET","http://localhost/api/state",nil)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,req)
	if w.Code!=200 || len(w.Result().Cookies())==0 {t.Fatal("session cookie missing")}
	cookie:=w.Result().Cookies()[0]
	if code,_:=ask(h,"POST","/api/action",`{"action":"start","name":"Officer Resave","class":"robot"}`,cookie);code!=200 {t.Fatalf("start http %d",code)}
	if code,_:=ask(h,"POST","/api/action",`{"action":"buy_condoms"}`,cookie);code!=200 {t.Fatalf("condom purchase http %d",code)}
	code,g:=ask(h,"POST","/api/action",`{"action":"nightlife"}`,cookie)
	if code!=200 || g.Phase!="redroom_result" || g.RedRoom==nil {t.Fatalf("no result %d %+v",code,g)}
	initial:=*g.RedRoom
	money:=g.Player.Money
	condoms:=g.Player.Condoms
	// Fresh process, same save directory and session cookie.
	s2,err:=newServer(dir);if err!=nil {t.Fatal(err)}
	code,reloaded:=ask(s2.handler(),"GET","/api/state","",cookie)
	if code!=200 || reloaded.Phase!="redroom_result" || reloaded.RedRoom==nil || *reloaded.RedRoom!=initial {
		t.Fatalf("reload skipped or rerolled encounter: %d %+v",code,reloaded)
	}
	if reloaded.Player.Money!=money || reloaded.Player.Condoms!=condoms {t.Fatal("reloading changed money or inventory")}
	if code,_:=ask(s2.handler(),"POST","/api/action",`{"action":"nightlife"}`,cookie);code!=400 {
		t.Fatalf("duplicate encounter accepted: %d",code)
	}
	code,returned:=ask(s2.handler(),"POST","/api/action",`{"action":"continue"}`,cookie)
	if code!=200 || returned.Phase!="hq" || returned.RedRoom!=nil || returned.Player.Money!=money {
		t.Fatalf("continue failed: %d %+v",code,returned)
	}
	if code,_:=ask(s2.handler(),"POST","/api/action",`{"action":"continue"}`,cookie);code!=400 {
		t.Fatal("double continue accepted")
	}
}

func TestRedRoomScreenRendersResultsSafely(t *testing.T) {
	page,err:=embeddedAssets.ReadFile("static/app.js");if err!=nil {t.Fatal(err)}
	js:=string(page)
	for _,needle:=range []string{
		"function redRoomResultPage(g)", "g.phase==='redroom_result'",
		"escapeHTML(r.scene)", "r.infectionGained", "r.totalInfections",
		"actionButton('continue'",
	} {
		if !strings.Contains(js,needle) {t.Fatalf("Red Room result UI missing %q",needle)}
	}
}
