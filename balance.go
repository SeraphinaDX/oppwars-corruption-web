// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
// Balance rules for fictional, temporary contraband status effects.
package main

const (
	baseMaxHP          = 200
	hpPerLevel         = 40
	maxOfficerHP       = 720
	maxClassPower      = 4
	maxPsychicEnergy   = 100
	psychicBlastCost   = 25 // Exactly four double-damage blasts per shift
	maxInventoryEach   = 20
	maxClubShares      = 5
	cocaineEncounters  = 3
	fentanylEncounters = 2
	lsdEncounters      = 2
	cocaineBonusCap    = 150
	victoryBasePay     = 300
	victoryPayPerLevel = 75
)

// Psychic energy is a fixed daily resource: no passive regeneration in combat.
// Keeping the field optional distinguishes old saves (nil) from a depleted meter
// (pointer to zero), so reloading does not refill exhausted characters.
func (p *Player) psychicLeft() int {
	if p.Class != "psychic" {
		return 0
	}
	if p.PsychicEnergy == nil {
		return maxPsychicEnergy // Legacy save with no meter yet.
	}
	return max(0, min(maxPsychicEnergy, *p.PsychicEnergy))
}
func (p *Player) refillPsychicEnergy() {
	value := maxPsychicEnergy
	p.PsychicEnergy = &value
}
func (p *Player) spendPsychicEnergy() bool {
	if p.Class != "psychic" || p.psychicLeft() < psychicBlastCost {
		return false
	}
	value := p.psychicLeft() - psychicBlastCost
	p.PsychicEnergy = &value
	return true
}
func (p *Player) normalizeLegacyClassResources() {
	if p.Class != "psychic" {
		return
	}
	value := p.psychicLeft()
	p.PsychicEnergy = &value
}

// Effects measure completed encounters, not button clicks. Only one kind of
// effect can be active. This deliberately disallows stacking different drugs.
func (p *Player) effectActive() bool {
	return p.CocaineEncounters > 0 || p.FentanylEncounters > 0 || p.LSDEncounters > 0
}

func (p *Player) maxHP() int {
	return min(maxOfficerHP, baseMaxHP+max(0, p.Level-1)*hpPerLevel)
}

func (p *Player) restoreHP(amount int) {
	p.HP = min(p.maxHP(), p.HP+amount)
}

func (p *Player) normalDamage() int {
	base := p.Attack * p.Level
	if p.FentanylEncounters > 0 {
		return base + base/4 // +25%, not inventory multiplied.
	}
	return base
}

func (p *Player) counterDamage() int {
	if p.FentanylEncounters > 0 {
		return 25 // In exchange for extra outgoing damage, retaliation hurts.
	}
	return 20
}

func (p *Player) earnedXP() (base, bonus int) {
	base = p.Level * 200
	if p.CocaineEncounters > 0 {
		bonus = min(cocaineBonusCap, base/4)
	}
	return base, bonus
}

// An LSD encounter is uncertain: its "foresight" can prevent retaliation,
// but distorted aim can also halve outgoing attacks, including specials.
// Rolls are parameters so both mechanics can be verified deterministically.
func distortedDamage(damage, roll int) (int, bool) {
	if roll < 20 {
		return max(1, damage/2), true
	}
	return damage, false
}

func foresightDodge(roll int) bool { return roll < 25 }

func (p *Player) finishEncounter() []string {
	var expired []string
	if p.CocaineEncounters > 0 {
		p.CocaineEncounters--
		if p.CocaineEncounters == 0 {
			expired = append(expired, "XP rush wore off")
		}
	}
	if p.FentanylEncounters > 0 {
		p.FentanylEncounters--
		if p.FentanylEncounters == 0 {
			expired = append(expired, "Reckless strength wore off")
		}
	}
	if p.LSDEncounters > 0 {
		p.LSDEncounters--
		if p.LSDEncounters == 0 {
			expired = append(expired, "Reality distortion faded")
		}
	}
	return expired
}

func (p *Player) clearEffects() bool {
	active := p.effectActive()
	p.CocaineEncounters, p.FentanylEncounters, p.LSDEncounters = 0, 0, 0
	return active
}

func (p *Player) shiftIncome() int {
	return 500 + min(p.ClubShares, maxClubShares)*500 - p.Infections*25
}

func (p *Player) victoryPay() int {
	return victoryBasePay + max(0, p.Level-1)*victoryPayPerLevel
}
