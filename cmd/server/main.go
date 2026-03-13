package main

import (
	"fmt"
	"os"

	"foundry/internal/orchestrator"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: foundry <article-url>")
		fmt.Fprintln(os.Stderr, "example: foundry https://en.wikipedia.org/wiki/Podcast")
		os.Exit(1)
	}

	url := os.Args[1]
	fmt.Printf("▶ Starting Foundry pipeline for: %s\n\n", url)

	result, err := orchestrator.Run(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipeline error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n── Result ─────────────────────────────\n")
	fmt.Printf("Article text length : %d chars\n", len(result.ArticleText))
	fmt.Printf("Summary             : %s\n", result.Summary[:min(len(result.Summary), 120)])
	fmt.Printf("Context sources     : %d\n", len(result.Context))
	fmt.Printf("Script preview      :\n%s\n", result.Script)
	fmt.Printf("Audio output path   : %s\n", result.AudioPath)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
