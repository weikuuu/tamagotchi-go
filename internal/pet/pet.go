// Package pet implements the tamagotchi's core simulation state:
// stats, time-based degradation, mood, and persistence.
package pet

import (
	"time"
)

// Stat is a bounded value in [MinStat, MaxStat]. It's a float so that
// decay accumulates smoothly across many small Tick calls (e.g. once per
// frame) instead of losing fractional progress to rounding every time.
type Stat float64

const (
	MinStat Stat = 0
	MaxStat Stat = 100
)

func (s Stat) clamp() Stat {
	if s < MinStat {
		return MinStat
	}
	if s > MaxStat {
		return MaxStat
	}
	return s
}

// Mood is derived from the pet's current stats.
type Mood int

const (
	MoodHappy Mood = iota
	MoodContent
	MoodBored
	MoodSad
	MoodHungry
	MoodTired
	MoodSick
)

func (m Mood) String() string {
	switch m {
	case MoodHappy:
		return "happy"
	case MoodContent:
		return "content"
	case MoodBored:
		return "bored"
	case MoodSad:
		return "sad"
	case MoodHungry:
		return "hungry"
	case MoodTired:
		return "tired"
	case MoodSick:
		return "sick"
	default:
		return "unknown"
	}
}

// criticalThreshold marks a stat as critically low.
const criticalThreshold = 20

// State is the tamagotchi's full simulation state.
type State struct {
	Hunger      Stat      `json:"hunger"`      // 0 = starving, 100 = full
	Energy      Stat      `json:"energy"`      // 0 = exhausted, 100 = rested
	Happiness   Stat      `json:"happiness"`   // 0 = miserable, 100 = joyful
	Cleanliness Stat      `json:"cleanliness"` // 0 = filthy, 100 = spotless
	Affection   Stat      `json:"affection"`   // 0 = stranger, 100 = inseparable; never decays, only grows
	LastTick    time.Time `json:"last_tick"`

	// LastActiveDate (YYYY-MM-DD, local) and StreakDays track consecutive
	// days with at least one interaction (feed/play/rest/pet), for the
	// "streak" flavor phrases.
	LastActiveDate string `json:"last_active_date"`
	StreakDays     int    `json:"streak_days"`
}

// NewState returns a freshly-created pet at full stats, timestamped now.
func NewState(now time.Time) State {
	return State{
		Hunger:      MaxStat,
		Energy:      MaxStat,
		Happiness:   MaxStat,
		Cleanliness: MaxStat,
		Affection:   25,
		LastTick:    now,
	}
}

const (
	hungerDecayPerHour      = 240.0     // 100 (full) -> 0 in 25 minutes
	energyDecayPerHour      = 300.0     // 100 (full) -> 0 in 20 minutes
	happinessDecayPerHour   = 480.0     // 100 (full) -> 0 in 12.5 minutes
	cleanlinessDecayPerHour = 100.0 / 8 // 100 (full) -> 0 in 8 hours
)

// Tick advances the state to now, decaying stats proportionally to elapsed
// time. Calling Tick with a now not after LastTick is a no-op, guarding
// against clock skew or replayed ticks.
func (s *State) Tick(now time.Time) {
	if !now.After(s.LastTick) {
		return
	}
	elapsedHours := now.Sub(s.LastTick).Hours()
	s.Hunger = decay(s.Hunger, hungerDecayPerHour, elapsedHours)
	s.Energy = decay(s.Energy, energyDecayPerHour, elapsedHours)
	s.Happiness = decay(s.Happiness, happinessDecayPerHour, elapsedHours)
	s.Cleanliness = decay(s.Cleanliness, cleanlinessDecayPerHour, elapsedHours)
	s.LastTick = now
}

func decay(v Stat, ratePerHour, hours float64) Stat {
	return (v - Stat(ratePerHour*hours)).clamp()
}

