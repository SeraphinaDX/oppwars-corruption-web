package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserAPIAndPersistence(t *testing.T) {
	dir := t.TempDir()
	app, err := newServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := app.handler()
	state := httptest.NewRecorder()
	h.ServeHTTP(state, httptest.NewRequest("GET", "http://localhost/api/state", nil))
	if state.Code != 200 {
		t.Fatal("state failed", state.Code)
	}
	cookies := state.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly {
		t.Fatal("missing secure session cookie")
	}
	sessionCookie := cookies[0]
	body := bytes.NewBufferString(`{"action":"start","name":"Officer Epsilon","class":"psychic"}`)
	req := httptest.NewRequest("POST", "http://localhost/api/action", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("start failed: %d %s", res.Code, res.Body.String())
	}
	var current Game
	if err := json.Unmarshal(res.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	if current.Player.Name != "Officer Epsilon" || current.Phase != "hq" {
		t.Fatal("bad player", current.Player)
	}
	if _, err := os.Stat(filepath.Join(dir, sessionCookie.Value+".json")); err != nil {
		t.Fatal("save not written", err)
	}
	// Simulate process restart: a fresh server reloads game state from disk.
	second, _ := newServer(dir)
	fresh := httptest.NewRequest("GET", "http://localhost/api/state", nil)
	fresh.AddCookie(sessionCookie)
	loaded := httptest.NewRecorder()
	second.handler().ServeHTTP(loaded, fresh)
	var restored Game
	if err := json.Unmarshal(loaded.Body.Bytes(), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Player.Name != "Officer Epsilon" {
		t.Fatal("game did not reload", restored)
	}
	bad := httptest.NewRequest("POST", "http://localhost/api/action", bytes.NewBufferString(`{"action":"clockout"}`))
	bad.Header.Set("Content-Type", "application/json")
	bad.Header.Set("Origin", "https://attacker.example")
	rejected := httptest.NewRecorder()
	h.ServeHTTP(rejected, bad)
	if rejected.Code != http.StatusForbidden {
		t.Fatal("cross-origin mutation accepted")
	}
	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest("GET", "http://localhost/", nil))
	if asset.Code != 200 || !bytes.Contains(asset.Body.Bytes(), []byte("OPP")) {
		t.Fatalf("missing UI: %d", asset.Code)
	}
}
