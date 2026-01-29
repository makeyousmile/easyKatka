package app

import "testing"

func TestCalcWinrateWithCount_UsesLimit(t *testing.T) {
	matches := []recentMatch{}
	for i := 0; i < 60; i++ {
		matches = append(matches, recentMatch{RadiantWin: i%2 == 0, PlayerSlot: 0})
	}

	winrate, games := calcWinrateWithCount(matches, 50)
	if games != 50 {
		t.Fatalf("games=%d, want 50", games)
	}
	if winrate != 50 {
		t.Fatalf("winrate=%.1f, want 50.0", winrate)
	}
}

func TestCalcWinrateWithCount_LimitGreaterThanMatches(t *testing.T) {
	matches := []recentMatch{{RadiantWin: true, PlayerSlot: 0}, {RadiantWin: false, PlayerSlot: 0}, {RadiantWin: true, PlayerSlot: 0}}
	winrate, games := calcWinrateWithCount(matches, 50)
	if games != 3 {
		t.Fatalf("games=%d, want 3", games)
	}
	if winrate != (2.0*100.0/3.0) {
		t.Fatalf("winrate=%.6f, want %f", winrate, 2.0*100.0/3.0)
	}
}

func TestCalcWinrateWithCount_ZeroLimit(t *testing.T) {
	matches := []recentMatch{{RadiantWin: true, PlayerSlot: 0}}
	winrate, games := calcWinrateWithCount(matches, 0)
	if games != 0 || winrate != 0 {
		t.Fatalf("games=%d winrate=%f, want 0/0", games, winrate)
	}
}

func TestMatchWin_PlayerSlot(t *testing.T) {
	if !matchWin(recentMatch{RadiantWin: true, PlayerSlot: 0}) {
		t.Fatal("radiant player should win when RadiantWin=true")
	}
	if matchWin(recentMatch{RadiantWin: true, PlayerSlot: 128}) {
		t.Fatal("dire player should lose when RadiantWin=true")
	}
}
