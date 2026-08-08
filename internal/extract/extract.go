package extract

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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

func fetchHTML(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch failed: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func FetchDirect(url string) (Result, error) {
	html, err := fetchHTML(url)
	if err != nil {
		return Result{}, err
	}
	title := extractTitle(html, url)
	text := stripHTML(html)
	if len(text) > 20000 {
		text = text[:20000]
	}
	return Result{Title: title, Text: text}, nil
}

func FetchJina(url string) (Result, error) {
	text, err := fetchHTML("https://r.jina.ai/" + url)
	if err != nil {
		return Result{}, err
	}
	text = strings.TrimSpace(text)
	if len(text) > 20000 {
		text = text[:20000]
	}
	return Result{Title: url, Text: text}, nil
}

func Article(url string) (Result, error) {
	res, err := FetchDirect(url)
	if err == nil && len(res.Text) > 200 {
		return res, nil
	}
	jres, jerr := FetchJina(url)
	if jerr == nil && len(jres.Text) > 0 {
		return jres, nil
	}
	if err != nil {
		return Result{}, err
	}
	return res, nil
}