const (
	feedHungerGain      = 30
	playHappinessGain   = 20
	playEnergyCost      = 10
	restEnergyGain      = 40
	petHappinessGain    = 8
	washCleanlinessGain = 50

	feedAffectionGain = 1.5
	playAffectionGain = 2
	restAffectionGain = 1
	petAffectionGain  = 1
	washAffectionGain = 1

	restHungerCost      = 8 // sleeping isn't free — she still gets hungry and a little rumpled
	restCleanlinessCost = 5
)

// Feed raises Hunger, as if the pet just ate.
func (s *State) Feed() {
	s.Hunger = (s.Hunger + feedHungerGain).clamp()
	s.Affection = (s.Affection + feedAffectionGain).clamp()
}

// Play raises Happiness but costs some Energy, as if the pet just played.
func (s *State) Play() {
	s.Happiness = (s.Happiness + playHappinessGain).clamp()
	s.Energy = (s.Energy - playEnergyCost).clamp()
	s.Affection = (s.Affection + playAffectionGain).clamp()
}

// Rest raises Energy, as if the pet just slept. Hunger and Cleanliness still
// tick down while she sleeps — resting isn't a free pause on everything else.
func (s *State) Rest() {
	s.Energy = (s.Energy + restEnergyGain).clamp()
	s.Hunger = (s.Hunger - restHungerCost).clamp()
	s.Cleanliness = (s.Cleanliness - restCleanlinessCost).clamp()
	s.Affection = (s.Affection + restAffectionGain).clamp()
}

// Pet gives a small Happiness boost, as if the pet was just petted (e.g.
// clicked on the desktop overlay). It's lighter-weight than Play.
func (s *State) Pet() {
	s.Happiness = (s.Happiness + petHappinessGain).clamp()
	s.Affection = (s.Affection + petAffectionGain).clamp()
}

// Wash raises Cleanliness, as if the pet was just bathed/groomed.
func (s *State) Wash() {
	s.Cleanliness = (s.Cleanliness + washCleanlinessGain).clamp()
	s.Affection = (s.Affection + washAffectionGain).clamp()
}

const (
	miniGameHappinessPerPoint = 2.5
	miniGameAffectionPerPoint = 0.8
)

// PlayMiniGame applies the reward for a completed "catch the hearts"
// mini-game round, scaled by how many hearts were caught.
func (s *State) PlayMiniGame(score int) {
	s.Happiness = (s.Happiness + Stat(float64(score)*miniGameHappinessPerPoint)).clamp()
	s.Affection = (s.Affection + Stat(float64(score)*miniGameAffectionPerPoint)).clamp()
}

// dirtyThreshold marks Cleanliness as low enough to be "dirty".
const dirtyThreshold = 30

// IsDirty reports whether the pet needs grooming.
func (s State) IsDirty() bool {
	return s.Cleanliness < dirtyThreshold
}

// RecordActivity marks that the user interacted with the pet at now,
// updating the day-streak. It reports whether the streak count just went up
// (i.e. today is the first interaction after a full previous day of at
// least one interaction), so callers can react to it (e.g. a phrase).
func (s *State) RecordActivity(now time.Time) (streakIncreased bool) {
	today := now.Local().Format("2006-01-02")
	if s.LastActiveDate == today {
		return false
	}
	yesterday := now.Local().AddDate(0, 0, -1).Format("2006-01-02")
	if s.LastActiveDate == yesterday {
		s.StreakDays++
		streakIncreased = true
	} else {
		s.StreakDays = 1
	}
	s.LastActiveDate = today
	return streakIncreased
}

// Mood computes the pet's current mood from its stats. It is a pure
// function of the current values; callers should call Tick first if they
// want mood based on elapsed time.
func (s State) Mood() Mood {
	switch {
	case s.Hunger < criticalThreshold && s.Energy < criticalThreshold && s.Happiness < criticalThreshold:
		return MoodSick
	case s.Hunger < criticalThreshold:
		return MoodHungry
	case s.Energy < criticalThreshold:
		return MoodTired
	case s.Happiness < criticalThreshold:
		return MoodSad
	case s.Happiness < 50:
		return MoodBored
	case s.Happiness < 80:
		return MoodContent
	default:
		return MoodHappy
	}
}
