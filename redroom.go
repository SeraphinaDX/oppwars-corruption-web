// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
package main

import (
	"fmt"
	"math/rand/v2"
)

// RedRoomResult is a one-encounter receipt, not a recurring effect.
// It survives reloads so players must see the outcome before continuing.
type RedRoomResult struct {
	Scene            string `json:"scene"`
	Protected        bool   `json:"protected"`
	InfectionGained  bool   `json:"infectionGained"`
	TotalInfections  int    `json:"totalInfections"`
	HealthRecovered  int    `json:"healthRecovered"`
	Cost             int    `json:"cost"`
}

// Keep the Red Room story non-graphic, in-universe, and varied.
// Infection status is generated from the actual result, never from these lines.
var redRoomScenes = []string{
	"The doorman waves %s past the velvet rope. Inside, a jukebox wheezes through an ancient love song while the bartender pretends not to recognize an officer in uniform. After a discreet private visit, %s heads back into the neon night.",
	"%s follows a hostess past the dance floor, narrowly avoiding a familiar face from Internal Affairs. Behind the red curtains, the evening stays private. By the time the taxi horns start outside, %s is ready to return to duty.",
	"A hostess welcomes %s to the Red Room and promises discretion. The music is terrible, the wallpaper is worse, and somebody has parked a patrol car in the loading zone. %s slips out after an eventful evening.",
	"The Red Room manager greets %s like an old friend and discreetly points toward a private suite. No incident report is filed about what happens next. %s returns to the street as the neon sign flickers back to life.",
	"%s steps into the Red Room as the band launches into a disastrously romantic ballad. A private appointment follows, then a quick exit past an officer from another precinct. Somehow, %s avoids an awkward conversation.",
}

func newRedRoomScene(officer string) string {
	template := redRoomScenes[rand.IntN(len(redRoomScenes))]
	return fmt.Sprintf(template, officer, officer)
}
