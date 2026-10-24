package main

import (
 "strings"
 "testing"
)

// Prices correspond to what the browser shows and must be consistent with
// the comparatively small benefits provided by these short-lived buffs.
func TestPropertyRoomPrices(t *testing.T) {
 cases := []struct {
  action string
  item string
  price int
  count func(*Player) int
 }{
  {"buy_cocaine","cocaine",cocainePrice,func(p *Player)int{return p.Cocaine}},
  {"buy_fentanyl","fentanyl",fentanylPrice,func(p *Player)int{return p.Fentanyl}},
  {"buy_lsd","lsd",lsdPrice,func(p *Player)int{return p.LSD}},
 }
 for _,tc:=range cases {
  t.Run(tc.item,func(t *testing.T){
   g:=newGame()
   if err:=g.act("start","Price Tester","robot");err!=nil {t.Fatal(err)}
   startMoney:=g.Player.Money
   if err:=g.act(tc.action,"","");err!=nil {t.Fatal(err)}
   if got:=startMoney-g.Player.Money;got!=tc.price {t.Fatalf("charged $%d, expected $%d",got,tc.price)}
   if got:=tc.count(&g.Player);got!=1 {t.Fatalf("received %d units, expected 1",got)}
   g.Player.RenegadeRank=10
   if err:=g.act(tc.action,"","");err!=nil {t.Fatal(err)}
   if got:=tc.count(&g.Player);got!=3 {t.Fatalf("renegade purchase left %d units, expected 3",got)}
   if got:=startMoney-g.Player.Money;got!=2*tc.price {t.Fatalf("wrong renegade price: %d",got)}
   g.Player.Money=tc.price-1
   if err:=g.act(tc.action,"","");err==nil {t.Fatal("insufficient-funds purchase succeeded")}
  })
 }
}

func TestVersionBumpAndLegacySaves(t *testing.T) {
 if gameVersion!="0.3.0" {t.Fatalf("release is %s, expected 0.3.0",gameVersion)}
 if g:=newGame();g.Version!=gameVersion {t.Fatal("new game has stale version")}
 // The browser must show the release from the server, rather than an
 // independently hard-coded version string.
 page,err:=embeddedAssets.ReadFile("static/app.js")
 if err!=nil {t.Fatal(err)}
 if !strings.Contains(string(page),"g.version") {t.Fatal("browser version not linked to server")}
}
