package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Second}

// FetchHTML does a real HTTP GET and returns raw HTML.
func FetchHTML(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("build request %s: %w", url, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Foundry/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(body), nil
}

// ExtractText strips HTML tags and collapses whitespace.
// Good enough for Phase 1; swap with a proper extractor (e.g. go-readability) later.
func ExtractText(html string) string {
	// Remove <script> and <style> blocks entirely
	re := regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	clean := re.ReplaceAllString(html, " ")

	// Strip remaining tags
	re2 := regexp.MustCompile(`<[^>]+>`)
	clean = re2.ReplaceAllString(clean, " ")

	// Decode common entities
	replacer := strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&nbsp;", " ", "&#39;", "'", "&quot;", `"`,
	)
	clean = replacer.Replace(clean)

	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s+`)
	clean = wsRe.ReplaceAllString(clean, " ")

	return strings.TrimSpace(clean)
}
