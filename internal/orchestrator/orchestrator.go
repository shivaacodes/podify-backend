// main pipeline (errgroup context scraping)

package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"foundry/internal/fetcher"
	"foundry/internal/logger"

	"golang.org/x/sync/errgroup"
)

// ContextSource represents one parallel scraping worker.
type ContextSource struct {
	Name string
	URL  string
}

// PipelineResult is the final output of one run.
type PipelineResult struct {
	ArticleText string
	Summary     string
	Context     map[string]string // source name → scraped text
	Script      string
	AudioPath   string
}

// Run executes the full Phase 1 pipeline.
func Run(articleURL string, duration string, style string, language string) (*PipelineResult, error) {
	log := logger.New()
	result := &PipelineResult{}

	// ── Stage 1: Fetch & extract ──────────────────────────────────────────
	var rawHTML string
	err := log.Track("fetch_html", func() error {
		var e error
		rawHTML, e = fetcher.FetchHTML(articleURL)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("stage fetch_html: %w", err)
	}

	err = log.Track("extract_text", func() error {
		result.ArticleText = fetcher.ExtractText(rawHTML)
		if len(result.ArticleText) < 100 {
			return fmt.Errorf("extracted text too short (%d chars)", len(result.ArticleText))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stage extract_text: %w", err)
	}

	// ── Stage 2: Summarize (STUB) ─────────────────────────────────────────
	err = log.Track("summarize", func() error {
		// Phase 3: replace with real LLM call via Python service
		time.Sleep(200 * time.Millisecond) // simulate latency
		result.Summary = fakeSummary(result.ArticleText)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stage summarize: %w", err)
	}

	// ── Stage 3: Parallel context scraping (errgroup) ─────────────────────
	// This is the pattern you'll use for real scrapers too.
	// errgroup gives you: fan-out goroutines + wait + first-error cancellation.
	err = log.Track("context_scraping", func() error {
		sources := []ContextSource{
			{Name: "reddit", URL: "https://reddit.com"}, // stub URLs
			{Name: "news_rss", URL: "https://news.ycombinator.com"},
			{Name: "wikipedia", URL: "https://en.wikipedia.org"},
		}

		// mu protects the result.Context map
		result.Context = make(map[string]string, len(sources))

		// Use a buffered channel to collect results safely.
		// errgroup cancels all goroutines if any returns an error.
		type scraped struct {
			name string
			text string
		}
		ch := make(chan scraped, len(sources))

		g, ctx := errgroup.WithContext(context.Background())
		_ = ctx // will be used in Phase 3 to pass to real HTTP calls

		for _, src := range sources {
			src := src // capture loop var — important!
			g.Go(func() error {
				// STUB: Phase 3 replace with real scraping
				time.Sleep(300 * time.Millisecond)
				ch <- scraped{
					name: src.Name,
					text: fmt.Sprintf("[STUB context from %s about: %s]", src.Name, firstN(result.Summary, 50)),
				}
				return nil
			})
		}

		// Wait for all goroutines, then close channel
		if err := g.Wait(); err != nil {
			return err
		}
		close(ch)

		for s := range ch {
			result.Context[s.name] = s.text
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stage context_scraping: %w", err)
	}

	// ── Stage 4: Generate podcast script (STUB → Phase 3: Python HTTP call) ──
	err = log.Track("generate_script", func() error {
		script, err := callPythonScriptService(result.Summary, result.Context, duration, style, language)
		if err != nil {
			return err
		}
		result.Script = script
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stage generate_script: %w", err)
	}

	// ── Stage 5: TTS audio (STUB → Phase 4: parallel Sarvam BulBul calls) ──
	err = log.Track("tts_and_stitch", func() error {
		// 1. Parse script into segments
		segments := ParseDialogue(result.Script)
		if len(segments) == 0 {
			return fmt.Errorf("no dialogue segments parsed from script")
		}

		// 2. Generate audio segments in parallel
		audioSegments, err := GenerateAudioParallel(context.Background(), segments, language)
		if err != nil {
			return fmt.Errorf("generate parallel audio: %w", err)
		}

		// 3. Stitch audio segments into final podcast
		voicePath, err := StitchAudio(audioSegments)
		if err != nil {
			return fmt.Errorf("stitch audio: %w", err)
		}

		// 4. Mix with Background Music (if available)
		bgmPath := "assets/bgm.mp3"
		if _, err := os.Stat(bgmPath); err == nil {
			finalPath, err := MixBGM(voicePath, bgmPath)
			if err != nil {
				return fmt.Errorf("mix bgm: %w", err)
			}
			result.AudioPath = finalPath
		} else {
			result.AudioPath = voicePath
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("stage tts_and_stitch: %w", err)
	}

	log.Print()
	return result, nil
}

// ── Service Integration ──────────────────────────────────────────────────────

type ScriptRequest struct {
	Summary  string            `json:"summary"`
	Context  map[string]string `json:"context"`
	Duration string            `json:"duration"`
	Style    string            `json:"style"`
	Language string            `json:"language"`
}

type ScriptResponse struct {
	Script string `json:"script"`
}

func callPythonScriptService(summary string, context map[string]string, duration string, style string, language string) (string, error) {
	url := "http://localhost:8000/generate-script"

	reqBody, err := json.Marshal(ScriptRequest{
		Summary:  summary,
		Context:  context,
		Duration: duration,
		Style:    style,
		Language: language,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create a client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second, // Increased timeout for real LLM
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("http post to python service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("python service returned status: %d", resp.StatusCode)
	}

	var scriptResp ScriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&scriptResp); err != nil {
		return "", fmt.Errorf("decode python response: %w", err)
	}

	return scriptResp.Script, nil
}

// ── Stub helpers ─────────────────────────────────────────────────────────────

func fakeSummary(text string) string {
	preview := firstN(text, 200)
	return fmt.Sprintf("[STUB SUMMARY] Article starts with: %s...", preview)
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
