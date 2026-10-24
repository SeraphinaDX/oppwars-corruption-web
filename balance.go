// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
// Balance rules for fictional, temporary contraband status effects.
package main

import "fmt"

const (
	baseMaxHP          = 200
	hpPerLevel         = 40
	maxOfficerHP       = 720
	maxClassPower      = 4
	maxPsychicEnergy   = 300
	psychicBlastCost   = 25 // 12 double-damage blasts per shift.
	psychicRedRoomRestore = 150 // Six additional blasts per paid Red Room visit, capped at 12.
	legacyPsychicEnergyMax = 100 // Meter size in saves from releases before v0.8.0.
	paragonAmbushRank = 100      // Formerly rank 10, now requires about four full shifts.
	renegadeShopRank = 100      // Same threshold for the 2-for-1 Property Room perk.
	maxInventoryEach   = 20
	maxClubShares      = 5
	// Hand cuffs allow a controlled arrest after an enemy is weakened.
	handcuffsPrice     = 100
	handcuffsPerPurchase = 1
	arrestHPPercent    = 50
	// Arrests grant half of the usual patrol cash and XP, plus one Paragon.
	// Eliminations retain full rewards, but add one Renegade.
	arrestRewardDivisor = 2
	combatKarmaGain     = 1
	// Red Room / Medical Wing economy: condoms cost $50 each in a five-pack.
	condomPackPrice    = 250
	condomsPerPack     = 5
	infectionClinicFee = 500 // Per infection; clinic cures all at once.
	infectionDamageLossPercent = 15 // Per infection, capped at 45%.
	infectionIncomeLoss = 150 // Per infection, paid each shift until cured.
	minShiftIncome      = 100 // Prevent financial softlocks.
	minInfectedPatrols  = 5   // Still enough patrols to earn a cure.
	infectionPatrolLoss = 3   // On top of the existing 15-patrol illness penalty.
	// Property Room prices (cash for one unit; Renegade purchases double units).
	cocainePrice      = 400
	fentanylPrice     = 150
	lsdPrice          = 75
	lsdPurchaseUnits  = 1
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

// Restore a limited amount of PSI without exceeding the daily meter cap.
// The returned amount is stored in the encounter receipt for the player.
func (p *Player) restorePsychicEnergy(amount int) int {
	if p.Class != "psychic" || amount <= 0 {
		return 0
	}
	before := p.psychicLeft()
	after := min(maxPsychicEnergy, before+amount)
	p.PsychicEnergy = &after
	return after-before
}

// Earlier releases stored PSI on a 0–100 scale. Preserve the remaining
// *fraction* of a saved meter when moving to the 0–300 scale, including
// exactly zero (a depleted meter must stay depleted after reloading).
func (p *Player) migratePsychicEnergy(savedVersion string) {
	if p.Class != "psychic" || p.PsychicEnergy == nil {
		return
	}
	var major, minor, patch int
	if savedVersion != "" {
		n, err := fmt.Sscanf(savedVersion, "%d.%d.%d", &major, &minor, &patch)
		if err != nil || n != 3 || major != 0 || minor >= 8 {
			return
		}
	}
	energy := max(0, min(legacyPsychicEnergyMax, *p.PsychicEnergy))*maxPsychicEnergy/legacyPsychicEnergyMax
	p.PsychicEnergy = &energy
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

// Infection weakens both normal attacks and class specials. The penalty
// stacks only to 45%, so ill officers can still earn treatment money.
func (p *Player) infectionAdjustedDamage(damage int) int {
	if damage <= 0 {
		return 0
	}
	loss := min(45, max(0, p.Infections)*infectionDamageLossPercent)
	return max(1, damage*(100-loss)/100)
}

func (p *Player) normalDamage() int {
	base := p.Attack * p.Level
	if p.FentanylEncounters > 0 {
		base += base/4 // +25%, not inventory multiplied.
	}
	return p.infectionAdjustedDamage(base)
}

// A shift begins with fewer patrol assignments while infected. Floor the
// capacity to avoid a dead-end when an officer cannot afford the doctor.
func (p *Player) shiftPatrols() int {
	if p.Infections <= 0 {
		return fightsPerDay
	}
	return max(minInfectedPatrols, fightsPerDay - 15 - p.Infections*infectionPatrolLoss)
}
func (p *Player) treatmentCost() int {
	return max(0, p.Infections)*infectionClinicFee
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
	return max(minShiftIncome, 500+min(p.ClubShares, maxClubShares)*500-max(0,p.Infections)*infectionIncomeLoss)
}

// Arrest can only succeed while the target is alive and at or below 50% HP.
func canArrestEnemy(e *Encounter) bool {
	return e != nil && e.MaxHP > 0 && e.HP > 0 && e.HP*100 <= e.MaxHP*arrestHPPercent
}

func (p *Player) victoryPay() int {
	return victoryBasePay + max(0, p.Level-1)*victoryPayPerLevel
}
