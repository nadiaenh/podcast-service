package extract

import (
	"context"
	"errors"
	"net/http"
	"podcast-service/internal/httpx"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Result struct {
	Title string
	Text  string
}

var (
	titleRe  = regexp.MustCompile(`(?is)<title[^>]*>([^<]*)</title>`)
	scriptRe = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	tagRe    = regexp.MustCompile(`<[^>]+>`)
	wsRe     = regexp.MustCompile(`\s+`)
)

var entityReplacer = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", "\"",
	"&#39;", "'",
	"&nbsp;", " ",
)

func decodeEntities(s string) string {
	return entityReplacer.Replace(s)
}

func stripHTML(html string) string {
	s := scriptRe.ReplaceAllString(html, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = decodeEntities(s)
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

func extractTitle(html, fallback string) string {
	m := titleRe.FindStringSubmatch(html)
	if m == nil {
		return fallback
	}
	return decodeEntities(strings.TrimSpace(m[1]))
}

var articleClient = httpx.NewClient(45*time.Second, true)

func fetchHTML(ctx context.Context, rawURL string) (string, error) {
	if err := httpx.ValidateURL(rawURL); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", errors.New("invalid article request")
	}
	req.Header.Set("User-Agent", "podcast-service/1.0")
	body, err := httpx.Do(articleClient, req, 4<<20, false)
	return string(body), err
}
func truncate(s string) string {
	if len(s) > 20000 {
		s = s[:20000]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return s
}
func FetchDirect(rawURL string) (Result, error) { return fetchDirect(context.Background(), rawURL) }
func fetchDirect(ctx context.Context, rawURL string) (Result, error) {
	html, err := fetchHTML(ctx, rawURL)
	if err != nil {
		return Result{}, err
	}
	return Result{Title: extractTitle(html, rawURL), Text: truncate(stripHTML(html))}, nil
}
func FetchJina(rawURL string) (Result, error) { return fetchJina(context.Background(), rawURL) }
func fetchJina(ctx context.Context, rawURL string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := httpx.ValidatePublicURL(ctx, rawURL); err != nil {
		return Result{}, err
	}
	text, err := fetchHTML(ctx, "https://r.jina.ai/"+rawURL)
	if err != nil {
		return Result{}, err
	}
	return Result{Title: rawURL, Text: truncate(strings.TrimSpace(text))}, nil
}
func Article(rawURL string) (Result, error) { return ArticleContext(context.Background(), rawURL) }
func ArticleContext(ctx context.Context, rawURL string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := httpx.ValidatePublicURL(ctx, rawURL); err != nil {
		return Result{}, err
	}
	res, err := fetchDirect(ctx, rawURL)
	if err == nil && len(res.Text) > 200 {
		return res, nil
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	jres, jerr := fetchJina(ctx, rawURL)
	if jerr == nil && len(jres.Text) > 200 {
		return jres, nil
	}
	if err == nil && strings.TrimSpace(res.Text) != "" {
		return res, nil
	}
	return Result{}, errors.New("article extraction failed; check that the URL contains accessible text")
}

func ValidateURL(raw string) error { return httpx.ValidateURL(raw) }
