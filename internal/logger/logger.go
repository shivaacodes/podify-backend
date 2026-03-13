// per-stage latency tracker
package logger

import (
	"fmt"
	"time"
)

type Stage struct {
	Name     string
	Duration time.Duration
}

type PipelineLogger struct {
	stages []Stage
	start  time.Time
}

func New() *PipelineLogger {
	return &PipelineLogger{start: time.Now()}
}

// Track wraps a function call and records how long it took.
// Usage: result, err = logger.Track("fetch_html", func() (T, error) { ... })
func (l *PipelineLogger) Track(name string, fn func() error) error {
	t := time.Now()
	err := fn()
	l.stages = append(l.stages, Stage{Name: name, Duration: time.Since(t)})
	return err
}

func (l *PipelineLogger) Print() {
	fmt.Println("\n── Pipeline Latency ──────────────────")
	for _, s := range l.stages {
		bar := ""
		ms := s.Duration.Milliseconds()
		for i := int64(0); i < ms/50; i++ { // 1 char per 50ms
			bar += "█"
		}
		fmt.Printf("  %-25s %6dms  %s\n", s.Name, ms, bar)
	}
	fmt.Printf("  %-25s %6dms\n", "TOTAL", time.Since(l.start).Milliseconds())
	fmt.Println("──────────────────────────────────────")
}
