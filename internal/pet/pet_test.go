package pet

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	s := NewState(now)

	if s.Hunger != MaxStat || s.Energy != MaxStat || s.Happiness != MaxStat || s.Cleanliness != MaxStat {
		t.Fatalf("expected all stats at MaxStat, got hunger=%v energy=%v happiness=%v cleanliness=%v", s.Hunger, s.Energy, s.Happiness, s.Cleanliness)
	}
	if !s.LastTick.Equal(now) {
		t.Fatalf("expected LastTick=%v, got %v", now, s.LastTick)
	}
}

func TestTick_Degradation(t *testing.T) {
	start := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	s := NewState(start)

	elapsed := 3 * time.Hour
	now := start.Add(elapsed)
	s.Tick(now)

	wantHunger := (MaxStat - Stat(hungerDecayPerHour*3)).clamp()
	wantEnergy := (MaxStat - Stat(energyDecayPerHour*3)).clamp()
	wantHappiness := (MaxStat - Stat(happinessDecayPerHour*3)).clamp()
	wantCleanliness := (MaxStat - Stat(cleanlinessDecayPerHour*3)).clamp()

	if s.Hunger != wantHunger {
		t.Errorf("Hunger = %v, want %v", s.Hunger, wantHunger)
	}
	if s.Energy != wantEnergy {
		t.Errorf("Energy = %v, want %v", s.Energy, wantEnergy)
	}
	if s.Happiness != wantHappiness {
		t.Errorf("Happiness = %v, want %v", s.Happiness, wantHappiness)
	}
	if s.Cleanliness != wantCleanliness {
		t.Errorf("Cleanliness = %v, want %v", s.Cleanliness, wantCleanliness)
	}
	if !s.LastTick.Equal(now) {
		t.Errorf("LastTick = %v, want %v", s.LastTick, now)
	}
}

func TestTick_Clamping(t *testing.T) {
	start := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	s := NewState(start)

	// A huge elapsed duration should drive every stat to MinStat, not negative.
	s.Tick(start.Add(1000 * time.Hour))

	if s.Hunger != MinStat {
		t.Errorf("Hunger = %v, want %v", s.Hunger, MinStat)
	}
	if s.Energy != MinStat {
		t.Errorf("Energy = %v, want %v", s.Energy, MinStat)
	}
	if s.Happiness != MinStat {
		t.Errorf("Happiness = %v, want %v", s.Happiness, MinStat)
	}
	if s.Cleanliness != MinStat {
		t.Errorf("Cleanliness = %v, want %v", s.Cleanliness, MinStat)
	}
}

func TestTick_NoOpWhenNotAfter(t *testing.T) {
	start := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	s := NewState(start)

	cases := []time.Time{
		start,
		start.Add(-time.Hour),
	}
	for _, now := range cases {
		before := s
		s.Tick(now)
		if s != before {
			t.Errorf("Tick(%v) mutated state: before=%+v after=%+v", now, before, s)
		}
	}
}

