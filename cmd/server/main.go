package main

import (
	"flag"
	"fmt"
	"os"

	"foundry/internal/orchestrator"
)

func main() {
	duration := flag.String("duration", "2 mins", "podcast duration (e.g., 1 min, 2 mins)")
	style := flag.String("style", "conversational", "podcast style (e.g., conversational, entertaining)")
	language := flag.String("lang", "english", "podcast language (e.g., english, malayalam)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: foundry [flags] <article-url>")
		os.Exit(1)
	}

	articleURL := args[0]
	fmt.Printf("▶ Starting Foundry pipeline for: %s\n", articleURL)
	fmt.Printf("  Options: duration=%s, style=%s, lang=%s\n", *duration, *style, *language)

	result, err := orchestrator.Run(articleURL, *duration, *style, *language)
	if err != nil {
		fmt.Printf("\npipeline error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n── Result ─────────────────────────────")
	fmt.Printf("Article text length : %d chars\n", len(result.ArticleText))
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
