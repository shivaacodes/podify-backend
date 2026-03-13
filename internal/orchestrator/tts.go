// parallel TTS pattern (Phase 4)
package orchestrator

// This file shows the Phase 4 TTS concurrency pattern.
// Wire it in once Python generates a real script.
//
// The key problem: you have N segments, fire N goroutines,
// but must collect results IN ORDER (seg1 before seg2, etc.)
//
// Solution: indexed result slice, not a channel.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sync/errgroup"
)

// DialogueSegment is one line of the podcast script.
type DialogueSegment struct {
	Index   int
	Speaker string // "Host A" or "Host B"
	Text    string
}

// AudioSegment is the result of one TTS call.
type AudioSegment struct {
	Index    int
	FilePath string // e.g. /tmp/seg_0.wav
}

// ParseDialogue splits the raw script string into ordered segments.
func ParseDialogue(script string) []DialogueSegment {
	segments := []DialogueSegment{}
	lines := strings.Split(script, "\n")

	var currentSpeaker string
	var currentText strings.Builder

	flush := func() {
		if currentSpeaker != "" {
			text := strings.TrimSpace(currentText.String())
			if text != "" {
				segments = append(segments, DialogueSegment{
					Index:   len(segments),
					Speaker: currentSpeaker,
					Text:    text,
				})
			}
		}
		currentText.Reset()
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Host A:") {
			flush()
			currentSpeaker = "Host A"
			currentText.WriteString(strings.TrimPrefix(line, "Host A:"))
		} else if strings.HasPrefix(line, "Host B:") {
			flush()
			currentSpeaker = "Host B"
			currentText.WriteString(strings.TrimPrefix(line, "Host B:"))
		} else {
			if currentSpeaker != "" {
				currentText.WriteString(" ")
				currentText.WriteString(line)
			}
		}
	}
	flush()

	return segments
}

// GenerateAudioParallel fires one goroutine per segment,
// collects results in index order, returns ordered AudioSegments.
// GenerateAudioParallel handles parallel TTS generation.
func GenerateAudioParallel(ctx context.Context, segments []DialogueSegment, language string) ([]AudioSegment, error) {
	// Pre-allocate result slice at full size.
	// Each goroutine writes to its own index — no mutex needed.
	results := make([]AudioSegment, len(segments))

	g, ctx := errgroup.WithContext(ctx)

	for _, seg := range segments {
		seg := seg // capture
		g.Go(func() error {
			path, err := callTTS(ctx, seg, language)
			if err != nil {
				return fmt.Errorf("TTS segment %d (%s): %w", seg.Index, seg.Speaker, err)
			}
			// Safe: each goroutine writes to its own index only
			results[seg.Index] = AudioSegment{
				Index:    seg.Index,
				FilePath: path,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// Sarvam TTS Request/Response
type sarvamTTSRequest struct {
	Inputs             []string `json:"inputs"`
	TargetLanguageCode string   `json:"target_language_code"`
	Speaker            string   `json:"speaker"`
	Model              string   `json:"model"`
}

type sarvamTTSResponse struct {
	AudioResponse string `json:"audios"` // Base64 strings? API docs vary, sometimes it's an array
}

// callTTS calls Sarvam BulBul V3 for one segment.
func callTTS(ctx context.Context, seg DialogueSegment, language string) (string, error) {
	apiKey := "YOUR_SARVAM_API_KEY_HERE"
	url := "https://api.sarvam.ai/text-to-speech"

	speaker := "shubh" // Male (Conversational)
	if seg.Speaker == "Host B" {
		speaker = "manan" // Male (Consistent)
	}

	langCode := "en-IN"
	if strings.ToLower(language) == "malayalam" {
		langCode = "ml-IN"
	}

	reqBody := struct {
		Inputs             []string `json:"inputs"`
		TargetLanguageCode string   `json:"target_language_code"`
		Speaker            string   `json:"speaker"`
		Model              string   `json:"model"`
	}{
		Inputs:             []string{seg.Text},
		TargetLanguageCode: langCode,
		Speaker:            speaker,
		Model:              "bulbul:v3",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-subscription-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sarvam tts failed with status %d: %s", resp.StatusCode, string(body))
	}

	// The API returns a JSON with base64 encoded audio in an array or single field
	var apiResp struct {
		Audios []string `json:"audios"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Audios) == 0 {
		return "", fmt.Errorf("no audio returned from sarvam")
	}

	audioData, err := base64.StdEncoding.DecodeString(apiResp.Audios[0])
	if err != nil {
		return "", fmt.Errorf("decode audio: %w", err)
	}

	path := fmt.Sprintf("/tmp/seg_%d.wav", seg.Index)
	if err := os.WriteFile(path, audioData, 0644); err != nil {
		return "", err
	}

	return path, nil
}

// StitchAudio concatenates ordered audio segments using ffmpeg.
func StitchAudio(segments []AudioSegment) (string, error) {
	outputPath := "/tmp/final_podcast.wav"

	// Create a temporary file list for ffmpeg concat
	listPath := "/tmp/segments_list.txt"
	var listContent strings.Builder
	for _, seg := range segments {
		listContent.WriteString(fmt.Sprintf("file '%s'\n", seg.FilePath))
	}

	if err := os.WriteFile(listPath, []byte(listContent.String()), 0644); err != nil {
		return "", fmt.Errorf("write segment list: %w", err)
	}

	// ffmpeg -f concat -safe 0 -i /tmp/segments_list.txt -c copy /tmp/final_podcast.wav
	cmd := exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return outputPath, nil
}