func TestMood(t *testing.T) {
	tests := []struct {
		name      string
		hunger    Stat
		energy    Stat
		happiness Stat
		want      Mood
	}{
		{"all max is happy", 100, 100, 100, MoodHappy},
		{"happiness just below 80 is content", 100, 100, 79, MoodContent},
		{"happiness just below 50 is bored", 100, 100, 49, MoodBored},
		{"happiness critical is sad", 100, 100, 19, MoodSad},
		{"hunger critical is hungry", 19, 100, 100, MoodHungry},
		{"energy critical is tired", 100, 19, 100, MoodTired},
		{"all critical is sick", 10, 10, 10, MoodSick},
		{"boundary at threshold is not critical", 20, 20, 100, MoodHappy},
		{"boundary at 80 happiness is happy", 100, 100, 80, MoodHappy},
		{"boundary at 50 happiness is content", 100, 100, 50, MoodContent},
		{"hunger takes priority over energy", 19, 19, 100, MoodHungry},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := State{Hunger: tc.hunger, Energy: tc.energy, Happiness: tc.happiness}
			if got := s.Mood(); got != tc.want {
				t.Errorf("Mood() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFeed(t *testing.T) {
	s := State{Hunger: 50, Energy: 50, Happiness: 50}
	s.Feed()
	if want := Stat(50 + feedHungerGain); s.Hunger != want {
		t.Errorf("Hunger = %v, want %v", s.Hunger, want)
	}
	if s.Energy != 50 || s.Happiness != 50 {
		t.Errorf("Feed changed other stats: %+v", s)
	}

	s.Hunger = MaxStat
	s.Feed()
	if s.Hunger != MaxStat {
		t.Errorf("Feed at MaxStat = %v, want clamped %v", s.Hunger, MaxStat)
	}
}

func TestPlay(t *testing.T) {
	s := State{Hunger: 50, Energy: 50, Happiness: 50}
	s.Play()
	if want := Stat(50 + playHappinessGain); s.Happiness != want {
		t.Errorf("Happiness = %v, want %v", s.Happiness, want)
	}
	if want := Stat(50 - playEnergyCost); s.Energy != want {
		t.Errorf("Energy = %v, want %v", s.Energy, want)
	}

	s.Energy = MinStat
	s.Play()
	if s.Energy != MinStat {
		t.Errorf("Play at MinStat energy = %v, want clamped %v", s.Energy, MinStat)
	}
}

func TestRest(t *testing.T) {
	s := State{Hunger: 50, Energy: 50, Happiness: 50}
	s.Rest()
	if want := Stat(50 + restEnergyGain); s.Energy != want {
		t.Errorf("Energy = %v, want %v", s.Energy, want)
	}

	s.Energy = MaxStat
	s.Rest()
	if s.Energy != MaxStat {
		t.Errorf("Rest at MaxStat = %v, want clamped %v", s.Energy, MaxStat)
	}
}

func TestPet(t *testing.T) {
	s := State{Hunger: 50, Energy: 50, Happiness: 50}
	s.Pet()
	if want := Stat(50 + petHappinessGain); s.Happiness != want {
		t.Errorf("Happiness = %v, want %v", s.Happiness, want)
	}

	s.Happiness = MaxStat
	s.Pet()
	if s.Happiness != MaxStat {
		t.Errorf("Pet at MaxStat = %v, want clamped %v", s.Happiness, MaxStat)
	}
}

func TestWash(t *testing.T) {
	s := State{Cleanliness: 30}
	s.Wash()
	if want := Stat(30 + washCleanlinessGain); s.Cleanliness != want {
		t.Errorf("Cleanliness = %v, want %v", s.Cleanliness, want)
	}

	s.Cleanliness = MaxStat
	s.Wash()
	if s.Cleanliness != MaxStat {
		t.Errorf("Wash at MaxStat = %v, want clamped %v", s.Cleanliness, MaxStat)
	}
}

func TestIsDirty(t *testing.T) {
	tests := []struct {
		cleanliness Stat
		want        bool
	}{
		{100, false},
		{dirtyThreshold, false},
		{dirtyThreshold - 1, true},
		{0, true},
	}
	for _, tc := range tests {
		s := State{Cleanliness: tc.cleanliness}
		if got := s.IsDirty(); got != tc.want {
			t.Errorf("IsDirty() at cleanliness=%v = %v, want %v", tc.cleanliness, got, tc.want)
		}
	}
}

func TestRecordActivity(t *testing.T) {
	day1 := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	s := State{}

	if inc := s.RecordActivity(day1); inc {
		t.Error("first-ever activity should not count as a streak increase")
	}
	if s.StreakDays != 1 {
		t.Errorf("StreakDays = %d, want 1", s.StreakDays)
	}

	// Same day again: no change.
	if inc := s.RecordActivity(day1.Add(2 * time.Hour)); inc {
		t.Error("same-day activity should not increase the streak")
	}
	if s.StreakDays != 1 {
		t.Errorf("StreakDays = %d, want 1", s.StreakDays)
	}

	// Next day: streak grows.
	day2 := day1.AddDate(0, 0, 1)
	if inc := s.RecordActivity(day2); !inc {
		t.Error("next-day activity should increase the streak")
	}
	if s.StreakDays != 2 {
		t.Errorf("StreakDays = %d, want 2", s.StreakDays)
	}

	// A gap of more than a day resets the streak.
	dayAfterGap := day2.AddDate(0, 0, 3)
	if inc := s.RecordActivity(dayAfterGap); inc {
		t.Error("activity after a gap should not report a streak increase")
	}
	if s.StreakDays != 1 {
		t.Errorf("StreakDays after gap = %d, want 1", s.StreakDays)
	}
}

func TestMood_String(t *testing.T) {
	tests := []struct {
		m    Mood
		want string
	}{
		{MoodHappy, "happy"},
		{MoodContent, "content"},
		{MoodBored, "bored"},
		{MoodSad, "sad"},
		{MoodHungry, "hungry"},
		{MoodTired, "tired"},
		{MoodSick, "sick"},
		{Mood(99), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.m.String(); got != tc.want {
			t.Errorf("Mood(%v).String() = %q, want %q", tc.m, got, tc.want)
		}
	}
}
