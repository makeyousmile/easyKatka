package app

import "testing"

func TestTrimTo(t *testing.T) {
	// Короткая строка не должна меняться.
	if got := trimTo("abc", 5); got != "abc" {
		t.Fatalf("trimTo short got %q", got)
	}
	// Ровно в размер - без изменений.
	if got := trimTo("abcde", 5); got != "abcde" {
		t.Fatalf("trimTo exact got %q", got)
	}
	// Длинная строка должна обрезаться с многоточием.
	if got := trimTo("abcdef", 5); got != "ab..." {
		t.Fatalf("trimTo long got %q", got)
	}
}

func TestFormatDuration(t *testing.T) {
	// Базовые преобразования длительности в формат M:SS.
	cases := map[int]string{
		0:    "0:00",
		59:   "0:59",
		61:   "1:01",
		3601: "60:01",
	}
	for seconds, want := range cases {
		if got := formatDuration(seconds); got != want {
			t.Fatalf("formatDuration(%d)=%q, want %q", seconds, got, want)
		}
	}
}

func TestFallbackName(t *testing.T) {
	// Пустые и пробельные имена должны заменяться на дефолт.
	if got := fallbackName(""); got != "неизвестный" {
		t.Fatalf("fallbackName empty got %q", got)
	}
	if got := fallbackName("   "); got != "неизвестный" {
		t.Fatalf("fallbackName spaces got %q", got)
	}
	// Валидное имя должно сохраняться.
	if got := fallbackName("Dota"); got != "Dota" {
		t.Fatalf("fallbackName value got %q", got)
	}
}

func TestCalcWinrateFromMatches(t *testing.T) {
	// Проверяем расчет винрейта на массиве матчей.
	matches := []playerMatch{
		{RadiantWin: true, PlayerSlot: 0},
		{RadiantWin: false, PlayerSlot: 0},
		{RadiantWin: true, PlayerSlot: 0},
	}
	winrate, games := calcWinrateFromMatches(matches)
	if games != 3 {
		t.Fatalf("games=%d, want 3", games)
	}
	if winrate != (2.0*100.0/3.0) {
		t.Fatalf("winrate=%f, want %f", winrate, 2.0*100.0/3.0)
	}
}
