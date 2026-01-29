package app

import (
	"strings"
	"testing"
	"time"
)

func TestBuildPlayerTable_BasicOutput(t *testing.T) {
	// Фиксируем часовой пояс, чтобы время было детерминированным.
	oldLocal := time.Local
	time.Local = time.UTC
	defer func() { time.Local = oldLocal }()

	matches := []recentMatch{{
		HeroID:     1,
		Kills:      10,
		Deaths:     2,
		Assists:    3,
		Duration:   65,
		StartTime:  0,
		PlayerSlot: 0,
		RadiantWin: true,
	}}
	heroes := map[int]string{1: "Axe"}
	out := buildPlayerTable(matches, heroes, "Player")

	// Проверяем, что в выводе есть базовые элементы строки матча.
	if !strings.Contains(out, "1970-01-01 00:00") {
		t.Fatalf("expected start time in output: %q", out)
	}
	if !strings.Contains(out, "Axe") {
		t.Fatalf("expected hero name in output: %q", out)
	}
	if !strings.Contains(out, "✅") {
		t.Fatalf("expected win mark in output: %q", out)
	}
	if !strings.Contains(out, "10/2/3") {
		t.Fatalf("expected K/D/A in output: %q", out)
	}
}
