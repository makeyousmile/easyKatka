package app

import "testing"

func TestIsStatCommand(t *testing.T) {
	// Проверяем допустимые формы команды /stat.
	cases := []string{"/stat", "/stat@bot", "/stat 123"}
	for _, input := range cases {
		if !isStatCommand(input) {
			t.Fatalf("expected true for %q", input)
		}
	}
	// Проверяем, что пустая строка не считается командой.
	if isStatCommand("") {
		t.Fatal("expected false for empty input")
	}
}

func TestIsRatingCommand(t *testing.T) {
	// Проверяем допустимые формы команды /rating.
	cases := []string{"/rating", "/rating@bot", "/rating 50"}
	for _, input := range cases {
		if !isRatingCommand(input) {
			t.Fatalf("expected true for %q", input)
		}
	}
	// Проверяем, что пустая строка не считается командой.
	if isRatingCommand("") {
		t.Fatal("expected false for empty input")
	}
}

func TestParseFriendsCommand_Default(t *testing.T) {
	// Без аргументов лимит должен быть 20.
	ok, limit, err := parseFriendsCommand("/friends")
	if !ok || err != nil {
		t.Fatalf("expected ok without error, got ok=%v err=%v", ok, err)
	}
	if limit != 20 {
		t.Fatalf("limit=%d, want 20", limit)
	}
}

func TestParseFriendsCommand_WithLimit(t *testing.T) {
	// С числом лимит должен парситься.
	ok, limit, err := parseFriendsCommand("/friends 50")
	if !ok || err != nil {
		t.Fatalf("expected ok without error, got ok=%v err=%v", ok, err)
	}
	if limit != 50 {
		t.Fatalf("limit=%d, want 50", limit)
	}
}

func TestParseFriendsCommand_Invalid(t *testing.T) {
	// Некорректный аргумент должен давать ошибку.
	ok, _, err := parseFriendsCommand("/friends abc")
	if !ok {
		t.Fatal("expected ok=true for /friends command")
	}
	if err == nil {
		t.Fatal("expected error for invalid limit")
	}
}

func TestParseFriendsCommand_NotCommand(t *testing.T) {
	// Для других команд parseFriendsCommand должен вернуть ok=false.
	ok, _, err := parseFriendsCommand("/stat")
	if ok || err != nil {
		t.Fatalf("expected ok=false without error, got ok=%v err=%v", ok, err)
	}
}

func TestIsChatIDCommand(t *testing.T) {
	// Проверяем допустимые формы команды /chatid.
	cases := []string{"/chatid", "/chatid@bot", "/chatid 1"}
	for _, input := range cases {
		if !isChatIDCommand(input) {
			t.Fatalf("expected true for %q", input)
		}
	}
	// Проверяем, что пустая строка не считается командой.
	if isChatIDCommand("") {
		t.Fatal("expected false for empty input")
	}
}
