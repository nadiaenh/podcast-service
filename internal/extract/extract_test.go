package extract

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRejectPrivateArticlesBeforeFallback(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1/secret", "http://[::1]/secret", "http://user:password@example.com"} {
		if _, e := Article(context.Background(), u); e == nil {
			t.Fatalf("accepted %s", u)
		}
		if _, e := fetchViaReader(context.Background(), u); e == nil {
			t.Fatalf("reader accepted %s", u)
		}
	}
}

func TestClipKeepsUTF8(t *testing.T) {
	s := clip(strings.Repeat("あ", 8000))
	if len(s) > maxTextChars || !utf8.ValidString(s) {
		t.Fatal("invalid text boundary")
	}
}
