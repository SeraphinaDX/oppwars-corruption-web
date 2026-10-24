package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func logContains(g *Game, fragment string) bool {
	for _, msg := range g.Messages {
		if strings.Contains(msg, fragment) {
			return true
		}
	}
	return false
}

func TestEnemyCatchphrasesAreDistinctAndSaved(t *testing.T) {
	if len(enemyCatchphrases) != len(enemies) {
		t.Fatalf("need catchphrases for each enemy: %d enemies, %d sets",len(enemies),len(enemyCatchphrases))
	}
	for _, name := range enemies {
		lines := enemyCatchphrases[name]
		if len(lines) < 3 {
			t.Fatalf("%q needs multiple catchphrases",name)
		}
		previous := nextCatchphrase(name, "")
		if !slices.Contains(lines,previous) {
			t.Fatalf("%q says unknown catchphrase %q", name, previous)
		}
		for i:=0;i<40;i++ {
			current:=nextCatchphrase(name,previous)
			if current==previous || !slices.Contains(lines,current) {
				t.Fatalf("%q repeated an identical catchphrase or emitted an unknown one",name)
			}
			previous=current
		}
	}
	g:=newGame()
	if err:=g.act("start","Officer Adder","robot");err!=nil {t.Fatal(err)}
	g.startEncounter()
	if g.Enemy==nil || g.Enemy.Catchphrase=="" {
		t.Fatal("spawned enemy must have a catchphrase")
	}
	if !logContains(g,g.Enemy.Catchphrase) || !logContains(g,"Officer Adder runs into") {
		t.Fatalf("encounter dialogue missing: %v",g.Messages)
	}
	data,err:=json.Marshal(g)
	if err!=nil {t.Fatal(err)}
	var restored Game
	if err:=json.Unmarshal(data,&restored);err!=nil {t.Fatal(err)}
	if restored.Enemy==nil || restored.Enemy.Catchphrase!=g.Enemy.Catchphrase {
		t.Fatal("enemy catchphrase lost during save/reload")
	}
	// Saves created before v0.4.0 don't need a catchphrase field.
	older := `{"phase":"combat","player":{"name":"Old","class":"robot","hp":200,"level":1},"enemy":{"name":"Rogue District Thug","hp":100,"maxHP":100}}`
	var legacy Game
	if err:=json.Unmarshal([]byte(older),&legacy);err!=nil || !validGame(&legacy) {
		t.Fatal("old combat save rejected",err)
	}
}

func TestCombatReportsNamePlayerAndOpponent(t *testing.T) {
	g:=newGame()
	if err:=g.act("start","Constable Vega","robot");err!=nil {t.Fatal(err)}
	g.startEncounter()
	enemy:=g.Enemy.Name
	g.Enemy.HP=1
	if err:=g.act("special","","");err!=nil {t.Fatal(err)}
	for _,required:=range []string{
		fmt.Sprintf("Constable Vega uses Charged Shot"),
		fmt.Sprintf("against %s",enemy),
		fmt.Sprintf("Constable Vega defeats %s!",enemy),
	} {
		if !logContains(g,required) {
			t.Errorf("missing named combat report %q: %v",required,g.Messages)
		}
	}
	g.startEncounter()
	if err:=g.act("retreat","","");err!=nil {t.Fatal(err)}
	if !logContains(g,"Constable Vega retreats") {
		t.Fatalf("retreat report must show officer name: %v",g.Messages)
	}
}

func TestRedRoomWordingAndOriginalEffects(t *testing.T) {
	html,err:=embeddedAssets.ReadFile("static/app.js")
	if err!=nil {t.Fatal(err)}
	copy:=string(html)
	if !strings.Contains(copy,"BANG A PROSTITUTE • $100") {
		t.Fatal("Red Room action label was not updated")
	}
	if strings.Contains(copy,"infection roll") || strings.Contains(copy,"This version uses venue") || strings.Contains(copy,"NIGHTLIFE BREAK") {
		t.Fatal("Red Room still has developer-facing copy")
	}
	if !strings.Contains(copy,"escapeHTML(e.catchphrase)") {
		t.Fatal("catchphrases must be escaped in the browser")
	}
	g:=newGame()
	if err:=g.act("start","Officer Avery","psychopath");err!=nil {t.Fatal(err)}
	g.Player.HP=50
	g.Player.Condoms=1
	before:=g.Player.Money
	if err:=g.act("nightlife","","");err!=nil {t.Fatal(err)}
	if g.Player.Money!=before-100 || g.Player.Condoms!=0 || g.Player.HP!=200 || g.Player.Infections!=0 {
		t.Fatalf("the Red Room gameplay unexpectedly changed: %+v",g.Player)
	}
	if !logContains(g,"Officer Avery leaves the Red Room") {
		t.Fatal("Red Room reports should name the officer")
	}
}
