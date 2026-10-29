// SPDX-License-Identifier: GPL-3.0-only
package main

import (
	"fmt"
	"time"
)

const campaignDays = 30

// utcDay is a timezone-independent calendar boundary, not a rolling 24-hour
// cooldown. Every officer resets at the same midnight UTC.
func utcDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
func utcDate(t time.Time) string { return utcDay(t).Format("2006-01-02") }

func parseUTCDate(s string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", s)
	if err != nil || d.Format("2006-01-02") != s {
		return time.Time{}, fmt.Errorf("invalid UTC campaign date")
	}
	return d, nil
}

// syncCalendar is invoked by both GET and POST server endpoints, and by
// direct game actions. It does not advance game time more than once per UTC
// date and never awards wages for missed days.
//
// An existing v0.9 save without a campaign calendar is migrated on first
// access: today's date becomes its active date, remaining campaign days and
// inventory are preserved. Progress from the old instant-day game is kept.
func (g *Game) syncCalendar(now time.Time) bool {
	if g.Phase == "setup" || g.Phase == "gameover" {
		return false
	}
	today := utcDay(now)
	changed := false
	if g.CampaignStart == "" {
		daysRemaining := max(1, min(campaignDays, g.Player.Days))
		// An officer already on day 10 in an old save should keep that
		// amount of progress rather than being extended to 30 new days.
		start := today.AddDate(0, 0, -(campaignDays-daysRemaining))
		g.CampaignStart = utcDate(start)
		g.LastResetDate = utcDate(today)
		g.Player.Days = daysRemaining
		g.log(fmt.Sprintf("Calendar activated: %d real-world days remaining. Daily reset at 00:00 UTC.", daysRemaining))
		return true
	}

	start, err := parseUTCDate(g.CampaignStart)
	if err != nil {
		// Corrupt saves are rejected by validGame. Avoid mutating them here.
		return false
	}
	elapsed := int(today.Sub(start).Hours() / 24)
	if elapsed < 0 {
		return false // A clock moving backwards must never replenish turns.
	}
	if elapsed >= campaignDays {
		g.Player.Days = 0
		g.Player.Bumfights = 0
		g.ShiftEnded = true
		g.Enemy, g.Event, g.RedRoom = nil, nil, nil
		g.Phase = "gameover"
		g.EndReason = "Your 30 real-world days of duty are complete. Captain Grimstead has filed the retirement paperwork."
		g.log(g.EndReason)
		return true
	}
	// Do not move a campaign backwards if the system clock is corrected or
	// a test injects an earlier timestamp after the last processed date.
	if g.LastResetDate != "" {
		lastDate, parseErr := parseUTCDate(g.LastResetDate)
		if parseErr != nil {return false}
		if today.Before(lastDate) {return false}
	}
	wantDays := campaignDays - elapsed
	if g.Player.Days != wantDays {
		g.Player.Days = wantDays
		changed = true
	}
	if g.LastResetDate == "" {
		g.LastResetDate = utcDate(today)
		return true
	}
	last, err := parseUTCDate(g.LastResetDate)
	if err != nil {
		return changed
	}
	if !today.After(last) {
		return changed
	}

	missed := int(today.Sub(last).Hours()/24)-1
	g.LastResetDate = utcDate(today)
	g.ShiftEnded = false
	g.Phase = "hq"
	g.Enemy, g.Event, g.RedRoom = nil, nil, nil
	g.Player.ClassPower = 1
	if g.Player.clearEffects() {
		g.log("A new UTC day begins. Temporary effects have expired.")
	}
	if g.Player.Class == "psychic" {
		g.Player.refillPsychicEnergy()
	}
	g.Player.Bumfights = g.Player.shiftPatrols()
	if missed > 0 {
		g.log(fmt.Sprintf("%d missed UTC day(s) passed without turns or shift pay.", missed))
	}
	g.log(fmt.Sprintf("UTC duty day %d/%d begins: %d patrol turns available. %d campaign days remaining.",
		elapsed+1,campaignDays,g.Player.Bumfights,g.Player.Days))
	return true
}

// endShift ends today's gameplay. Neither Clock Out nor defeat may advance
// the real-world calendar; next-day resources arrive only at UTC midnight.
func (g *Game) endShift(defeated bool) {
	p := &g.Player
	g.Enemy, g.Event, g.RedRoom = nil, nil, nil
	g.Phase = "hq"
	g.ShiftEnded = true
	p.Bumfights = 0
	p.ClassPower = 1
	p.clearEffects()
	if defeated {
		p.HP = max(1, p.maxHP()/2)
		g.log(fmt.Sprintf("Forced clock-out: %s recovers to %d HP. Shift pay and unused turns forfeited. Return after midnight UTC.",p.Name,p.HP))
		return
	}
	income := p.shiftIncome()
	p.Money += income
	g.log(fmt.Sprintf("%s clocks out: +$%d shift pay. Patrol is closed until 00:00 UTC.",p.Name,income))
}
