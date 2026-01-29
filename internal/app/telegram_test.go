package app

import (
	"strings"
	"testing"
)

func TestEscapeHTML(t *testing.T) {
	// Проверяем экранирование специальных символов.
	in := "a&b<c>d"
	want := "a&amp;b&lt;c&gt;d"
	if got := escapeHTML(in); got != want {
		t.Fatalf("escapeHTML got %q, want %q", got, want)
	}
}

func TestSplitText(t *testing.T) {
	// Разбиваем строку по максимальной длине.
	parts := splitText("abcdef", 2)
	if len(parts) != 3 {
		t.Fatalf("parts=%d, want 3", len(parts))
	}
	if parts[0] != "ab" || parts[1] != "cd" || parts[2] != "ef" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}

func TestBuildTelegramMessages_Short(t *testing.T) {
	// Короткий текст должен уложиться в одно сообщение.
	header := "HEAD\n"
	body := "line1\nline2"
	msgs := buildTelegramMessages(body, header)
	if len(msgs) != 1 {
		t.Fatalf("len=%d, want 1", len(msgs))
	}
	if !strings.HasPrefix(msgs[0], header+"<pre>") {
		t.Fatalf("message missing header and <pre>")
	}
}

func TestBuildTelegramMessages_Long(t *testing.T) {
	// Длинный текст должен быть разбит на несколько сообщений.
	header := "HEAD\n"
	body := strings.Repeat("a", telegramMaxLen+100)
	msgs := buildTelegramMessages(body, header)
	if len(msgs) < 2 {
		t.Fatalf("len=%d, want >= 2", len(msgs))
	}
	if !strings.HasPrefix(msgs[0], header+"<pre>") {
		t.Fatalf("first message missing header and <pre>")
	}
}
