# Foundry: AI-Powered Podcast Generator

Foundry is an end-to-end podcast generation tool that fetches articles, scrapes contextual information, generates a two-host conversational script using LLMs, and synthesizes it into high-quality audio.

## Architecture

- **Go Orchestrator**: Handles the main pipeline, including parallel HTML fetching, text extraction, and concurrent context scraping from multiple sources. It manages the flow between stages and calls the Python service for script generation.
- **Python FastAPI Service**: A modular service that interacts with LLMs (Sarvam API) to generate natural, engaging scripts between two hosts (Host A & Host B).
- **TTS Engine**: Integrated with **Sarvam BulBul V3**, the Go orchestrator processes dialogue segments in parallel and stitches them using **ffmpeg** for ultra-fast audio production.

## Features

- [x] **Parallel Scraping**: Uses `errgroup` for fast, concurrent fetching from Reddit, News RSS, and Wikipedia.
- [x] **LLM Scripting**: Generates context-rich discussions using Sarvam's `sarvam-30b` model.
- [x] **Parallel TTS**: Concurrently synthesizes audio clips for each host, reducing end-to-end latency.
- [x] **Audio Stitching**: Seamlessly combines dialogue into a single WAV file using `ffmpeg`.

## Setup

### Prerequisites

- Go 1.25.3+
- Python 3.11+
- Conda (recommended)
- ffmpeg

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/foundry.git
   cd foundry
   ```

2. **Set up the Python Environment**:
   ```bash
   cd python
   conda create -n podify python=3.11 -y
   conda activate podify
   pip install -r requirements.txt
   ```

3. **Install Go Dependencies**:
   ```bash
   go mod tidy
   ```

### Configuration

Add your **Sarvam API Key** in the following locations:
- `python/main.py`: `SARVAM_API_KEY = "your_key_here"`
- `internal/orchestrator/tts.go`: `apiKey := "your_key_here"`

## Usage

1. **Start the Python LLM Service**:
   ```bash
   conda run -n podify python python/main.py
   ```

2. **Run the Go Orchestrator**:
   ```bash
   go run cmd/server/main.go <article-url>
   ```

The final podcast will be generated at `/tmp/final_podcast.wav`.

## License

MIT
