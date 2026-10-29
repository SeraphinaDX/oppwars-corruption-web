package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func apiCall(t *testing.T, handler http.Handler, method,path,body string,cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req:=httptest.NewRequest(method,"http://localhost"+path,bytes.NewBufferString(body))
	if body!="" {req.Header.Set("Content-Type","application/json")}
	if cookie!=nil {req.AddCookie(cookie)}
	w:=httptest.NewRecorder()
	handler.ServeHTTP(w,req)
	return w
}
func sessionFrom(t *testing.T,w *httptest.ResponseRecorder)*http.Cookie {
	t.Helper()
	for _,c:=range w.Result().Cookies() {if c.Name=="oppwars_session" {return c}}
	t.Fatal("no session cookie")
	return nil
}
func assertHTTP(t *testing.T,w *httptest.ResponseRecorder,status int) {
	t.Helper()
	if w.Code!=status {t.Fatalf("expected %d, got %d: %s",status,w.Code,w.Body.String())}
}

func TestMultiplayerAccountClaimAndCrossBrowserLogin(t *testing.T) {
	dir:=t.TempDir()
	s,err:=newServer(dir)
	if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){s.db.Close()})
	h:=s.handler()
	// An existing guest officer can be attached to the account.
	guest:=apiCall(t,h,"GET","/api/state","",nil)
	assertHTTP(t,guest,200)
	guestCookie:=sessionFrom(t,guest)
	created:=apiCall(t,h,"POST","/api/action",`{"action":"start","name":"Captain Violet","class":"psychic"}`,guestCookie)
	assertHTTP(t,created,200)
	reg:=apiCall(t,h,"POST","/api/auth/register",`{"username":"Violet12","password":"very-long-test-password"}`,guestCookie)
	assertHTTP(t,reg,200)
	firstCookie:=sessionFrom(t,reg)
	if !firstCookie.HttpOnly || firstCookie.SameSite!=http.SameSiteStrictMode {t.Fatal("unsafe account cookie")}
	if _,err:=os.Stat(filepath.Join(dir,guestCookie.Value+".json")); !os.IsNotExist(err) {
		t.Fatal("claimed guest save still accessible under old session cookie")
	}
	one:=apiCall(t,h,"GET","/api/state","",firstCookie)
	assertHTTP(t,one,200)
	var game Game
	if err:=json.Unmarshal(one.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
	if game.Player.Name!="Captain Violet" || game.Player.Class!="psychic" {t.Fatalf("did not attach guest officer: %+v",game)}
	// Anonymous players cannot read the shared precinct or private account data.
	assertHTTP(t,apiCall(t,h,"GET","/api/precinct","",nil),401)
	roster:=apiCall(t,h,"GET","/api/precinct","",firstCookie)
	assertHTTP(t,roster,200)
	var snapshot precinctSnapshot
	if err:=json.Unmarshal(roster.Body.Bytes(),&snapshot);err!=nil {t.Fatal(err)}
	if len(snapshot.Roster)!=1 || snapshot.Roster[0].Username!="Violet12" || !snapshot.Roster[0].Online ||
		snapshot.Roster[0].OfficerName!="Captain Violet" || len(snapshot.Leaderboard)!=1 {
		t.Fatalf("online roster incorrect: %+v",snapshot)
	}
	if strings.Contains(roster.Body.String(),"very-long-test-password") ||
		strings.Contains(roster.Body.String(),"password_hash") ||
		strings.Contains(roster.Body.String(),"\"money\"") {
		t.Fatal("precinct leaked credentials or private game economy")
	}
	public:=apiCall(t,h,"GET","/api/officers/violet12","",firstCookie)
	assertHTTP(t,public,200) // Usernames are case insensitive.
	if !strings.Contains(public.Body.String(),"Captain Violet") {t.Fatal("officer not found")}
	// An authenticated officer may sign in from a different browser and keep
	// one canonical game and inventory.
	secondLogin:=apiCall(t,h,"POST","/api/auth/login",`{"username":"Violet12","password":"very-long-test-password"}`,nil)
	assertHTTP(t,secondLogin,200)
	secondCookie:=sessionFrom(t,secondLogin)
	if secondCookie.Value==firstCookie.Value {t.Fatal("login reused session token")}
	secondSave:=apiCall(t,h,"GET","/api/state","",secondCookie)
	assertHTTP(t,secondSave,200)
	var restored Game
	if err:=json.Unmarshal(secondSave.Body.Bytes(),&restored);err!=nil {t.Fatal(err)}
	if restored.Player.Name!="Captain Violet" || restored.Player.PsychicEnergy==nil {t.Fatal("second browser lost account save")}
	assertHTTP(t,apiCall(t,h,"POST","/api/action",`{"action":"buy_handcuffs"}`,secondCookie),200)
	firstUpdated:=apiCall(t,h,"GET","/api/state","",firstCookie)
	assertHTTP(t,firstUpdated,200)
	if err:=json.Unmarshal(firstUpdated.Body.Bytes(),&game);err!=nil {t.Fatal(err)}
	if game.Player.Handcuffs!=1 {t.Fatal("first browser did not see second browser purchase")}
	// Log out only this browser's token.
	assertHTTP(t,apiCall(t,h,"POST","/api/auth/logout","",secondCookie),200)
	assertHTTP(t,apiCall(t,h,"GET","/api/precinct","",secondCookie),401)
	assertHTTP(t,apiCall(t,h,"GET","/api/precinct","",firstCookie),200)
	// Restart server: database credentials and game ownership both survive.
	s2,err:=newServer(dir);if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){s2.db.Close()})
	h2:=s2.handler()
	fresh:=apiCall(t,h2,"GET","/api/state","",firstCookie)
	assertHTTP(t,fresh,200)
	var again Game
	if err:=json.Unmarshal(fresh.Body.Bytes(),&again);err!=nil {t.Fatal(err)}
	if again.Player.Handcuffs!=1 || again.Player.Name!="Captain Violet" {t.Fatal("server restart lost account game")}
}

