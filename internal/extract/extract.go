package extract

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"podcast-service/internal/httpx"
)

type Result struct {
	Title string
	Text  string
}

const (
	maxTextChars = 20000
	minTextChars = 200
)

var (
	titleRe  = regexp.MustCompile(`(?is)<title[^>]*>([^<]*)</title>`)
	scriptRe = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	tagRe    = regexp.MustCompile(`<[^>]+>`)
	wsRe     = regexp.MustCompile(`\s+`)

	entities = strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", `"`, "&#39;", "'", "&nbsp;", " ",
	)

	articleClient = httpx.NewClient(45*time.Second, true)
)

func Article(ctx context.Context, url string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := httpx.ValidatePublicURL(ctx, url); err != nil {
		return Result{}, err
	}

	direct, err := fetchDirect(ctx, url)
	if err == nil && len(direct.Text) > minTextChars {
		return direct, nil
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if reader, rerr := fetchViaReader(ctx, url); rerr == nil && len(reader.Text) > minTextChars {
		return reader, nil
	}
	if err == nil && strings.TrimSpace(direct.Text) != "" {
		return direct, nil
	}
	return Result{}, errors.New("article extraction failed; check that the URL contains accessible text")
}

func fetchDirect(ctx context.Context, url string) (Result, error) {
	html, err := fetchHTML(ctx, url)
	if err != nil {
		return Result{}, err
	}
	title := url
	if m := titleRe.FindStringSubmatch(html); m != nil {
		title = entities.Replace(strings.TrimSpace(m[1]))
	}
	return Result{Title: title, Text: clip(plainText(html))}, nil
}

func fetchViaReader(ctx context.Context, url string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := httpx.ValidatePublicURL(ctx, url); err != nil {
		return Result{}, err
	}
	text, err := fetchHTML(ctx, "https://r.jina.ai/"+url)
	if err != nil {
		return Result{}, err
	}
	return Result{Title: url, Text: clip(strings.TrimSpace(text))}, nil
}

func fetchHTML(ctx context.Context, url string) (string, error) {
	if err := httpx.ValidateURL(url); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.New("invalid article request")
	}
	req.Header.Set("User-Agent", "podcast-service/1.0")
	body, err := httpx.Do(articleClient, req, 4<<20, false)
	return string(body), err
}

func plainText(html string) string {
	s := scriptRe.ReplaceAllString(html, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = entities.Replace(s)
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

func clip(s string) string {
	if len(s) <= maxTextChars {
		return s
	}
	s = s[:maxTextChars]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
