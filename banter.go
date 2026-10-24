// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
package main

import "math/rand/v2"

// Each enemy has its own ridiculous things to shout at the officer.
// Keep this dialogue silly, short and non-graphic. It appears in the
// encounter panel and dispatch transcript.
var enemyCatchphrases = map[string][]string{
	"Hostile Street Drifter": {
		"Is that a badge, or did you lose a bet?",
		"I was here first, copper!",
		"Tell dispatch I said hello!",
		"My lawyer is a vending machine!",
	},
	"Alleyway Troublemaker": {
		"You picked the wrong alley!",
		"No refunds on bad decisions!",
		"That badge doesn't impress the pigeons!",
		"Keep walking, officer. Nothing to see!",
	},
	"Unhinged Roadside Heckler": {
		"I've got more complaints than you have paperwork!",
		"Your siren's off-key!",
		"Is this a traffic stop or a fashion show?",
		"Put that in your incident report!",
	},
	"Aggressive Tent-Camp Raider": {
		"Who made you king of this street?",
		"You couldn't patrol a parking lot!",
		"I've seen scarier parking tickets!",
		"Go bother City Hall!",
	},
	"Rogue District Thug": {
		"Wrong neighbourhood, officer!",
		"My boss outranks your boss!",
		"That badge won't save your overtime!",
		"There's a toll for walking through here!",
	},
}

var genericCatchphrases = []string{
	"I demand to speak to your supervisor!",
	"I know the mayor!",
	"This is going straight into my complaint!",
}

// nextCatchphrase chooses a new line for a named enemy; when possible,
// it cannot immediately repeat the previously displayed line.
func nextCatchphrase(enemyName, previous string) string {
	lines := enemyCatchphrases[enemyName]
	if len(lines) == 0 {
		lines = genericCatchphrases
	}
	if len(lines) == 1 {
		return lines[0]
	}
	// Pick uniformly among all lines other than the current one.
	index := rand.IntN(len(lines))
	if lines[index] == previous {
		index = (index + 1 + rand.IntN(len(lines)-1)) % len(lines)
	}
	return lines[index]
}
