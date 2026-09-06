package extract

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRejectPrivateArticlesBeforeFallback(t *testing.T) {
	for _, u := range []string{"http://127.0.0.1/secret", "http://[::1]/secret", "http://user:password@example.com"} {
		if _, e := ArticleContext(context.Background(), u); e == nil {
			t.Fatalf("accepted %s", u)
		}
		if _, e := FetchJina(u); e == nil {
			t.Fatalf("reader accepted %s", u)
		}
	}
}
func TestTruncateKeepsUTF8(t *testing.T) {
	s := truncate(strings.Repeat("あ", 8000))
	if len(s) > 20000 || !utf8.ValidString(s) {
		t.Fatal("invalid text boundary")
	}
}