func TestMultiplayerAccountValidationAndIsolation(t *testing.T) {
	s,err:=newServer(t.TempDir());if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){s.db.Close()})
	h:=s.handler()
	w:=apiCall(t,h,"GET","/api/state","",nil);assertHTTP(t,w,200)
	cookie:=sessionFrom(t,w)
	for _,input:=range []string{
		`{"username":"x","password":"long-enough-password"}`,
		`{"username":"ValidName","password":"short"}`,
		`{"username":"<script>","password":"long-enough-password"}`,
		`{"username":"ValidName","password":"long-enough-password","role":"admin"}`,
		`{"username":"ValidName","password":"long-enough-password"}{}`,
	}{
		r:=apiCall(t,h,"POST","/api/auth/register",input,cookie)
		if r.Code==200 {t.Fatalf("registered invalid credentials %s",input)}
	}
	reg:=apiCall(t,h,"POST","/api/auth/register",`{"username":"MuffinCop","password":"the-password-is-long"}`,cookie)
	assertHTTP(t,reg,200)
	authCookie:=sessionFrom(t,reg)
	assertHTTP(t,apiCall(t,h,"POST","/api/auth/register",`{"username":"AnotherCop","password":"the-password-is-long"}`,authCookie),409)
	assertHTTP(t,apiCall(t,h,"POST","/api/auth/login",`{"username":"MuffinCop","password":"wrong-password"}`,nil),401)
	assertHTTP(t,apiCall(t,h,"GET","/api/officers/doesnotexist","",authCookie),404)
	req:=httptest.NewRequest("POST","http://localhost/api/auth/logout",nil)
	req.Header.Set("Origin","https://attacker.example")
	req.AddCookie(authCookie)
	x:=httptest.NewRecorder();h.ServeHTTP(x,req)
	assertHTTP(t,x,403)
	assertHTTP(t,apiCall(t,h,"GET","/api/auth","",authCookie),200)
}

func TestCaddySecureCookieSetting(t *testing.T) {
	s,err:=newServer(t.TempDir());if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){s.db.Close()})
	s.secureCookies=true
	w:=apiCall(t,s.handler(),"GET","/api/state","",nil)
	assertHTTP(t,w,200)
	if !sessionFrom(t,w).Secure {t.Fatal("behind-Caddy mode must set Secure guest cookies")}
	reg:=apiCall(t,s.handler(),"POST","/api/auth/register",
		`{"username":"BehindCaddy","password":"a-long-test-password"}`,sessionFrom(t,w))
	assertHTTP(t,reg,200)
	if !sessionFrom(t,reg).Secure {t.Fatal("behind-Caddy mode must set Secure account cookies")}
}
